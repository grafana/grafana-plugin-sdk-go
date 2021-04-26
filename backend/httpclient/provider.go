package httpclient

import (
	"crypto/tls"
	"net/http"
)

// Provider provides abilities to create http.Client, http.RoundTripper and tls.Config.
type Provider interface {
	// New creates a new http.Client given provided options.
	New(opts *Options) (*http.Client, error)

	// GetTransport creates a new http.RoundTripper given provided options.
	GetTransport(opts *Options) (http.RoundTripper, error)

	// GetTLSConfig creates a new tls.Config given provided options.
	GetTLSConfig(opts *TLSOptions) (*tls.Config, error)
}

// DefaultProvider the default HTTP client provider implementation.
type DefaultProvider struct {
	Middlewares []Middleware
}

// NewProvider creates a new HTTP client provider.
// Optionally provide a list of middlewares that will be executed for each request.
// If no middlewares are provided the default middlewares will be used,
// BasicAuthenticationMiddleware and CustomHeadersMiddleware. If you provide
// middlewares no default middlewares will be used per default.
// Note: Middlewares will be executed in the same order as provided.
func NewProvider(middlewares ...Middleware) *DefaultProvider {
	if middlewares == nil {
		middlewares = append(middlewares, BasicAuthenticationMiddleware())
		middlewares = append(middlewares, CustomHeadersMiddleware())
	}

	return &DefaultProvider{
		Middlewares: middlewares,
	}
}

// New creates a new http.Client given provided options.
// If opts is nil the http.DefaultClient will be returned and no
// outgoing request middleware applied.
func (p *DefaultProvider) New(opts *Options) (*http.Client, error) {
	if opts == nil {
		return http.DefaultClient, nil
	}

	p.checkOpts(opts)
	return New(opts)
}

// GetTransport creates a new http.RoundTripper given provided options.
// If opts is nil the http.DefaultTransport will be returned and no
// outgoing request middleware applied.
func (p *DefaultProvider) GetTransport(opts *Options) (http.RoundTripper, error) {
	if opts == nil {
		return http.DefaultTransport, nil
	}

	p.checkOpts(opts)
	return GetTransport(opts)
}

// GetTLSConfig creates a new tls.Config given provided options.
func (p *DefaultProvider) GetTLSConfig(opts *TLSOptions) (*tls.Config, error) {
	return GetTLSConfig(opts)
}

func (p *DefaultProvider) checkOpts(opts *Options) {
	if opts.Middlewares == nil {
		opts.Middlewares = []Middleware{}
	}

	middlewares := make([]Middleware, len(p.Middlewares))
	copy(middlewares, p.Middlewares)
	middlewares = append(middlewares, opts.Middlewares...)
	opts.Middlewares = make([]Middleware, len(middlewares))
	copy(opts.Middlewares, middlewares)
}
