package e2eproxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/har"
)

// Entry represents a http.Request and http.Response pair.
type Entry struct {
	Request  *http.Request
	Response *http.Response
}

// Storage is an interface for storing Entry objects.
type Storage interface {
	Add(*http.Request, *http.Response)
	Load() error
	Save() error
	Entries() []*Entry
}

// HARStorage is a Storage implementation that stores requests and responses in HAR format on disk.
type HARStorage struct {
	lock sync.Mutex
	path string
	har  *har.HAR
}

// NewHARStorage creates a new HARStorage.
func NewHARStorage(path string) *HARStorage {
	storage := &HARStorage{
		path: path,
		har:  &har.HAR{},
	}
	if err := storage.Load(); err != nil {
		fmt.Println("Unable to load HAR", "path", path)
		storage.har.Log = &har.Log{
			Version: "1.2",
			Creator: &har.Creator{
				Name:    "grafana-plugin-sdk-go",
				Version: "experimental",
			},
			Entries: make([]*har.Entry, 0),
		}
	}
	return storage
}

// Add converts the http.Request and http.Response to a har.Entry and adds it to the Fixture.
func (s *HARStorage) Add(req *http.Request, res *http.Response) {
	var (
		err     error
		reqBody []byte
		resBody []byte
	)

	reqHeaders := make([]*har.NameValuePair, 0)
	for name, value := range req.Header.Clone() {
		reqHeaders = append(reqHeaders, &har.NameValuePair{Name: name, Value: value[0]})
	}

	resHeaders := make([]*har.NameValuePair, 0)
	for name, value := range res.Header.Clone() {
		resHeaders = append(resHeaders, &har.NameValuePair{Name: name, Value: value[0]})
	}

	queryString := make([]*har.NameValuePair, 0)
	for name, value := range req.URL.Query() {
		queryString = append(queryString, &har.NameValuePair{Name: name, Value: value[0]})
	}

	if req.Body != nil {
		reqBody, err = io.ReadAll(req.Body)
		if err != nil {
			fmt.Println("Failed to read request body", "err", err)
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
	}

	if res.Body != nil {
		resBody, err = io.ReadAll(res.Body)
		if err != nil {
			fmt.Println("Failed to read response body", "err", err)
		}
		res.Body = ioutil.NopCloser(bytes.NewReader(resBody))
	}

	reqCookies := make([]*har.Cookie, 0)
	for _, cookie := range req.Cookies() {
		reqCookies = append(reqCookies, &har.Cookie{Name: cookie.Name, Value: cookie.Value})
	}

	resCookies := make([]*har.Cookie, 0)
	for _, cookie := range res.Cookies() {
		resCookies = append(resCookies, &har.Cookie{Name: cookie.Name, Value: cookie.Value})
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.har.Log.Entries = append(s.har.Log.Entries, &har.Entry{
		Request: &har.Request{
			Method:      req.Method,
			HTTPVersion: req.Proto,
			URL:         req.URL.String(),
			Headers:     reqHeaders,
			QueryString: queryString,
			Cookies:     reqCookies,
			BodySize:    int64(len(reqBody)),
			PostData: &har.PostData{
				MimeType: req.Header.Get("Content-Type"),
				Text:     string(reqBody),
			},
		},
		Response: &har.Response{
			Status:      int64(res.StatusCode),
			StatusText:  res.Status,
			HTTPVersion: res.Proto,
			Headers:     resHeaders,
			HeadersSize: -1,
			BodySize:    int64(len(resBody)),
			Cookies:     resCookies,
			RedirectURL: res.Header.Get("Location"),
			Content: &har.Content{
				Size:     int64(len(resBody)),
				MimeType: res.Header.Get("Content-Type"),
				Text:     string(resBody),
			},
		},
	})
}

// Entries converts HAR entries to a slice of Entry (http.Request and http.Response pairs).
func (s *HARStorage) Entries() []*Entry {
	entries := make([]*Entry, len(s.har.Log.Entries))
	s.lock.Lock()
	defer s.lock.Unlock()

	for i, e := range s.har.Log.Entries {
		postData := ""
		if e.Request.PostData != nil {
			postData = e.Request.PostData.Text
		}
		req, err := http.NewRequest(e.Request.Method, e.Request.URL, nil)
		if err != nil {
			fmt.Println("Failed to create request", "err", err)
			continue
		}
		req.Body = ioutil.NopCloser(strings.NewReader(postData))
		req.ContentLength = e.Request.BodySize
		req.Header = make(http.Header)
		for _, header := range e.Request.Headers {
			req.Header.Add(header.Name, header.Value)
		}

		bodyReq := req.Clone(context.Background())
		bodyReq.Body = ioutil.NopCloser(strings.NewReader(postData))
		res := &http.Response{
			StatusCode:    int(e.Response.Status),
			Status:        e.Response.StatusText,
			Proto:         e.Response.HTTPVersion,
			Header:        make(http.Header),
			Body:          ioutil.NopCloser(strings.NewReader(e.Response.Content.Text)),
			ContentLength: int64(len(e.Response.Content.Text)),
			Request:       bodyReq,
		}

		for _, header := range e.Response.Headers {
			res.Header.Add(header.Name, header.Value)
		}

		entries[i] = &Entry{
			Request:  req,
			Response: res,
		}
	}

	return entries
}

// Save writes the HAR to disk.
func (s *HARStorage) Save() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	err := os.MkdirAll(filepath.Dir(s.path), os.ModePerm)
	if err != nil {
		return err
	}
	raw, err := s.har.MarshalJSON()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.path, raw, 0600)
}

// Load reads the HAR from disk.
func (s *HARStorage) Load() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	raw, err := ioutil.ReadFile(s.path)
	if err != nil {
		return err
	}
	return s.har.UnmarshalJSON(raw)
}
