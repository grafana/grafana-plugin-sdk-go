package backend

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
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

	entry := buildSDKHAREntry(req, nil, resp, nil, time.Now(), time.Millisecond)

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

	entry := buildSDKHAREntry(req, nil, &http.Response{Header: http.Header{}}, nil, time.Now(), time.Millisecond)

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

	entry := buildSDKHAREntry(req, nil, resp, nil, time.Now(), time.Millisecond)

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

	entry := buildSDKHAREntry(req, nil, nil, rtErr, time.Now(), time.Millisecond)

	if entry.Request.URL != "http://ds.example.com/q" {
		t.Errorf("failed request must still be captured, got URL %q", entry.Request.URL)
	}
	if entry.Response.Status != 0 {
		t.Errorf("no-response entry must have zero status, got %d", entry.Response.Status)
	}
	if !strings.Contains(entry.Comment, "connection refused") {
		t.Errorf("transport error must be recorded in Comment, got %q", entry.Comment)
	}
}

// TestSDKHARCaptureBuffer_totalSizeCap asserts that once the cumulative retained body budget is
// exceeded, later entries keep their metadata/sizes but drop the body text.
func TestSDKHARCaptureBuffer_totalSizeCap(t *testing.T) {
	buf := newSDKHARCaptureBuffer()
	big := strings.Repeat("a", maxCapturedBodyBytes) // one per-body-capped chunk each

	// Enough entries to blow past the total budget.
	for i := 0; i < (maxCapturedTotalBytes/maxCapturedBodyBytes)+2; i++ {
		req, err := http.NewRequest(http.MethodGet, "http://ds.example.com", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp := &http.Response{Header: http.Header{}, Body: io.NopCloser(strings.NewReader(big))}
		buf.addEntry(req, nil, resp, nil, time.Now(), time.Millisecond)
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
	if total > maxCapturedTotalBytes+2*maxCapturedBodyBytes {
		t.Errorf("retained body text %d exceeds the cap budget", total)
	}
	if !droppedText {
		t.Error("expected later entries to drop body text once over the total budget")
	}
	if !keptTrueSize {
		t.Error("true body size must be preserved even when text is dropped")
	}
}
