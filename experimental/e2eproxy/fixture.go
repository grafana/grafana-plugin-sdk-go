package e2eproxy

import (
	"bytes"
	"io"
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

func (f *Fixture) Match(orignalReq *http.Request) *http.Response {
	req := f.processRequest(orignalReq)
	for _, entry := range f.store.Entries() {
		if f.match(entry.Request, req) {
			return f.processResponse(entry.Response)
		}
	}
	return nil
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

	aBody, err := io.ReadAll(a.Body)
	if err != nil {
		return false
	}

	bBody, err := io.ReadAll(b.Body)
	if err != nil {
		return false
	}

	if !bytes.Equal(aBody, bBody) {
		return false
	}

	return true
}

// DefaultProcessRequest is a default implementation of ProcessRequest.
// It returns the original unmodified request.
func DefaultProcessRequest(req *http.Request) *http.Request {
	return req
}

// DefaultProcessResponse is a default implementation of ProcessResponse.
// It returns the original unmodified response.
func DefaultProcessResponse(res *http.Response) *http.Response {
	return res
}
