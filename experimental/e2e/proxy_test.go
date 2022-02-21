package e2e_test

import (
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/config"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/fixture"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/storage"
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
	fixture := fixture.NewFixture(store)
	config, err := config.LoadConfig("proxy.json")
	if err != nil {
		panic(err)
	}
	proxy = e2e.NewProxy(mode, fixture, config)
	server = httptest.NewServer(proxy.Server)
	proxyURL, err := url.Parse(server.URL)
	if err != nil {
		panic(err)
	}
	tr := &http.Transport{TLSClientConfig: acceptAllCerts, Proxy: http.ProxyURL(proxyURL)}
	client = &http.Client{Transport: tr}
	return
}

func setupFixture() (*http.Request, *http.Response) {
	req, err := http.NewRequest("POST", "http://example.com", ioutil.NopCloser(strings.NewReader("test")))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Content-Type", "application/json")
	res := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       ioutil.NopCloser(strings.NewReader("{\"foo\":\"bar\"}")),
	}
	return req, res
}

type fakeStorage struct {
	entries []*storage.Entry
	err     error
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{
		entries: make([]*storage.Entry, 0),
		err:     nil,
	}
}

func (s *fakeStorage) Add(req *http.Request, res *http.Response) {
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	res.Body = io.NopCloser(bytes.NewBuffer(resBody))
	resCopy := *res
	resCopy.Body = ioutil.NopCloser(bytes.NewBuffer(resBody))
	s.entries = append(s.entries, &storage.Entry{
		ID:       uuid.New().String(),
		Request:  req,
		Response: &resCopy,
	})
}

func (s *fakeStorage) Delete(id string) bool {
	for i, entry := range s.entries {
		if entry.ID == id {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			return true
		}
	}
	return false
}

func (s *fakeStorage) Load() error {
	s.entries = make([]*storage.Entry, 0)
	req, res := setupFixture()
	defer res.Body.Close()
	s.entries = append(s.entries, &storage.Entry{
		ID:       uuid.New().String(),
		Request:  req,
		Response: res,
	})
	return s.err
}

func (s *fakeStorage) Save() error {
	return s.err
}

func (s *fakeStorage) Entries() []*storage.Entry {
	return s.entries
}
