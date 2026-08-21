package backend

import (
	"context"
	"net/http"
	"net/textproto"
)

// EndpointCallCustomRoute friendly name for the call custom route endpoint/handler.
const EndpointCallCustomRoute Endpoint = "callCustomRoute"

// CustomRouteHandler is an EXPERIMENTAL service that handles HTTP-style calls to
// custom routes or kind subresources attached to a resource group/version.
// This is modeled after the App.CallCustomRoute method in grafana-app-sdk.
// This is EXPERIMENTAL and is a subject to change till Grafana 12.
type CustomRouteHandler interface {
	// CallCustomRoute handles a call to a custom route, streaming the response back via sender.
	CallCustomRoute(ctx context.Context, req *CallCustomRouteRequest, sender CallCustomRouteResponseSender) error
}

// FullIdentifier fully identifies the resource a custom route is attached to.
// It mirrors grafana-app-sdk resource.FullIdentifier. For a non-subresource route,
// the runner provides all the information it can: in practice that often means either
// Kind or Plural is present, but not necessarily both, and Name may be empty.
type FullIdentifier struct {
	Namespace string
	Name      string
	Group     string
	Version   string
	Kind      string
	Plural    string
}

// CallCustomRouteRequest represents a request to a custom route or kind subresource.
type CallCustomRouteRequest struct {
	// PluginContext the contextual information for the request.
	PluginContext PluginContext

	// Identifier fully identifies the resource the route is attached to.
	Identifier FullIdentifier

	// Path the path past the identifier information. For a subresource route this is
	// the subresource (e.g. `bar` in `foos/foo/bar`); for a non-subresource route it
	// is the path section past the namespace (or version, if the route is not namespaced).
	Path string

	// Method the forwarded HTTP method for the request.
	Method string

	// URL the forwarded HTTP URL for the request.
	URL string

	// Headers the forwarded HTTP headers for the request, if any.
	//
	// Recommended to use GetHTTPHeaders or GetHTTPHeader
	// since it automatically handles canonicalization of
	// HTTP header keys.
	Headers map[string][]string

	// Body the forwarded HTTP body for the request, if any.
	Body []byte
}

// SetHTTPHeader sets the header entries associated with key to the
// single element value. It replaces any existing values
// associated with key. The key is case-insensitive; it is
// canonicalized by textproto.CanonicalMIMEHeaderKey.
func (req *CallCustomRouteRequest) SetHTTPHeader(key, value string) {
	if req.Headers == nil {
		req.Headers = map[string][]string{}
	}

	req.Headers[key] = []string{value}
}

// DeleteHTTPHeader deletes the values associated with key.
// The key is case-insensitive; it is canonicalized by
// CanonicalHeaderKey.
func (req *CallCustomRouteRequest) DeleteHTTPHeader(key string) {
	if req.Headers == nil {
		return
	}

	for k := range req.Headers {
		if textproto.CanonicalMIMEHeaderKey(k) == textproto.CanonicalMIMEHeaderKey(key) {
			delete(req.Headers, k)
			break
		}
	}
}

// GetHTTPHeader gets the first value associated with the given key. If
// there are no values associated with the key, Get returns "".
// It is case-insensitive; textproto.CanonicalMIMEHeaderKey is
// used to canonicalize the provided key. Get assumes that all
// keys are stored in canonical form.
func (req *CallCustomRouteRequest) GetHTTPHeader(key string) string {
	return req.GetHTTPHeaders().Get(key)
}

// GetHTTPHeaders returns HTTP headers.
func (req *CallCustomRouteRequest) GetHTTPHeaders() http.Header {
	httpHeaders := http.Header{}

	for k, v := range req.Headers {
		for _, strVal := range v {
			httpHeaders.Add(k, strVal)
		}
	}

	return httpHeaders
}

// CallCustomRouteResponse represents a (streamed) response from a custom route call.
type CallCustomRouteResponse struct {
	// Status the HTTP response status.
	Status int

	// Headers the HTTP response headers.
	Headers map[string][]string

	// Body the HTTP response body.
	Body []byte
}

// CallCustomRouteResponseSender is used for sending custom route call responses.
type CallCustomRouteResponseSender interface {
	Send(*CallCustomRouteResponse) error
}

// CallCustomRouteResponseSenderFunc is an adapter to allow the use of
// ordinary functions as [CallCustomRouteResponseSender]. If f is a function
// with the appropriate signature, CallCustomRouteResponseSenderFunc(f) is a
// [CallCustomRouteResponseSender] that calls f.
type CallCustomRouteResponseSenderFunc func(resp *CallCustomRouteResponse) error

// Send calls fn(resp).
func (fn CallCustomRouteResponseSenderFunc) Send(resp *CallCustomRouteResponse) error {
	return fn(resp)
}

// CallCustomRouteHandlerFunc is an adapter to allow the use of
// ordinary functions as [CustomRouteHandler]. If f is a function
// with the appropriate signature, CallCustomRouteHandlerFunc(f) is a
// [CustomRouteHandler] that calls f.
type CallCustomRouteHandlerFunc func(ctx context.Context, req *CallCustomRouteRequest, sender CallCustomRouteResponseSender) error

// CallCustomRoute calls fn(ctx, req, sender).
func (fn CallCustomRouteHandlerFunc) CallCustomRoute(ctx context.Context, req *CallCustomRouteRequest, sender CallCustomRouteResponseSender) error {
	return fn(ctx, req, sender)
}

var _ ForwardHTTPHeaders = (*CallCustomRouteRequest)(nil)
