package mcp

import "context"

// ResourceReader returns the resource body and its MIME type. The default
// MIME type from the Resource struct is used if the reader returns "".
type ResourceReader func(ctx context.Context) (body []byte, mimeType string, err error)

// Resource is a registerable MCP resource.
type Resource struct {
	URI         string
	Name        string
	Description string
	MIMEType    string
	Reader      ResourceReader
}

func (s *Server) RegisterResource(r Resource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.resources {
		if existing.URI == r.URI {
			s.resources[i] = r
			return
		}
	}
	s.resources = append(s.resources, r)
}
