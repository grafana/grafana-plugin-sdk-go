package e2eproxy_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2eproxy"
	"github.com/stretchr/testify/require"
)

func TestHARStorage(t *testing.T) {
	t.Run("Add", func(t *testing.T) {
		t.Run("should add a new entry to the storage", func(t *testing.T) {
			s := e2eproxy.NewHARStorage("testdata/example_add.har")
			req, err := http.NewRequest("GET", "http://example.com/", nil)
			require.NoError(t, err)
			res := &http.Response{
				StatusCode: 404,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}
			s.Add(req, res)
			require.Len(t, s.Entries(), 1)
			require.Equal(t, req.URL.String(), s.Entries()[0].Request.URL.String())
			require.Equal(t, res.Status, s.Entries()[0].Response.Status)
		})
	})

	t.Run("Load", func(t *testing.T) {
		t.Run("should load the HAR from disk", func(t *testing.T) {
			s := e2eproxy.NewHARStorage("testdata/example.har")
			req := s.Entries()[0].Request
			res := s.Entries()[0].Response
			require.Equal(t, "https://grafana.com/api/plugins", req.URL.String())
			require.Len(t, req.Header, 13)
			require.Equal(t, http.MethodGet, req.Method)
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Len(t, res.Header, 14)
			require.Equal(t, int64(2), res.ContentLength)

			req = s.Entries()[1].Request
			res = s.Entries()[1].Response
			require.Equal(t, "https://grafana.com/favicon.ico", req.URL.String())
			require.Len(t, req.Header, 6)
			require.Equal(t, http.MethodGet, req.Method)
			require.Equal(t, 0, res.StatusCode)
			require.Len(t, res.Header, 0)
			require.Equal(t, int64(0), res.ContentLength)
		})
	})

	t.Run("Save", func(t *testing.T) {
		t.Run("should save", func(t *testing.T) {
			source := e2eproxy.NewHARStorage("testdata/example.har")
			f, err := os.CreateTemp("", "example_*.har")
			require.NoError(t, err)
			dest := e2eproxy.NewHARStorage(f.Name())
			for _, entry := range source.Entries() {
				dest.Add(entry.Request, entry.Response)
			}
			err = dest.Save()
			require.NoError(t, err)
			sourceData, err := os.ReadFile("testdata/example.har")
			require.NoError(t, err)
			destData, err := os.ReadFile(f.Name())
			require.NoError(t, err)
			// we can't compare the two HAR files directly because header maps are not ordered
			require.Equal(t, len(sourceData), len(destData))
			err = os.Remove(f.Name())
			require.NoError(t, err)
		})
	})
}
