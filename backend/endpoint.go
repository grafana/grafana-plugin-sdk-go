package backend

import "context"

// Endpoint used for defining names for endpoints/handlers.
type Endpoint string

// IsEmpty returns true if endpoint is not set/empty string.
func (e Endpoint) IsEmpty() bool {
	return e == ""
}

type endpointCtxKeyType struct{}

var endpointCtxKey = endpointCtxKeyType{}

// WithEndpoint adds endpoint to ctx.
func WithEndpoint(ctx context.Context, endpoint Endpoint) context.Context {
	return context.WithValue(ctx, endpointCtxKey, endpoint)
}

// EndpointFromContext extracts [Endpoint] from ctx if available, otherwise empty [Endpoint].
func EndpointFromContext(ctx context.Context) Endpoint {
	if ep := ctx.Value(endpointCtxKey); ep != nil {
		return ep.(Endpoint)
	}

	return Endpoint("")
}
