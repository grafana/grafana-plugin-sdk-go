package harcapture

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// defaultMaxCapturedBodyBytes caps how much of one request/response body capture reads into memory;
// the untouched remainder streams straight to the real consumer (see readAndRestoreBody). Past the cap
// the captured prefix is kept and the entry reports BodySize -1 (unavailable), so an over-cap body is
// still evidence, just marked incomplete.
//
// defaultMaxCapturedTotalBytes caps the total payload retained across one request's entries -- the
// size of the serialized __har__ frame. It's measured in jsonEscapedLen, not raw bytes, since that's
// what actually crosses the wire: json.Marshal can expand a byte into a 6-byte \u00XX escape. How
// many full bodies fit is therefore content-dependent: escape-free text fits four, markup (whose <,
// >, & all escape 6 bytes) fewer, and a single body dense in control characters can exceed the whole
// budget escaped and have its payload dropped on arrival.
//
// Sizing: the __har__ frame rides the same plugin<->core gRPC message as the QueryDataResponse data
// itself. Both directions of that hop default to ~2 GiB -- the plugin serve side leaves
// GRPCSettings.MaxSendMsgSize at math.MaxInt32, and the Grafana core side dials with go-plugin's
// default call options (MaxCallRecvMsgSize(math.MaxInt32), go-plugin grpc_client.go) and does not
// override them -- but serialization transiently holds full extra copies of what is retained
// (json.Marshal's output, then its string conversion), and the response data shares the message. 256
// MiB post-escaping keeps the frame plus those copies comfortably under the ceiling. Peak memory can
// also exceed the total by roughly two body caps live (~3x allocated) per concurrent over-cap
// response: the capped head is held transiently before being dropped, and the capture buffer's growth
// chain past the pre-grow clamp (see readAndRestoreBody) transiently holds copies of comparable size.
//
// Each Buffer holds its own copy of these caps (see NewBuffer), rather than reading package vars, so
// a test can shrink them for one Buffer without a data race against any other test's Buffer.
const (
	defaultMaxCapturedBodyBytes  int64 = 64 << 20  // 64 MiB
	defaultMaxCapturedTotalBytes int64 = 256 << 20 // 256 MiB, measured post-escaping (see jsonEscapedLen)
)

// redactedValue replaces the value of anything capture treats as sensitive (see
// isSensitiveHeaderName, isSensitiveQueryParamName, and sdkCookies), so the __har__ frame -- which
// is returned to whoever enabled capture -- never carries datasource credentials.
const redactedValue = "REDACTED"

// sensitiveHeaderNames are header names whose values are redacted before capture (matched
// case-insensitively by isSensitiveHeaderName), since they routinely carry datasource credentials
// (bearer tokens, API keys, session cookies). This is a defense-in-depth safety net, not a substitute
// for redaction wherever the captured HAR is ultimately stored or displayed: it cannot be exhaustive,
// since datasources are free to invent their own auth header names.
var sensitiveHeaderNames = map[string]struct{}{
	"authorization":       {},
	"proxy-authorization": {},
	"cookie":              {},
	"set-cookie":          {},
	"x-api-key":           {},
}

func isSensitiveHeaderName(name string) bool {
	_, ok := sensitiveHeaderNames[strings.ToLower(name)]
	return ok
}

// sensitiveQueryParamNames are query string parameter names whose values are redacted before
// capture, since datasource URLs commonly carry credentials as query params (API keys, signed-URL
// signatures) rather than headers. Matched case-insensitively by isSensitiveQueryParamName.
var sensitiveQueryParamNames = map[string]struct{}{
	"api_key":         {},
	"apikey":          {},
	"api-key":         {},
	"access_token":    {},
	"token":           {},
	"sig":             {},
	"signature":       {},
	"x-amz-signature": {},
	"key":             {},
}

func isSensitiveQueryParamName(name string) bool {
	_, ok := sensitiveQueryParamNames[strings.ToLower(name)]
	return ok
}

// Buffer collects HTTP request/response pairs in HAR 1.2 format in memory.
// Used by the SDK HAR capture middleware to accumulate traffic from external plugin HTTP clients.
type Buffer struct {
	mu       sync.Mutex
	entries  []sdkHAREntry
	retained int64 // running total of retained payload bytes, for the total-size cap

	// maxBodyBytes and maxTotalBytes are set once, at construction, and never modified afterwards, so
	// reading them needs no synchronization of its own. Zero means "unset" (the zero-value Buffer{} is
	// a valid, pre-NewBuffer construction some callers still use) and is resolved to the matching
	// default by maxBody/maxTotal rather than read directly.
	maxBodyBytes  int64
	maxTotalBytes int64
}

func NewBuffer() *Buffer {
	return &Buffer{
		maxBodyBytes:  defaultMaxCapturedBodyBytes,
		maxTotalBytes: defaultMaxCapturedTotalBytes,
	}
}

func (b *Buffer) maxBody() int64 {
	if b.maxBodyBytes == 0 {
		return defaultMaxCapturedBodyBytes
	}
	return b.maxBodyBytes
}

func (b *Buffer) maxTotal() int64 {
	if b.maxTotalBytes == 0 {
		return defaultMaxCapturedTotalBytes
	}
	return b.maxTotalBytes
}

func (b *Buffer) AddEntry(req *http.Request, reqBody []byte, reqTruncated bool, resp *http.Response, rtErr error, started time.Time, elapsed time.Duration) {
	b.appendEntry(buildSDKHAREntry(req, reqBody, reqTruncated, resp, rtErr, started, elapsed, b.maxBody()))
}

// appendEntry adds an already-built entry under the buffer's cumulative retained-payload budget.
// Every producer goes through here -- HTTP round trips and non-HTTP query interactions alike -- so
// one budget bounds the whole document whatever mix of traffic a request produced.
func (b *Buffer) appendEntry(entry sdkHAREntry) {
	// Size the payload before taking the lock: payloadBytes is an O(n) scan over the entry's retained
	// strings, and the entry is not shared until appended.
	payload := entry.payloadBytes()
	b.mu.Lock()
	defer b.mu.Unlock()
	// Enforce the cumulative retained-payload budget: an entry whose payload would take the request
	// past maxTotalBytes keeps its metadata (headers, sizes, timings, error) but drops the payload
	// itself, so the __har__ frame can't grow without bound. The check is on the sum rather than on
	// what is already retained, so the entry that crosses the budget is trimmed too rather than kept
	// whole.
	if b.retained+payload > b.maxTotal() {
		entry.dropPayload()
		b.retained += entry.payloadBytes()
	} else {
		b.retained += payload
	}
	b.entries = append(b.entries, entry)
}

// payloadBytes is what an entry contributes to the buffer's retained-payload budget. Measured as
// jsonEscapedLen rather than raw length: the budget bounds the serialized __har__ frame that actually
// crosses the plugin<->core gRPC boundary (see Buffer.maxTotalBytes), and json.Marshal's own escaping
// is what determines that size, not how many bytes are held in memory before marshaling runs.
func (e *sdkHAREntry) payloadBytes() int64 {
	n := jsonEscapedLen(e.Response.Content.Text)
	if e.Request.PostData != nil {
		n += jsonEscapedLen(e.Request.PostData.Text)
	}
	return n + e.Query.payloadBytes()
}

// dropPayload clears an entry's payload, keeping what tells a reader the exchange happened and how it
// went: headers, the true sizes, timings and any error.
func (e *sdkHAREntry) dropPayload() {
	e.Response.Content.Text = ""
	e.Response.Content.Encoding = ""
	if e.Request.PostData != nil {
		e.Request.PostData.Text = ""
		e.Request.PostData.Encoding = ""
	}
	e.Query.dropPayload()
}

// DrainRequestBody reads and returns the request body (up to b's per-body capture cap) and whether
// it was larger than the cap (truncated), restoring it so the request can still be sent. It must be
// called before the request is sent: a real http.Transport consumes (and closes) req.Body while
// sending, so reading it afterwards yields nothing. Returns nil when there is no body.
func (b *Buffer) DrainRequestBody(req *http.Request) ([]byte, bool) {
	if req == nil || req.Body == nil || req.Body == http.NoBody {
		return nil, false
	}
	body, truncated, restored := readAndRestoreBody(req.Body, req.ContentLength, b.maxBody())
	req.Body = restored
	return body, truncated
}

// DrainRequestBody reads and returns the request body (up to the default per-body capture cap) and
// whether it was larger than the cap, restoring it so the request can still be sent.
//
// Deprecated: use Buffer.DrainRequestBody, which applies the buffer's own caps.
func DrainRequestBody(req *http.Request) ([]byte, bool) {
	return NewBuffer().DrainRequestBody(req)
}

// readAndRestoreBody reads up to maxBodyBytes of rc for capture and returns those bytes, whether the
// captured bytes are NOT the complete body (sizeUnknown -- the body exceeded the cap, or the read
// failed part-way, so its true size is unavailable), and a ReadCloser that hands the original
// consumer the full body -- the captured prefix followed by the untouched, lazily-streamed remainder
// -- so capture never buffers more than the cap regardless of how large the body is. When the read
// fails partway (e.g. this SDK's ResponseLimitMiddleware deliberately errors past a size cap, or a
// transient network error), the captured bytes are what was read so far and the replay reader
// re-surfaces the same error after them, exactly what downstream would have observed. rc is closed
// once the returned ReadCloser is closed (or immediately when there is no remainder to stream).
//
// contentLength is the body's declared Content-Length (-1/0 if unknown, matching http.Request/
// http.Response's own convention) -- used only to size the capture buffer up front; it never changes
// what is read or returned.
func readAndRestoreBody(rc io.ReadCloser, contentLength, maxBodyBytes int64) ([]byte, bool, io.ReadCloser) {
	// Grow the buffer up front instead of letting io.Copy's internal buffer double repeatedly as it
	// fills: that repeated doubling can transiently hold close to twice the final size in memory for
	// no reason, on top of every later copy (encodeBody, json.Marshal) that already has to pay for the
	// real size. Only do this when contentLength is known (a chunked body reports -1), and trust the
	// declared length only up to preGrowLimitBytes: the header is unvalidated, so a lying or failed
	// response must not be able to reserve the whole body cap before any bytes arrive. Larger honest
	// bodies just pay a few doublings past the pre-grown size.
	const preGrowLimitBytes = 4 << 20
	var buf bytes.Buffer
	if contentLength > 0 {
		buf.Grow(int(min(contentLength, maxBodyBytes, preGrowLimitBytes)) + 1)
	}
	// Read one byte past the cap so a full body (<= cap) can be told from a truncated one (> cap)
	// without buffering the whole thing.
	_, err := io.Copy(&buf, io.LimitReader(rc, maxBodyBytes+1))
	captured := buf.Bytes()
	if err != nil {
		// Read failed part-way: we hold a partial prefix and the true size is unavailable.
		_ = rc.Close()
		return captured, true, &errorReader{r: bytes.NewReader(captured), err: err}
	}
	if int64(len(captured)) <= maxBodyBytes {
		// Whole body fit within the cap; nothing left in rc.
		_ = rc.Close()
		return captured, false, io.NopCloser(bytes.NewReader(captured))
	}
	// Body is larger than the cap: retain only the capped prefix for the HAR, but let the consumer
	// read the full buffered head (captured, which is cap+1 bytes) followed by the untouched remainder
	// streamed lazily from rc, so we never buffer the whole body.
	head := captured[:maxBodyBytes]
	return head, true, &bodyRemainder{r: io.MultiReader(bytes.NewReader(captured), rc), c: rc}
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

func (b *Buffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}

func (b *Buffer) ToHARString() (string, error) {
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
	// request that never produced an HTTP response; such entries have a zero-status response. For a
	// query entry it carries the query's own error, in the same shape.
	Comment string `json:"comment,omitempty"`
	// Query is set only on an entry that describes a non-HTTP datasource exchange.
	Query *sdkHARQueryInfo `json:"_query,omitempty"`
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
// corrupted to U+FFFD, and caps the retained bytes at maxBodyBytes. The caller records the true,
// uncapped size separately (bodySize/content.size).
func encodeBody(body []byte, maxBodyBytes int64) (text, encoding string) {
	keep := body
	if int64(len(keep)) > maxBodyBytes {
		keep = keep[:maxBodyBytes]
	}
	if utf8.Valid(keep) {
		return string(keep), ""
	}
	return base64.StdEncoding.EncodeToString(keep), "base64"
}

// jsonEscapedLen estimates how many bytes s occupies once json.Marshal encodes it as a JSON string,
// including the wrapping quotes -- used to budget the aggregate retained-payload cap against the
// actual serialized size instead of raw length (see Buffer.maxTotalBytes). The estimate never falls
// below the actual size: every rune json.Marshal escapes is charged its real cost --
//   - 2 bytes for the short escapes: \" \\ \b \f \n \r \t
//   - 6 bytes for \u00XX (every other byte below 0x20), for the HTML-unsafe runes Marshal always
//     escapes regardless of content (<, >, &), for U+2028/U+2029 (also unconditional), and for a
//     malformed byte (encoded as �) -- charging 6 here even for a literal, validly-encoded
//     U+FFFD character (which Marshal does NOT escape, and which costs only 3) is a deliberate
//     over-estimate: telling the two apart isn't worth the complexity when erring high is safe.
//   - its own byte length for everything else, since a valid, unescaped UTF-8 sequence passes through
//     unchanged.
//
// Matched against encoding/json's actual escaping table (encode.go's appendString / tables.go's
// htmlSafeSet) rather than assumed, since an incorrect table here would silently undercount.
func jsonEscapedLen(s string) int64 {
	n := int64(2) // the wrapping quotes
	for _, r := range s {
		switch r {
		case '"', '\\', '\b', '\f', '\n', '\r', '\t':
			n += 2
		case '<', '>', '&', '\u2028', '\u2029', utf8.RuneError:
			n += 6
		default:
			if r < 0x20 {
				n += 6
			} else {
				n += int64(utf8.RuneLen(r))
			}
		}
	}
	return n
}

type sdkHARTimings struct {
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

// durationMs renders a duration for HAR's millisecond-valued time and timings fields, keeping the
// fraction: a database/sql call over a pooled connection is routinely sub-millisecond, and truncating
// would make it indistinguishable from an instantaneous one.
func durationMs(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / float64(time.Millisecond)
}

// buildSDKHAREntry builds a HAR entry from the request/response pair. reqBody is the request body
// captured before the request was sent (see DrainRequestBody); it is passed in rather than read
// from req.Body here, because by the time capture runs the transport has already drained the body.
// rtErr is the RoundTrip error (nil on success): a transport-level failure (connection refused,
// DNS/TLS error, timeout) leaves resp nil, and the entry records the error in Comment.
func buildSDKHAREntry(req *http.Request, reqBody []byte, reqTruncated bool, resp *http.Response, rtErr error, started time.Time, elapsed time.Duration, maxBodyBytes int64) sdkHAREntry {
	reqHeaders := sdkHeadersToNameValue(req.Header)
	queryString := make([]sdkHARNameValue, 0, len(req.URL.Query()))
	for k, vals := range req.URL.Query() {
		redact := isSensitiveQueryParamName(k)
		for _, v := range vals {
			if redact {
				v = redactedValue
			}
			queryString = append(queryString, sdkHARNameValue{Name: k, Value: v})
		}
	}

	var postData *sdkHARPostData
	reqBodySize := int64(len(reqBody))
	if reqTruncated {
		// Only a capped prefix was read, so the true length is unknown: report -1 (HAR
		// "unavailable"), symmetric with the response side.
		reqBodySize = -1
	}
	if len(reqBody) > 0 {
		text, encoding := encodeBody(reqBody, maxBodyBytes)
		postData = &sdkHARPostData{
			MimeType: req.Header.Get("Content-Type"),
			Text:     text,
			Encoding: encoding,
		}
	}

	// Default bodySize -1 ("unavailable" in HAR): when there is no response at all (a transport
	// failure), an entry with bodySize 0 would misrepresent it as an empty body.
	harResp := sdkHARResponse{HeadersSize: -1, BodySize: -1}
	if resp != nil {
		harResp.Status = resp.StatusCode
		harResp.StatusText = resp.Status
		harResp.HTTPVersion = resp.Proto
		harResp.Headers = sdkHeadersToNameValue(resp.Header)
		harResp.Cookies = sdkCookies(resp.Cookies())
		harResp.RedirectURL = resp.Header.Get("Location")
		harResp.BodySize = 0 // have a response; 0 unless a body is read below
		if resp.Body != nil {
			// Always restore resp.Body -- even on a read error -- so capturing never truncates the
			// response the plugin actually receives (see readAndRestoreBody).
			body, truncated, restored := readAndRestoreBody(resp.Body, resp.ContentLength, maxBodyBytes)
			resp.Body = restored
			text, encoding := encodeBody(body, maxBodyBytes)
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

	waitMs := durationMs(elapsed)
	return sdkHAREntry{
		StartedDateTime: started.UTC().Format(time.RFC3339Nano),
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

// sdkHeadersToNameValue converts an http.Header into HAR name/value pairs, redacting the value of
// any header in sensitiveHeaderNames (Authorization, Cookie, ...) so capture never surfaces
// datasource credentials.
func sdkHeadersToNameValue(h http.Header) []sdkHARNameValue {
	result := make([]sdkHARNameValue, 0, len(h))
	for name, vals := range h {
		redact := isSensitiveHeaderName(name)
		// Emit one entry per value so repeated headers (e.g. multiple Set-Cookie) are preserved.
		for _, v := range vals {
			if redact {
				v = redactedValue
			}
			result = append(result, sdkHARNameValue{Name: name, Value: v})
		}
	}
	return result
}

// sdkCookies converts parsed HTTP cookies into HAR cookie entries, matching the e2e HAR storage
// output so captured traffic stays replayable by the E2E fixture proxy. Values are always redacted:
// unlike header names, cookie names aren't a reliable signal of sensitivity, and a cookie's value is
// itself typically the credential (session ID, auth token), so there is no safe default to keep.
func sdkCookies(cookies []*http.Cookie) []sdkHARCookie {
	result := make([]sdkHARCookie, 0, len(cookies))
	for _, c := range cookies {
		result = append(result, sdkHARCookie{Name: c.Name, Value: redactedValue})
	}
	return result
}
