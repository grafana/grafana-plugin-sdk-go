package backend

import (
	"context"
	"net/http"
)

// CallResourceRequest represents a request for a resource call.
type CallResourceRequest struct {
	ForwardHTTPHeaders
	PluginContext PluginContext
	Path          string
	Method        string
	URL           string
	Headers       map[string][]string
	Body          []byte
}

func (req *CallResourceRequest) SetHTTPHeader(key, value string) {
	if req.Headers == nil {
		req.Headers = map[string][]string{}
	}

	req.Headers[key] = []string{value}
}

func (req CallResourceRequest) GetHTTPHeader(key string) string {
	return req.GetHTTPHeaders().Get(key)
}

func (req CallResourceRequest) GetHTTPHeaders() http.Header {
	httpHeaders := http.Header{}

	for k, v := range req.Headers {
		for _, strVal := range v {
			httpHeaders.Add(k, strVal)
		}
	}

	return httpHeaders
}

// CallResourceResponse represents a response from a resource call.
type CallResourceResponse struct {
	Status  int
	Headers map[string][]string
	Body    []byte
}

// CallResourceResponseSender is used for sending resource call responses.
type CallResourceResponseSender interface {
	Send(*CallResourceResponse) error
}

// CallResourceHandler handles resource calls.
type CallResourceHandler interface {
	CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error
}

// CallResourceHandlerFunc is an adapter to allow the use of
// ordinary functions as backend.CallResourceHandler. If f is a function
// with the appropriate signature, CallResourceHandlerFunc(f) is a
// Handler that calls f.
type CallResourceHandlerFunc func(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error

// CallResource calls fn(ctx, req, sender).
func (fn CallResourceHandlerFunc) CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
	return fn(ctx, req, sender)
}
