package e2e_test

import (
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e"
	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	t.Run("Append", func(t *testing.T) {
		t.Run("should add new request to store", func(t *testing.T) {
			proxy, client, s := setupProxy(e2e.ProxyModeAppend)
			defer s.Close()
			req, err := http.NewRequest("GET", srv.URL+"/foo", nil)
			require.NoError(t, err)
			res, err := client.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()
			require.Equal(t, "/foo", proxy.Fixture.Entries()[0].Request.URL.Path)
			require.Equal(t, 200, proxy.Fixture.Entries()[0].Response.StatusCode)
			require.Equal(t, 200, res.StatusCode)
			resBody, err := ioutil.ReadAll(res.Body)
			require.NoError(t, err)
			require.Equal(t, "/foo", string(resBody))
		})

		t.Run("should not add or modify existing request", func(t *testing.T) {
			var err error
			proxy, client, s := setupProxy(e2e.ProxyModeAppend)
			defer s.Close()
			// Add an existing request directly to the fixture
			req, err := http.NewRequest("GET", srv.URL+"/foo", nil)
			require.NoError(t, err)
			req.Header = make(http.Header)
			req.Body = ioutil.NopCloser(bytes.NewBuffer([]byte("bar")))
			res := &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString("bar")),
				Request:    req,
			}
			proxy.Fixture.Add(req, res)
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, "/foo", proxy.Fixture.Entries()[0].Request.URL.Path)
			require.Equal(t, 200, proxy.Fixture.Entries()[0].Response.StatusCode)
			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, "bar", string(body))
		})
	})

	t.Run("Overwrite", func(t *testing.T) {
		t.Run("should add new request to store", func(t *testing.T) {
			proxy, client, s := setupProxy(e2e.ProxyModeOverwrite)
			defer s.Close()
			req, err := http.NewRequest("GET", srv.URL+"/foo", nil)
			require.NoError(t, err)
			res, err := client.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()
			require.Equal(t, "/foo", proxy.Fixture.Entries()[0].Request.URL.Path)
			require.Equal(t, 200, proxy.Fixture.Entries()[0].Response.StatusCode)
			require.Equal(t, 200, res.StatusCode)
			resBody, err := ioutil.ReadAll(res.Body)
			require.NoError(t, err)
			require.Equal(t, "/foo", string(resBody))
		})

		t.Run("should replace existing request", func(t *testing.T) {
			var err error
			proxy, client, s := setupProxy(e2e.ProxyModeOverwrite)
			defer s.Close()
			// Add an existing request directly to the fixture
			req, err := http.NewRequest("GET", srv.URL+"/foo", nil)
			require.NoError(t, err)
			req.Header = make(http.Header)
			req.Body = ioutil.NopCloser(bytes.NewBuffer([]byte("bar")))
			res := &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString("bar")),
				Request:    req,
			}
			proxy.Fixture.Add(req, res)
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, "/foo", proxy.Fixture.Entries()[0].Request.URL.Path)
			require.Equal(t, 200, proxy.Fixture.Entries()[0].Response.StatusCode)
			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, "/foo", string(body))
		})
	})
}

// ignoring the G402 error here because this proxy is only used for testing
// nolint:gosec
var acceptAllCerts = &tls.Config{InsecureSkipVerify: true}

type pathEcho struct{}

func (pathEcho) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	_, err := io.WriteString(w, req.URL.Path)
	if err != nil {
		panic(err)
	}
}

var srv = httptest.NewServer(pathEcho{})

func setupProxy(mode e2e.ProxyMode) (proxy *e2e.Proxy, client *http.Client, server *httptest.Server) {
	store := newFakeStorage()
	fixture := e2e.NewFixture(store)
	proxy = e2e.NewProxy(mode, fixture, ":9999")
	server = httptest.NewServer(proxy.Server)
	proxyURL, err := url.Parse(server.URL)
	if err != nil {
		panic(err)
	}
	tr := &http.Transport{TLSClientConfig: acceptAllCerts, Proxy: http.ProxyURL(proxyURL)}
	client = &http.Client{Transport: tr}
	return
}
