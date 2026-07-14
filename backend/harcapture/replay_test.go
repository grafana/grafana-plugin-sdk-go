package harcapture

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/storage"
)

// TestHARReplay_ThroughFixtureStorage captures a request/response with our middleware's HAR buffer,
// writes it as a .har file, then loads it through the real E2E fixture-proxy storage and confirms
// the recorded response is replayed for an equivalent request -- i.e. our HAR is genuinely usable
// by the existing replay tooling.
func TestHARReplay_ThroughFixtureStorage(t *testing.T) {
	const url = "https://api.example.com/query?a=1&b=2"
	const reqBody = `{"x":1}`
	const respBody = `{"ok":true}`

	// 1. capture a request/response pair via our middleware's buffer.
	capReq, err := http.NewRequest(http.MethodPost, url, strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	capReq.Proto = "HTTP/1.1"
	capReq.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	_, _ = rec.WriteString(respBody)
	capResp := rec.Result()
	capResp.Proto = "HTTP/1.1"

	buf := NewBuffer()
	buf.AddEntry(capReq, []byte(reqBody), false, capResp, nil, time.Now(), 3*time.Millisecond)
	harStr, err := buf.ToHARString()
	if err != nil {
		t.Fatal(err)
	}

	// 2. write it as a .har file.
	path := filepath.Join(t.TempDir(), "capture.har")
	if err := os.WriteFile(path, []byte(harStr), 0o600); err != nil {
		t.Fatal(err)
	}

	// 3. load it through the fixture-proxy storage (the same loader the proxy uses) and replay.
	store := storage.NewHARStorage(path)

	incoming, err := http.NewRequest(http.MethodPost, url, strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	incoming.Header.Set("Content-Type", "application/json")

	resp := store.Match(incoming)
	if resp == nil {
		t.Fatal("fixture-proxy storage did NOT match/replay our HAR entry")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("replayed status = %d, want 200", resp.StatusCode)
	}
	got, _ := io.ReadAll(resp.Body)
	if !bytes.Equal(got, []byte(respBody)) {
		t.Errorf("replayed body = %q, want %q", got, respBody)
	}
	t.Logf("REPLAY OK via fixture storage: status=%d body=%s", resp.StatusCode, got)
}
