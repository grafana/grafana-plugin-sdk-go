package e2e_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e"
	"github.com/stretchr/testify/require"
)

func TestFixtureAdd(t *testing.T) {
	t.Run("should add request and response to fixture store", func(t *testing.T) {
		req, res := setup()
		defer req.Body.Close()
		defer res.Body.Close()
		store := newFakeStorage()
		f := e2e.NewFixture(store)
		require.Equal(t, 0, len(f.Entries()))
		f.Add(req, res)
		require.Equal(t, 1, len(f.Entries()))
		require.Equal(t, "http://example.com", f.Entries()[0].Request.URL.String())
		require.Equal(t, 200, f.Entries()[0].Response.StatusCode)
	})

	t.Run("should apply request processor", func(t *testing.T) {
		req, res := setup()
		defer req.Body.Close()
		defer res.Body.Close()
		store := newFakeStorage()
		f := e2e.NewFixture(store)
		f.WithRequestProcessor(func(req *http.Request) *http.Request {
			req.URL.Path = "/example"
			return req
		})
		require.Equal(t, 0, len(f.Entries()))
		f.Add(req, res)
		require.Equal(t, 1, len(f.Entries()))
		require.Equal(t, "http://example.com/example", f.Entries()[0].Request.URL.String())
	})

	t.Run("should apply response processor", func(t *testing.T) {
		req, res := setup()
		defer req.Body.Close()
		defer res.Body.Close()
		store := newFakeStorage()
		f := e2e.NewFixture(store)
		f.WithResponseProcessor(func(res *http.Response) *http.Response {
			res.StatusCode = 201
			return res
		})
		require.Equal(t, 0, len(f.Entries()))
		f.Add(req, res)
		require.Equal(t, 1, len(f.Entries()))
		require.Equal(t, 201, f.Entries()[0].Response.StatusCode)
	})

	t.Run("should apply response processor", func(t *testing.T) {
		req, res := setup()
		defer req.Body.Close()
		defer res.Body.Close()
		store := newFakeStorage()
		f := e2e.NewFixture(store)
		f.WithResponseProcessor(func(res *http.Response) *http.Response {
			res.StatusCode = 201
			return res
		})
		require.Equal(t, 0, len(f.Entries()))
		f.Add(req, res)
		require.Equal(t, 1, len(f.Entries()))
		require.Equal(t, 201, f.Entries()[0].Response.StatusCode)
	})
}

func TestFixtureMatch(t *testing.T) {
	t.Run("should match request and return request ID and response", func(t *testing.T) {
		store := newFakeStorage()
		_ = store.Load()
		f := e2e.NewFixture(store)
		f.WithMatcher(func(a, b *http.Request) bool {
			return true
		})
		id, res := f.Match(store.entries[0].Request)
		defer res.Body.Close()
		require.Equal(t, store.entries[0].ID, id)
		require.Equal(t, 200, res.StatusCode)
	})

	t.Run("should not match", func(t *testing.T) {
		store := newFakeStorage()
		_ = store.Load()
		f := e2e.NewFixture(store)
		f.WithMatcher(func(a, b *http.Request) bool {
			return false
		})
		id, res := f.Match(store.entries[0].Request) // nolint:bodyclose
		require.Equal(t, "", id)
		require.Nil(t, res)
	})

	t.Run("default matcher", func(t *testing.T) {
		t.Run("should match", func(t *testing.T) {
			store := newFakeStorage()
			_ = store.Load()
			f := e2e.NewFixture(store)
			req, resp := setup()
			defer resp.Body.Close()
			_, res := f.Match(req)
			defer res.Body.Close()
			require.NotNil(t, res)
		})

		t.Run("should not return response if req method does not match", func(t *testing.T) {
			store := newFakeStorage()
			_ = store.Load()
			f := e2e.NewFixture(store)
			req, resp := setup()
			defer resp.Body.Close()
			req.Method = "PUT"
			_, res := f.Match(req) //nolint:bodyclose
			require.Nil(t, res)
		})

		t.Run("should not return response if URL does not match", func(t *testing.T) {
			store := newFakeStorage()
			_ = store.Load()
			f := e2e.NewFixture(store)
			req, resp := setup()
			defer resp.Body.Close()
			req.URL.Path = "/foo"
			_, res := f.Match(req) //nolint:bodyclose
			require.Nil(t, res)
		})

		t.Run("should not return response if headers do not match", func(t *testing.T) {
			store := newFakeStorage()
			_ = store.Load()
			f := e2e.NewFixture(store)
			req, resp := setup()
			defer resp.Body.Close()
			req.Header.Set("Content-Type", "plain/text")
			_, res := f.Match(req) //nolint:bodyclose
			require.Nil(t, res)
		})

		t.Run("should not return response if request body does not match", func(t *testing.T) {
			store := newFakeStorage()
			_ = store.Load()
			f := e2e.NewFixture(store)
			req, resp := setup()
			defer resp.Body.Close()
			req.Body = ioutil.NopCloser(bytes.NewBufferString("foo"))
			_, res := f.Match(req) // nolint:bodyclose
			require.Nil(t, res)
		})
	})
}

func TestDefaultProcessRequest(t *testing.T) {
	t.Run("should remove headers as expected", func(t *testing.T) {
		req, resp := setup()
		defer resp.Body.Close()
		req.Header.Add("Date", "foo")
		req.Header.Add("Coookie", "bar")
		req.Header.Add("Authorization", "baz")
		req.Header.Add("User-Agent", "qux")
		req.Header.Add("Content-Type", "application/json")
		require.Equal(t, 5, len(req.Header))
		proccessedReq := e2e.DefaultProcessRequest(req)
		require.Equal(t, 1, len(proccessedReq.Header))
		require.Equal(t, "application/json", proccessedReq.Header.Get("Content-Type"))
	})
}

func TestFixtureDelete(t *testing.T) {
	t.Run("should delete fixture from storage", func(t *testing.T) {
		store := newFakeStorage()
		_ = store.Load()
		f := e2e.NewFixture(store)
		require.Equal(t, 1, len(f.Entries()))
		f.Delete(f.Entries()[0].ID)
		require.Equal(t, 0, len(f.Entries()))
	})
}

func setup() (*http.Request, *http.Response) {
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
	entries []*e2e.Entry
	err     error
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{
		entries: make([]*e2e.Entry, 0),
		err:     nil,
	}
}

func (s *fakeStorage) Add(req *http.Request, res *http.Response) {
	s.entries = append(s.entries, &e2e.Entry{
		ID:       uuid.New().String(),
		Request:  req,
		Response: res,
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
	s.entries = make([]*e2e.Entry, 0)
	req, res := setup()
	defer res.Body.Close()
	s.entries = append(s.entries, &e2e.Entry{
		ID:       uuid.New().String(),
		Request:  req,
		Response: res,
	})
	return s.err
}

func (s *fakeStorage) Save() error {
	return s.err
}

func (s *fakeStorage) Entries() []*e2e.Entry {
	return s.entries
}
