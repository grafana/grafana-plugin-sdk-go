package e2eproxy

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/chromedp/cdproto/har"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Entry struct {
	Request  *http.Request
	Response *http.Response
}

type Storage interface {
	Add(*http.Request, *http.Response)
	Load() error
	Save() error
	Entries() []*Entry
}

type HARStorage struct {
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
		backend.Logger.Debug("Unable to load HAR", "path", path)
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
	for name, value := range req.Header {
		reqHeaders = append(reqHeaders, &har.NameValuePair{Name: name, Value: value[0]})
	}

	resHeaders := make([]*har.NameValuePair, 0)
	for name, value := range res.Header {
		resHeaders = append(resHeaders, &har.NameValuePair{Name: name, Value: value[0]})
	}

	queryString := make([]*har.NameValuePair, 0)
	for name, value := range req.URL.Query() {
		queryString = append(queryString, &har.NameValuePair{Name: name, Value: value[0]})
	}

	if req.Body != nil {
		reqBody, err = io.ReadAll(req.Body)
		defer req.Body.Close()
		if err != nil {
			backend.Logger.Error("Failed to read request body", "err", err)
		}
	}

	if res.Body != nil {
		resBody, err = io.ReadAll(res.Body)
		defer res.Body.Close()
		if err != nil {
			backend.Logger.Error("Failed to read response body", "err", err)
		}
	}

	cookies := make([]*har.Cookie, 0)
	for _, cookie := range res.Cookies() {
		cookies = append(cookies, &har.Cookie{Name: cookie.Name, Value: cookie.Value})
	}

	s.har.Log.Entries = append(s.har.Log.Entries, &har.Entry{
		Request: &har.Request{
			Method:      req.Method,
			HTTPVersion: req.Proto,
			URL:         req.URL.String(),
			Headers:     reqHeaders,
			QueryString: queryString,
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
			BodySize:    res.ContentLength,
			Cookies:     cookies,
			RedirectURL: res.Header.Get("Location"),
			Content: &har.Content{
				Size:     res.ContentLength,
				MimeType: res.Header.Get("Content-Type"),
				Text:     string(resBody),
			},
		},
	})
}

// Add converts the http.Request and http.Response to a har.Entry and adds it to the Fixture.
func (s *HARStorage) Entries() []*Entry {
	entries := make([]*Entry, len(s.har.Log.Entries))

	for i, e := range s.har.Log.Entries {
		postData := ""
		if (e.Request.PostData != nil) && (e.Request.PostData.Text != "") {
			postData = e.Request.PostData.Text
		}
		req, err := http.NewRequest(e.Request.Method, e.Request.URL, ioutil.NopCloser(strings.NewReader(postData)))
		if err != nil {
			continue
		}

		req.Header = make(http.Header)
		for _, header := range e.Request.Headers {
			req.Header.Add(header.Name, header.Value)
		}

		res := &http.Response{
			StatusCode:    int(e.Response.Status),
			Status:        e.Response.StatusText,
			Proto:         e.Response.HTTPVersion,
			Header:        make(http.Header),
			Body:          ioutil.NopCloser(strings.NewReader(e.Response.Content.Text)),
			ContentLength: e.Response.Content.Size,
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

func (s *HARStorage) Save() error {
	raw, err := s.har.MarshalJSON()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.path, raw, 0600)
}

func (s *HARStorage) Load() error {
	raw, err := ioutil.ReadFile(s.path)
	if err != nil {
		return err
	}
	return s.har.UnmarshalJSON(raw)
}
