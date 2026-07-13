package backend

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/storage"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/utils"
)

// TestHARReplay_RealCapturedFile replays an actual diagnostics-bundle traffic.har through the
// E2E fixture-proxy storage. Point HAR_REPLAY_FILE at a captured traffic.har to run it.
func TestHARReplay_RealCapturedFile(t *testing.T) {
	path := os.Getenv("HAR_REPLAY_FILE")
	if path == "" {
		t.Skip("set HAR_REPLAY_FILE to a captured traffic.har to run this replay test")
	}

	store := storage.NewHARStorage(path)
	entries := store.Entries()
	if len(entries) == 0 {
		t.Fatalf("no entries loaded from %s", path)
	}
	t.Logf("loaded %d entr(ies) from captured HAR", len(entries))

	e := entries[0]
	body, err := utils.ReadRequestBody(e.Request)
	if err != nil {
		t.Fatal(err)
	}

	// Build an incoming request mirroring the captured one and replay it. The method/URL come from
	// a local HAR fixture the developer points the test at, not from any live request.
	incoming, err := http.NewRequest(e.Request.Method, e.Request.URL.String(), bytes.NewReader(body)) //nolint:gosec // G704: URL is from a local test fixture, not user input
	if err != nil {
		t.Fatal(err)
	}
	incoming.Header = e.Request.Header.Clone()

	resp := store.Match(incoming)
	if resp == nil {
		t.Fatalf("fixture storage did NOT replay captured request: %s %s", e.Request.Method, e.Request.URL)
	}
	defer func() { _ = resp.Body.Close() }()
	got, _ := io.ReadAll(resp.Body)
	t.Logf("REPLAYED %s %s -> status=%d, %d bytes of response body", e.Request.Method, e.Request.URL, resp.StatusCode, len(got))
	if resp.StatusCode != http.StatusOK {
		t.Errorf("replayed status = %d, want 200", resp.StatusCode)
	}
}
