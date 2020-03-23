package backend

import "context"

type CallResourceRequest struct {
	PluginConfig PluginConfig
	Path         string
	Method       string
	URL          string
	Headers      map[string][]string
	Body         []byte
	User         *User
}

type CallResourceResponse struct {
	Status  int
	Headers map[string][]string
	Body    []byte
}

// CallResourceResponseSender used for sending resource call responses.
type CallResourceResponseSender interface {
	Send(*CallResourceResponse) error
}

// CallResourceHandler handles resource calls.
type CallResourceHandler interface {
	CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error
}
