// Package togglelist reads a comma-separated list of feature toggle names.
package togglelist

import "strings"

// List is a comma-separated list of feature toggle names.
type List struct {
	names string
}

// New returns the List of the comma-separated names.
func New(names string) List {
	return List{names: names}
}

// Contains reports whether name is one of the names in l.
func (l List) Contains(name string) bool {
	if l.names == "" {
		return false
	}
	rest := l.names
	for {
		n, after, found := strings.Cut(rest, ",")
		if n == name {
			return true
		}
		if !found {
			return false
		}
		rest = after
	}
}
