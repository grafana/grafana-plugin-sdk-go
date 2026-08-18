package harcapture

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/querycapture"
)

// flakyBody yields data and then returns err (not io.EOF), simulating a body whose read fails
// partway -- e.g. the SDK's ResponseLimitMiddleware erroring past a size cap.
type flakyBody struct {
	data []byte
	pos  int
	err  error
}

func (f *flakyBody) Read(p []byte) (int, error) {
	if f.pos >= len(f.data) {
		return 0, f.err
	}
	n := copy(p, f.data[f.pos:])
	f.pos += n
	return n, nil
}

func (f *flakyBody) Close() error { return nil }

// testMaxBodyBytes and testMaxTotalBytes are small stand-ins for the real caps, so cap-boundary tests
// exercise the capping algorithm against small fixtures instead of allocating and copying data sized
// to the real caps.
const (
	testMaxBodyBytes  int64 = 4096
	testMaxTotalBytes int64 = 16384
)

// newBufferWithLimits returns a Buffer with the given caps instead of the real defaults, for
// cap-boundary tests. Each Buffer holds its own caps (see NewBuffer), so unlike a package-level
// override this needs no cleanup and is safe alongside any other test's Buffer, parallel or not.
func newBufferWithLimits(maxBodyBytes, maxTotalBytes int64) *Buffer {
	return &Buffer{maxBodyBytes: maxBodyBytes, maxTotalBytes: maxTotalBytes}
}

// TestBuildSDKHAREntry_restoresBodyOnReadError asserts that capturing a response whose body read
// fails still restores resp.Body, so the plugin's downstream consumer sees the same bytes and the
// same error it would have without capture (capture must never truncate the real response).
func TestBuildSDKHAREntry_restoresBodyOnReadError(t *testing.T) {
	wantErr := errors.New("response size limit exceeded")
	req, err := http.NewRequest(http.MethodGet, "http://ds.example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Proto:      "HTTP/1.1",
		Header:     http.Header{},
		Body:       &flakyBody{data: []byte("partial-body"), err: wantErr},
	}

	entry := buildSDKHAREntry(req, nil, false, resp, nil, time.Now(), time.Millisecond, testMaxBodyBytes)

	// The HAR entry captures whatever bytes were read before the error.
	if entry.Response.Content.Text != "partial-body" {
		t.Errorf("captured body = %q, want %q", entry.Response.Content.Text, "partial-body")
	}

	// Downstream still sees the partial bytes followed by the original error, not an empty body.
	got, readErr := io.ReadAll(resp.Body)
	if string(got) != "partial-body" {
		t.Errorf("restored body = %q, want %q", got, "partial-body")
	}
	if !errors.Is(readErr, wantErr) {
		t.Errorf("restored body read error = %v, want %v", readErr, wantErr)
	}
}

// TestBuildSDKHAREntry_multiValuedHeadersAndQuery asserts repeated headers and query params are all
// captured, not just the first value (HAR parity).
func TestBuildSDKHAREntry_multiValuedHeadersAndQuery(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://ds.example.com/q?a=1&a=2&b=3", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("X-Multi", "one")
	req.Header.Add("X-Multi", "two")

	entry := buildSDKHAREntry(req, nil, false, &http.Response{Header: http.Header{}}, nil, time.Now(), time.Millisecond, testMaxBodyBytes)

	countHeader := func(pairs []sdkHARNameValue, name, value string) bool {
		for _, p := range pairs {
			if p.Name == name && p.Value == value {
				return true
			}
		}
		return false
	}
	if !countHeader(entry.Request.Headers, "X-Multi", "one") || !countHeader(entry.Request.Headers, "X-Multi", "two") {
		t.Errorf("both X-Multi header values must be captured, got %+v", entry.Request.Headers)
	}

	var aValues []string
	for _, p := range entry.Request.QueryString {
		if p.Name == "a" {
			aValues = append(aValues, p.Value)
		}
	}
	if len(aValues) != 2 {
		t.Errorf("both values of query param a must be captured, got %v", aValues)
	}
}

// TestBuildSDKHAREntry_binaryBodyBase64 asserts a non-UTF-8 response body is base64-encoded with
// encoding="base64", rather than corrupted to U+FFFD by json.Marshal.
func TestBuildSDKHAREntry_binaryBodyBase64(t *testing.T) {
	binary := []byte{0x00, 0x01, 0xff, 0xfe, 0x80}
	req, err := http.NewRequest(http.MethodGet, "http://ds.example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp := &http.Response{Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(binary))}

	entry := buildSDKHAREntry(req, nil, false, resp, nil, time.Now(), time.Millisecond, testMaxBodyBytes)

	if entry.Response.Content.Encoding != "base64" {
		t.Fatalf("non-UTF-8 body must be marked encoding=base64, got %q", entry.Response.Content.Encoding)
	}
	if entry.Response.Content.Text != base64.StdEncoding.EncodeToString(binary) {
		t.Errorf("body not base64-encoded: %q", entry.Response.Content.Text)
	}
	if entry.Response.Content.Size != int64(len(binary)) {
		t.Errorf("content size = %d, want %d (true byte length)", entry.Response.Content.Size, len(binary))
	}
}

// TestBuildSDKHAREntry_transportError asserts a failed RoundTrip (no HTTP response) is still
// captured: the request is recorded, the response has zero status, and the error lands in Comment.
func TestBuildSDKHAREntry_transportError(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://ds.example.com/q", nil)
	if err != nil {
		t.Fatal(err)
	}
	rtErr := errors.New("dial tcp: connection refused")

	entry := buildSDKHAREntry(req, nil, false, nil, rtErr, time.Now(), time.Millisecond, testMaxBodyBytes)

	if entry.Request.URL != "http://ds.example.com/q" {
		t.Errorf("failed request must still be captured, got URL %q", entry.Request.URL)
	}
	if entry.Response.Status != 0 {
		t.Errorf("no-response entry must have zero status, got %d", entry.Response.Status)
	}
	if entry.Response.BodySize != -1 {
		t.Errorf("no-response entry must report bodySize -1 (unavailable), got %d", entry.Response.BodySize)
	}
	if !strings.Contains(entry.Comment, "connection refused") {
		t.Errorf("transport error must be recorded in Comment, got %q", entry.Comment)
	}
}

// TestReadAndRestoreBody_capsCaptureButDeliversFullBody asserts a body larger than the per-body cap
// is only partially buffered for capture, yet the original consumer still receives every byte.
func TestReadAndRestoreBody_capsCaptureButDeliversFullBody(t *testing.T) {
	full := bytes.Repeat([]byte("x"), int(testMaxBodyBytes)+4096)

	captured, truncated, restored := readAndRestoreBody(io.NopCloser(bytes.NewReader(full)), -1, testMaxBodyBytes)

	if !truncated {
		t.Fatal("a body larger than the cap must be reported as truncated")
	}
	if int64(len(captured)) != testMaxBodyBytes {
		t.Errorf("captured %d bytes, want the cap %d (capture must not buffer the whole body)", len(captured), testMaxBodyBytes)
	}

	got, err := io.ReadAll(restored)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(full) {
		t.Errorf("consumer received %d bytes, want the full %d", len(got), len(full))
	}
	if err := restored.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// TestReadAndRestoreBody_knownContentLength asserts a known Content-Length only changes how the
// capture buffer is pre-sized, never what is captured or delivered to the consumer -- including when
// the declared length is itself larger than the cap (the buffer must still grow only to the cap, not
// to the declared length).
func TestReadAndRestoreBody_knownContentLength(t *testing.T) {
	t.Run("within the cap", func(t *testing.T) {
		body := []byte("hello world")
		captured, truncated, restored := readAndRestoreBody(io.NopCloser(bytes.NewReader(body)), int64(len(body)), testMaxBodyBytes)
		if truncated {
			t.Error("a body within the cap must not be reported as truncated")
		}
		if string(captured) != string(body) {
			t.Errorf("captured %q, want %q", captured, body)
		}
		got, err := io.ReadAll(restored)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(body) {
			t.Errorf("consumer received %q, want %q", got, body)
		}
	})

	t.Run("declared length past the cap", func(t *testing.T) {
		full := bytes.Repeat([]byte("x"), int(testMaxBodyBytes)+4096)
		captured, truncated, restored := readAndRestoreBody(io.NopCloser(bytes.NewReader(full)), int64(len(full)), testMaxBodyBytes)
		if !truncated {
			t.Fatal("a body larger than the cap must be reported as truncated")
		}
		if int64(len(captured)) != testMaxBodyBytes {
			t.Errorf("captured %d bytes, want the cap %d", len(captured), testMaxBodyBytes)
		}
		got, err := io.ReadAll(restored)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(full) {
			t.Errorf("consumer received %d bytes, want the full %d", len(got), len(full))
		}
	})
}

// TestBuildSDKHAREntry_truncatedBodyReportsUnknownSize asserts an over-cap response reports bodySize
// -1 (HAR "unavailable") since the true length isn't known, while content still holds the prefix.
func TestBuildSDKHAREntry_truncatedBodyReportsUnknownSize(t *testing.T) {
	full := bytes.Repeat([]byte("y"), int(testMaxBodyBytes)+4096)
	req, err := http.NewRequest(http.MethodGet, "http://ds.example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp := &http.Response{Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(full))}

	entry := buildSDKHAREntry(req, nil, false, resp, nil, time.Now(), time.Millisecond, testMaxBodyBytes)

	if entry.Response.BodySize != -1 {
		t.Errorf("truncated body bodySize = %d, want -1 (unknown)", entry.Response.BodySize)
	}
	if entry.Response.Content.Size != testMaxBodyBytes {
		t.Errorf("content size = %d, want the captured prefix length %d", entry.Response.Content.Size, testMaxBodyBytes)
	}
}

// TestSDKHARCaptureBuffer_totalSizeCap asserts that once the cumulative retained body budget is
// exhausted, later entries keep their metadata/sizes but drop the body text.
func TestSDKHARCaptureBuffer_totalSizeCap(t *testing.T) {
	buf := newBufferWithLimits(testMaxBodyBytes, testMaxTotalBytes)
	big := strings.Repeat("a", int(testMaxBodyBytes)) // one per-body-capped chunk each

	// Enough entries to blow past the total budget.
	for i := int64(0); i < (testMaxTotalBytes/testMaxBodyBytes)+2; i++ {
		req, err := http.NewRequest(http.MethodGet, "http://ds.example.com", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp := &http.Response{Header: http.Header{}, Body: io.NopCloser(strings.NewReader(big))}
		buf.AddEntry(req, nil, false, resp, nil, time.Now(), time.Millisecond)
	}

	var total int
	var droppedText, keptTrueSize bool
	for _, e := range buf.entries {
		total += len(e.Response.Content.Text)
		if e.Response.Content.Text == "" && e.Response.Content.Size == int64(len(big)) {
			droppedText = true // metadata/size preserved, text dropped
		}
		if e.Response.Content.Size == int64(len(big)) {
			keptTrueSize = true
		}
	}
	if int64(total) > testMaxTotalBytes {
		t.Errorf("retained body text %d exceeds the cap budget %d", total, testMaxTotalBytes)
	}
	if !droppedText {
		t.Error("expected later entries to drop body text once over the total budget")
	}
	if !keptTrueSize {
		t.Error("true body size must be preserved even when text is dropped")
	}
}

// TestSDKHARCaptureBuffer_totalSizeCapIsTight asserts that the budget bounds the document rather than
// merely triggering on it: the entry that would take the total past the cap is trimmed itself, instead
// of being retained whole and only trimming its successors. It is filled unevenly on purpose, so that
// an entry lands astride the cap -- the case a mixed query-and-HTTP capture reaches easily, since the
// two producers contribute entries of very different sizes.
func TestSDKHARCaptureBuffer_totalSizeCapIsTight(t *testing.T) {
	buf := newBufferWithLimits(testMaxBodyBytes, testMaxTotalBytes)
	addEntry := func(body string) {
		req, err := http.NewRequest(http.MethodGet, "http://ds.example.com", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp := &http.Response{Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
		buf.AddEntry(req, nil, false, resp, nil, time.Now(), time.Millisecond)
	}

	// Leave the budget with less room than one full chunk, so the next entry straddles the cap.
	big := strings.Repeat("a", int(testMaxBodyBytes))
	for i := int64(0); i < (testMaxTotalBytes/testMaxBodyBytes)-1; i++ {
		addEntry(big)
	}
	addEntry(strings.Repeat("b", 1024))
	addEntry(big)

	var total int
	for _, e := range buf.entries {
		total += len(e.Response.Content.Text)
	}
	if int64(total) > testMaxTotalBytes {
		t.Errorf("retained body text %d exceeds the cap budget %d", total, testMaxTotalBytes)
	}
	if last := buf.entries[len(buf.entries)-1]; last.Response.Content.Text != "" {
		t.Error("the entry that crosses the budget must drop its body text, not be retained whole")
	}
}

// TestBuffer_zeroValueUsesDefaultCaps asserts that a Buffer built as its zero value (var b Buffer,
// rather than through NewBuffer) still applies the default caps instead of the zero-value caps, which
// would capture no body and drop every payload.
func TestBuffer_zeroValueUsesDefaultCaps(t *testing.T) {
	var buf Buffer
	req, err := http.NewRequest(http.MethodGet, "http://ds.example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	const body = "hello"
	resp := &http.Response{Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
	buf.AddEntry(req, nil, false, resp, nil, time.Now(), time.Millisecond)

	if got := buf.entries[0].Response.Content.Text; got != body {
		t.Errorf("zero-value Buffer must capture like NewBuffer(); got body text %q, want %q", got, body)
	}
}

// TestSDKHARCaptureBuffer_retainedAccountsForDroppedError asserts that once an entry's payload is
// dropped for being over the total budget, what dropPayload leaves behind (the query error, kept
// because it's what makes an over-budget entry worth keeping) is still added to the running total --
// otherwise repeated over-budget entries retain error text that the budget never accounts for.
func TestSDKHARCaptureBuffer_retainedAccountsForDroppedError(t *testing.T) {
	buf := newBufferWithLimits(testMaxBodyBytes, testMaxTotalBytes)
	bigErr := strings.Repeat("e", int(testMaxTotalBytes))

	// First entry alone already exceeds the total budget, so its payload is dropped on arrival but its
	// error (small next to bigErr's own size once escaped) is kept.
	buf.AddQueryInteraction(querycapture.Interaction{Kind: querycapture.KindSQLQuery, Statement: "SELECT 1", Err: bigErr})

	var wantRetained int64
	for _, e := range buf.entries {
		wantRetained += e.payloadBytes()
	}
	if buf.retained != wantRetained {
		t.Errorf("buf.retained = %d, want %d (sum of what entries actually retain)", buf.retained, wantRetained)
	}
}
