package backend

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	// maxCapturedBodyBytes caps how much of any single request/response body is read into memory and
	// retained in the HAR; the untouched remainder is streamed on to the real consumer rather than
	// buffered (see readAndRestoreBody), so capture never holds more than this per body.
	// maxCapturedTotalBytes caps the total retained across all entries in one request. Together they
	// stop a heavy or high-traffic datasource -- exactly what an operator is diagnosing -- from
	// ballooning plugin memory or pushing the __har__ response frame past the plugin<->core gRPC
	// message size limit.
	maxCapturedBodyBytes  = 8 << 20  // 8 MiB
	maxCapturedTotalBytes = 32 << 20 // 32 MiB
)

// sdkHARCaptureBuffer collects HTTP request/response pairs in HAR 1.2 format in memory.
// Used by the SDK HAR capture middleware to accumulate traffic from external plugin HTTP clients.
type sdkHARCaptureBuffer struct {
	mu       sync.Mutex
	entries  []sdkHAREntry
	retained int64 // running total of retained body text bytes, for the total-size cap
}

func newSDKHARCaptureBuffer() *sdkHARCaptureBuffer {
	return &sdkHARCaptureBuffer{}
}

func (b *sdkHARCaptureBuffer) addEntry(req *http.Request, reqBody []byte, resp *http.Response, rtErr error, started time.Time, elapsed time.Duration) {
	entry := buildSDKHAREntry(req, reqBody, resp, rtErr, started, elapsed)
	b.mu.Lock()
	defer b.mu.Unlock()
	// Enforce the cumulative retained-body budget: once the request's captured bodies exceed
	// maxCapturedTotalBytes, keep the entry's metadata (headers, sizes, timings) but drop its body
	// text so the __har__ frame can't grow without bound. Per-body truncation already happened in
	// buildSDKHAREntry, so a single entry adds at most 2*maxCapturedBodyBytes here.
	entryBytes := int64(len(entry.Response.Content.Text))
	if entry.Request.PostData != nil {
		entryBytes += int64(len(entry.Request.PostData.Text))
	}
	if b.retained >= maxCapturedTotalBytes {
		entry.Response.Content.Text = ""
		entry.Response.Content.Encoding = ""
		if entry.Request.PostData != nil {
			entry.Request.PostData.Text = ""
			entry.Request.PostData.Encoding = ""
		}
	} else {
		b.retained += entryBytes
	}
	b.entries = append(b.entries, entry)
}

// drainRequestBody reads and returns the request body (up to the capture cap), restoring it so the
// request can still be sent. It must be called before the request is sent: a real http.Transport
// consumes (and closes) req.Body while sending, so reading it afterwards yields nothing. Returns nil
// when there is no body. (Request bodies are query payloads and rarely exceed the cap, so truncation
// is not reflected in the request bodySize.)
func drainRequestBody(req *http.Request) []byte {
	if req == nil || req.Body == nil || req.Body == http.NoBody {
		return nil
	}
	body, _, restored := readAndRestoreBody(req.Body)
	req.Body = restored
	return body
}

// readAndRestoreBody reads up to maxCapturedBodyBytes of rc for capture and returns those bytes,
// whether the body was longer than the cap (truncated), and a ReadCloser that hands the original
// consumer the full body -- the captured prefix followed by the untouched, lazily-streamed remainder
// -- so capture never buffers more than the cap regardless of how large the body is. When the read
// fails partway (e.g. this SDK's ResponseLimitMiddleware deliberately errors past a size cap, or a
// transient network error), the captured bytes are what was read so far and the replay reader
// re-surfaces the same error after them, exactly what downstream would have observed. rc is closed
// once the returned ReadCloser is closed (or immediately when there is no remainder to stream).
func readAndRestoreBody(rc io.ReadCloser) ([]byte, bool, io.ReadCloser) {
	// Read one byte past the cap so a full body (<= cap) can be told from a truncated one (> cap)
	// without buffering the whole thing.
	buf, err := io.ReadAll(io.LimitReader(rc, maxCapturedBodyBytes+1))
	if err != nil {
		_ = rc.Close()
		return buf, false, &errorReader{r: bytes.NewReader(buf), err: err}
	}
	if int64(len(buf)) <= maxCapturedBodyBytes {
		// Whole body fit within the cap; nothing left in rc.
		_ = rc.Close()
		return buf, false, io.NopCloser(bytes.NewReader(buf))
	}
	// Body is larger than the cap: retain only the capped prefix for the HAR, but let the consumer
	// read the full buffered head (buf, which is cap+1 bytes) followed by the untouched remainder
	// streamed lazily from rc, so we never buffer the whole body.
	captured := buf[:maxCapturedBodyBytes]
	return captured, true, &bodyRemainder{r: io.MultiReader(bytes.NewReader(buf), rc), c: rc}
}

// errorReader replays buffered bytes and then returns err in place of io.EOF, reproducing a body
// read that failed partway so capture doesn't hide the failure from the body's original consumer.
type errorReader struct {
	r   *bytes.Reader
	err error
}

func (e *errorReader) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if err == io.EOF {
		return n, e.err
	}
	return n, err
}

func (e *errorReader) Close() error { return nil }

// bodyRemainder is a ReadCloser over a size-capped body: it replays the buffered prefix and streams
// the untouched remainder, closing the underlying body on Close so the connection is released.
type bodyRemainder struct {
	r io.Reader
	c io.Closer
}

func (b *bodyRemainder) Read(p []byte) (int, error) { return b.r.Read(p) }
func (b *bodyRemainder) Close() error               { return b.c.Close() }

func (b *sdkHARCaptureBuffer) len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}

func (b *sdkHARCaptureBuffer) toHARString() (string, error) {
	b.mu.Lock()
	entries := make([]sdkHAREntry, len(b.entries))
	copy(entries, b.entries)
	b.mu.Unlock()

	doc := sdkHARDocument{
		Log: sdkHARLog{
			Version: "1.2",
			Creator: sdkHARCreator{Name: "grafana-plugin-sdk-go", Version: "1.0"},
			Entries: entries,
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

type sdkHARDocument struct {
	Log sdkHARLog `json:"log"`
}

type sdkHARLog struct {
	Version string        `json:"version"`
	Creator sdkHARCreator `json:"creator"`
	Entries []sdkHAREntry `json:"entries"`
}

type sdkHARCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type sdkHAREntry struct {
	StartedDateTime string         `json:"startedDateTime"`
	Time            float64        `json:"time"`
	Request         sdkHARRequest  `json:"request"`
	Response        sdkHARResponse `json:"response"`
	Cache           sdkHARCache    `json:"cache"`
	Timings         sdkHARTimings  `json:"timings"`
	// Comment carries a transport-level error (connection refused, DNS/TLS failure, timeout) for a
	// request that never produced an HTTP response; such entries have a zero-status response.
	Comment string `json:"comment,omitempty"`
}

type sdkHARRequest struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	HTTPVersion string            `json:"httpVersion"`
	Headers     []sdkHARNameValue `json:"headers"`
	QueryString []sdkHARNameValue `json:"queryString"`
	Cookies     []sdkHARCookie    `json:"cookies"`
	PostData    *sdkHARPostData   `json:"postData,omitempty"`
	BodySize    int64             `json:"bodySize"`
	HeadersSize int64             `json:"headersSize"`
}

type sdkHARResponse struct {
	Status      int               `json:"status"`
	StatusText  string            `json:"statusText"`
	HTTPVersion string            `json:"httpVersion"`
	Headers     []sdkHARNameValue `json:"headers"`
	Cookies     []sdkHARCookie    `json:"cookies"`
	Content     sdkHARContent     `json:"content"`
	RedirectURL string            `json:"redirectURL"`
	BodySize    int64             `json:"bodySize"`
	HeadersSize int64             `json:"headersSize"`
}

type sdkHARNameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// sdkHARCookie mirrors the HAR cookie object (name/value, as the e2e HAR storage emits).
type sdkHARCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// sdkHARCache is the HAR per-entry cache object. We don't model cache state, so it serializes as
// an empty object -- which is what the spec/e2e replay expects when caching isn't recorded.
type sdkHARCache struct{}

type sdkHARPostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
	// Encoding is "base64" when Text is a base64 encoding of a non-UTF-8 body. HAR 1.2 defines
	// encoding only on response content, so this is an extension; canonical HAR parsers ignore it.
	Encoding string `json:"encoding,omitempty"`
}

type sdkHARContent struct {
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
	// Encoding is "base64" when Text is a base64 encoding of a non-UTF-8 body (HAR 1.2 content.encoding).
	Encoding string `json:"encoding,omitempty"`
}

// encodeBody renders a body for a HAR text field. It base64-encodes when the bytes are not valid
// UTF-8, so binary payloads (protobuf, images, ...) survive json.Marshal instead of being silently
// corrupted to U+FFFD, and caps the retained bytes at maxCapturedBodyBytes. The caller records the
// true, uncapped size separately (bodySize/content.size).
func encodeBody(body []byte) (text, encoding string) {
	keep := body
	if len(keep) > maxCapturedBodyBytes {
		keep = keep[:maxCapturedBodyBytes]
	}
	if utf8.Valid(keep) {
		return string(keep), ""
	}
	return base64.StdEncoding.EncodeToString(keep), "base64"
}

type sdkHARTimings struct {
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

// buildSDKHAREntry builds a HAR entry from the request/response pair. reqBody is the request body
// captured before the request was sent (see drainRequestBody); it is passed in rather than read
// from req.Body here, because by the time capture runs the transport has already drained the body.
// rtErr is the RoundTrip error (nil on success): a transport-level failure (connection refused,
// DNS/TLS error, timeout) leaves resp nil, and the entry records the error in Comment.
func buildSDKHAREntry(req *http.Request, reqBody []byte, resp *http.Response, rtErr error, started time.Time, elapsed time.Duration) sdkHAREntry {
	reqHeaders := sdkHeadersToNameValue(req.Header)
	queryString := make([]sdkHARNameValue, 0, len(req.URL.Query()))
	for k, vals := range req.URL.Query() {
		for _, v := range vals {
			queryString = append(queryString, sdkHARNameValue{Name: k, Value: v})
		}
	}

	var postData *sdkHARPostData
	reqBodySize := int64(len(reqBody))
	if len(reqBody) > 0 {
		text, encoding := encodeBody(reqBody)
		postData = &sdkHARPostData{
			MimeType: req.Header.Get("Content-Type"),
			Text:     text,
			Encoding: encoding,
		}
	}

	harResp := sdkHARResponse{HeadersSize: -1}
	if resp != nil {
		harResp.Status = resp.StatusCode
		harResp.StatusText = resp.Status
		harResp.HTTPVersion = resp.Proto
		harResp.Headers = sdkHeadersToNameValue(resp.Header)
		harResp.Cookies = sdkCookies(resp.Cookies())
		harResp.RedirectURL = resp.Header.Get("Location")
		if resp.Body != nil {
			// Always restore resp.Body -- even on a read error -- so capturing never truncates the
			// response the plugin actually receives (see readAndRestoreBody).
			body, truncated, restored := readAndRestoreBody(resp.Body)
			resp.Body = restored
			text, encoding := encodeBody(body)
			// When the body exceeded the capture cap we hold only a prefix, so the true size is
			// unknown: report -1 (HAR "unavailable") for bodySize; content.size is what we captured.
			harResp.BodySize = int64(len(body))
			if truncated {
				harResp.BodySize = -1
			}
			harResp.Content = sdkHARContent{
				Size:     int64(len(body)),
				MimeType: resp.Header.Get("Content-Type"),
				Text:     text,
				Encoding: encoding,
			}
		}
	}

	var comment string
	if rtErr != nil {
		comment = "transport error: " + rtErr.Error()
	}

	waitMs := float64(elapsed.Milliseconds())
	return sdkHAREntry{
		StartedDateTime: started.UTC().Format(time.RFC3339),
		Time:            waitMs,
		Request: sdkHARRequest{
			Method:      req.Method,
			URL:         req.URL.String(),
			HTTPVersion: req.Proto,
			Headers:     reqHeaders,
			QueryString: queryString,
			Cookies:     sdkCookies(req.Cookies()),
			PostData:    postData,
			BodySize:    reqBodySize,
			HeadersSize: -1,
		},
		Response: harResp,
		Cache:    sdkHARCache{},
		Timings:  sdkHARTimings{Send: 0, Wait: waitMs, Receive: 0},
		Comment:  comment,
	}
}

func sdkHeadersToNameValue(h http.Header) []sdkHARNameValue {
	result := make([]sdkHARNameValue, 0, len(h))
	for name, vals := range h {
		// Emit one entry per value so repeated headers (e.g. multiple Set-Cookie) are preserved.
		for _, v := range vals {
			result = append(result, sdkHARNameValue{Name: name, Value: v})
		}
	}
	return result
}

// sdkCookies converts parsed HTTP cookies into HAR cookie entries (name/value), matching the
// e2e HAR storage output so captured traffic stays replayable by the E2E fixture proxy.
func sdkCookies(cookies []*http.Cookie) []sdkHARCookie {
	result := make([]sdkHARCookie, 0, len(cookies))
	for _, c := range cookies {
		result = append(result, sdkHARCookie{Name: c.Name, Value: c.Value})
	}
	return result
}
