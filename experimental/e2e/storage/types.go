package storage

import (
	"bytes"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/utils"
)

// Entry represents a http.Request and http.Response pair.
type Entry struct {
	Request  *http.Request
	Response *http.Response
}

// Match compares the given http.Request with the stored http.Request and returns the stored http.Response if a match is found.
func (e *Entry) Match(incoming *http.Request) *http.Response {
	if e.Request.Method != incoming.Method {
		return nil
	}

	if e.Request.URL.String() != incoming.URL.String() {
		return nil
	}

	for name := range e.Request.Header {
		if e.Request.Header.Get(name) != incoming.Header.Get(name) {
			return nil
		}
	}

	if e.Request.Body == nil && incoming.Body == nil {
		return nil
	}

	entryRequestBody, err := utils.ReadRequestBody(e.Request)
	if err != nil {
		return nil
	}

	incomingBody, err := utils.ReadRequestBody(incoming)
	if err != nil {
		return nil
	}

	if !bytes.Equal(entryRequestBody, incomingBody) {
		return nil
	}

	return e.Response
}

// Storage is an interface for storing Entry objects.
type Storage interface {
	Add(*http.Request, *http.Response)
	Delete(*http.Request) bool
	Load() error
	Save() error
	Entries() []*Entry
	Match(*http.Request) *http.Response
}
