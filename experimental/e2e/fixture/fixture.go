package fixture

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/storage"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/utils"
)

type RequestProcessor func(req *http.Request) *http.Request
type ResponseProcessor func(res *http.Response) *http.Response
type Matcher func(a *http.Request, b *http.Request) bool

type Fixture struct {
	processRequest  RequestProcessor
	processResponse ResponseProcessor
	match           Matcher
	store           storage.Storage
}

// NewFixture creates a new Fixture.
func NewFixture(store storage.Storage) *Fixture {
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
func (f *Fixture) Entries() []*storage.Entry {
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
// Note: Overriding the fixture's matcher will not affect the host matching behavior that can be configured in proxy.json.
func (f *Fixture) WithMatcher(matcher Matcher) {
	f.match = matcher
}

// Match compares incoming request to entries from the Fixture's Storage.
func (f *Fixture) Match(originalReq *http.Request) (string, *http.Response) {
	req := f.processRequest(originalReq)
	for _, entry := range f.store.Entries() {
		if f.match(entry.Request, req) {
			return entry.ID, f.processResponse(entry.Response)
		}
	}
	return "", nil
}

// DefaultMatcher is a default implementation of Matcher.
// It compares the request method, url, headers, and body to the stored request.
func DefaultMatcher(stored *http.Request, incoming *http.Request) bool {
	if stored.Method != incoming.Method {
		return false
	}

	if stored.URL.String() != incoming.URL.String() {
		return false
	}

	for name := range stored.Header {
		if stored.Header.Get(name) != incoming.Header.Get(name) {
			return false
		}
	}

	if stored.Body == nil && incoming.Body == nil {
		return true
	}

	storedBody, err := utils.ReadRequestBody(stored)
	if err != nil {
		return false
	}

	incomingBody, err := utils.ReadRequestBody(incoming)
	if err != nil {
		return false
	}

	return bytes.Equal(storedBody, incomingBody)
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
	b, err := utils.ReadRequestBody(processedReq)
	if err != nil {
		return processedReq
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	return processedReq
}

// DefaultProcessResponse is a default implementation of ProcessResponse.
// It removes the Set-Cookie header from the response.
func DefaultProcessResponse(res *http.Response) *http.Response {
	res.Header.Del("Set-Cookie")
	return res
}
