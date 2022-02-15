package e2e

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

type RequestProcessor func(req *http.Request) *http.Request
type ResponseProcessor func(res *http.Response) *http.Response
type Matcher func(a *http.Request, b *http.Request) bool

type Fixture struct {
	processRequest  RequestProcessor
	processResponse ResponseProcessor
	match           Matcher
	store           Storage
}

// NewFixture creates a new Fixture.
func NewFixture(store Storage) *Fixture {
	return &Fixture{
		processRequest:  DefaultProcessRequest,
		processResponse: DefaultProcessResponse,
		match:           DefaultMatcher,
		store:           store,
	}
}

// Add processes the http.Request and http.Response with the Fixture's RequestProcessor and ResponseProcessor and adds them to the Fixure's Storage.
func (f *Fixture) Add(originalReq *http.Request, originalRes *http.Response) {
	req := f.processRequest(originalReq)
	res := f.processResponse(originalRes)
	defer res.Body.Close()
	f.store.Add(req, res)
}

// Delete deletes the entry with the given ID from the Fixture's Storage.
func (f *Fixture) Delete(id string) bool {
	return f.store.Delete(id)
}

// Entries returns the entries from the Fixture's Storage.
func (f *Fixture) Entries() []*Entry {
	return f.store.Entries()
}

// Save saves the current state of the Fixture's Storage.
func (f *Fixture) Save() error {
	return f.store.Save()
}

// WithRequestProcessor sets the RequestProcessor for the Fixture.
func (f *Fixture) WithRequestProcessor(processRequest RequestProcessor) {
	f.processRequest = processRequest
}

// WithResponseProcessor sets the ResponseProcessor for the Fixture.
func (f *Fixture) WithResponseProcessor(processResponse ResponseProcessor) {
	f.processResponse = processResponse
}

// WithMatcher sets the Matcher for the Fixture.
func (f *Fixture) WithMatcher(matcher Matcher) {
	f.match = matcher
}

// Match compares incoming request to entries from the Fixture's Storage.
func (f *Fixture) Match(orignalReq *http.Request) (string, *http.Response) {
	req := f.processRequest(orignalReq)
	for _, entry := range f.store.Entries() {
		if f.match(entry.Request, req) {
			return entry.ID, f.processResponse(entry.Response)
		}
	}
	return "", nil
}

// DefaultMatcher is a default implementation of Matcher.
func DefaultMatcher(a *http.Request, b *http.Request) bool {
	if a.Method != b.Method {
		return false
	}

	if a.URL.String() != b.URL.String() {
		return false
	}

	for name := range a.Header {
		if a.Header.Get(name) != b.Header.Get(name) {
			return false
		}
	}

	if a.Body == nil && b.Body == nil {
		return true
	}

	aBody, err := io.ReadAll(a.Body)
	if err != nil {
		return false
	}
	a.Body = ioutil.NopCloser(bytes.NewBuffer(aBody))

	bBody, err := io.ReadAll(b.Body)
	if err != nil {
		return false
	}
	b.Body = ioutil.NopCloser(bytes.NewBuffer(bBody))

	return bytes.Equal(aBody, bBody)
}

// DefaultProcessRequest is a default implementation of ProcessRequest.
// It removes the Date, Cookie, Authorization, and User-Agent headers.
func DefaultProcessRequest(req *http.Request) *http.Request {
	processedReq := req.Clone(req.Context())
	processedReq.Header.Del("Date")
	processedReq.Header.Del("Coookie")
	processedReq.Header.Del("Authorization")
	processedReq.Header.Del("User-Agent")
	if processedReq.Body == nil {
		return processedReq
	}
	b, err := io.ReadAll(processedReq.Body)
	if err != nil {
		return processedReq
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	processedReq.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	return processedReq
}

// DefaultProcessResponse is a default implementation of ProcessResponse.
// It returns the original unmodified response.
func DefaultProcessResponse(res *http.Response) *http.Response {
	return res
}
