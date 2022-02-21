package storage

import "net/http"

// Entry represents a http.Request and http.Response pair.
type Entry struct {
	ID       string
	Request  *http.Request
	Response *http.Response
}

// Storage is an interface for storing Entry objects.
type Storage interface {
	Add(*http.Request, *http.Response)
	Delete(string) bool
	Load() error
	Save() error
	Entries() []*Entry
}
