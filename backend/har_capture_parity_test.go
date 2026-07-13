package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	chhar "github.com/chromedp/cdproto/har"
)

// TestHARParity_UnmarshalsIntoChromedpHAR verifies our hand-rolled HAR output parses into the
// canonical chromedp/cdproto/har model (what experimental/e2e/storage + the E2E fixture proxy use).
func TestHARParity_UnmarshalsIntoChromedpHAR(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://api.example.com/query?a=1&b=2", strings.NewReader(`{"x":1}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Proto = "HTTP/1.1"
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})

	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	rec.Header().Set("Set-Cookie", "sid=xyz")
	_, _ = rec.WriteString(`{"ok":true}`)
	resp := rec.Result()
	resp.Proto = "HTTP/1.1"

	buf := newSDKHARCaptureBuffer()
	buf.addEntry(req, resp, time.Now(), 5*time.Millisecond)
	s, err := buf.toHARString()
	if err != nil {
		t.Fatal(err)
	}

	var h chhar.HAR
	if err := json.Unmarshal([]byte(s), &h); err != nil {
		t.Fatalf("our HAR does NOT unmarshal into chromedp har.HAR: %v", err)
	}
	if h.Log == nil || len(h.Log.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %+v", h.Log)
	}
	e := h.Log.Entries[0]
	if e.Request == nil || e.Request.Method != "POST" {
		t.Fatalf("request method not parsed: %+v", e.Request)
	}
	if len(e.Request.Cookies) != 1 || e.Request.Cookies[0].Name != "session" {
		t.Errorf("request cookie not parsed: %+v", e.Request.Cookies)
	}
	if len(e.Response.Cookies) != 1 || e.Response.Cookies[0].Name != "sid" {
		t.Errorf("response cookie not parsed: %+v", e.Response.Cookies)
	}
	if e.Cache == nil {
		t.Errorf("cache object missing")
	}
	t.Logf("OUR HAR (round-trips into chromedp har.HAR):\n%s", s)
}
