package backend

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

// sdkHARCaptureBuffer collects HTTP request/response pairs in HAR 1.2 format in memory.
// Used by the SDK HAR capture middleware to accumulate traffic from external plugin HTTP clients.
type sdkHARCaptureBuffer struct {
	mu      sync.Mutex
	entries []sdkHAREntry
}

func newSDKHARCaptureBuffer() *sdkHARCaptureBuffer {
	return &sdkHARCaptureBuffer{}
}

func (b *sdkHARCaptureBuffer) addEntry(req *http.Request, reqBody []byte, resp *http.Response, started time.Time, elapsed time.Duration) {
	entry := buildSDKHAREntry(req, reqBody, resp, started, elapsed)
	b.mu.Lock()
	b.entries = append(b.entries, entry)
	b.mu.Unlock()
}

// drainRequestBody reads and returns the request body, restoring it so the request can still be
// sent. It must be called before the request is sent: a real http.Transport consumes (and closes)
// req.Body while sending, so reading it afterwards yields nothing. Returns nil when there is no
// body.
func drainRequestBody(req *http.Request) []byte {
	if req == nil || req.Body == nil || req.Body == http.NoBody {
		return nil
	}
	body, restored := readAndRestoreBody(req.Body)
	req.Body = restored
	return body
}

// readAndRestoreBody reads rc fully for capture and returns the bytes read together with a
// ReadCloser that replays them to the original consumer, so capture never alters what the plugin
// sees. When the read fails partway (e.g. this SDK's ResponseLimitMiddleware deliberately errors
// past a size cap, or a transient network error), the returned bytes are what was read so far and
// the replay reader re-surfaces the same error after those bytes -- exactly what downstream would
// have observed without capture. rc is closed.
func readAndRestoreBody(rc io.ReadCloser) ([]byte, io.ReadCloser) {
	body, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		return body, &errorReader{r: bytes.NewReader(body), err: err}
	}
	return body, io.NopCloser(bytes.NewReader(body))
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
}

type sdkHARContent struct {
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type sdkHARTimings struct {
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

// buildSDKHAREntry builds a HAR entry from the request/response pair. reqBody is the request body
// captured before the request was sent (see drainRequestBody); it is passed in rather than read
// from req.Body here, because by the time capture runs the transport has already drained the body.
func buildSDKHAREntry(req *http.Request, reqBody []byte, resp *http.Response, started time.Time, elapsed time.Duration) sdkHAREntry {
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
		postData = &sdkHARPostData{
			MimeType: req.Header.Get("Content-Type"),
			Text:     string(reqBody),
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
			body, restored := readAndRestoreBody(resp.Body)
			resp.Body = restored
			harResp.BodySize = int64(len(body))
			harResp.Content = sdkHARContent{
				Size:     int64(len(body)),
				MimeType: resp.Header.Get("Content-Type"),
				Text:     string(body),
			}
		}
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
