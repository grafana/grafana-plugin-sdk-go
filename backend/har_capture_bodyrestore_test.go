package backend

import (
	"errors"
	"io"
	"net/http"
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

	entry := buildSDKHAREntry(req, nil, resp, time.Now(), time.Millisecond)

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

	entry := buildSDKHAREntry(req, nil, &http.Response{Header: http.Header{}}, time.Now(), time.Millisecond)

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
