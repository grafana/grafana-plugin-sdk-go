package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
)

// New creates a new http.Client.
// If opts is nil the http.DefaultClient will be returned.
// If no middlewares are provided the DefaultMiddlewares will be used. If you
// provide middlewares you have to manually add the DefaultMiddlewares for it to be
// enabled.
// Note: Middlewares will be executed in the same order as provided.
func New(opts *Options) (*http.Client, error) {
	if opts == nil {
		return http.DefaultClient, nil
	}

	checkOpts(opts)
	transport, err := GetTransport(opts)
	if err != nil {
		return nil, err
	}

	c := http.Client{
		Transport: transport,
		Timeout:   opts.Timeouts.Timeout,
	}

	return &c, nil
}

// GetTransport creates a new http.RoundTripper given provided options.
// If opts is nil the http.DefaultTransport will be returned.
// If no middlewares are provided the DefaultMiddlewares() will be used. If you
// provide middlewares you have to manually add the DefaultMiddlewares() for it to be
// enabled.
// Note: Middlewares will be executed in the same order as provided.
func GetTransport(opts *Options) (http.RoundTripper, error) {
	if opts == nil {
		return http.DefaultTransport, nil
	}

	checkOpts(opts)
	tlsConfig, err := GetTLSConfig(opts.TLS)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   opts.Timeouts.Timeout,
			KeepAlive: opts.Timeouts.KeepAlive,
		}).DialContext,
		TLSHandshakeTimeout:   opts.Timeouts.TLSHandshakeTimeout,
		ExpectContinueTimeout: opts.Timeouts.ExpectContinueTimeout,
		MaxIdleConns:          opts.Timeouts.MaxIdleConns,
		IdleConnTimeout:       opts.Timeouts.IdleConnTimeout,
	}

	if opts.ConfigureMiddleware != nil {
		opts.Middlewares = opts.ConfigureMiddleware(opts.Middlewares)
	}

	return roundTripperFromMiddlewares(opts, opts.Middlewares, transport), nil
}

// GetTLSConfig creates a new tls.Config given provided options.
func GetTLSConfig(opts *TLSOptions) (*tls.Config, error) {
	if opts == nil {
		// #nosec
		return &tls.Config{}, nil
	}

	// #nosec
	config := tls.Config{
		InsecureSkipVerify: opts.InsecureSkipVerify,
		ServerName:         opts.ServerName,
	}

	if len(opts.CACertificate) > 0 {
		caPool := x509.NewCertPool()
		ok := caPool.AppendCertsFromPEM([]byte(opts.CACertificate))
		if !ok {
			return nil, errors.New("failed to parse TLS CA PEM certificate")
		}
		config.RootCAs = caPool
	}

	if len(opts.ClientCertificate) > 0 && len(opts.ClientKey) > 0 {
		cert, err := tls.X509KeyPair([]byte(opts.ClientCertificate), []byte(opts.ClientKey))
		if err != nil {
			return nil, err
		}
		config.Certificates = []tls.Certificate{cert}
	}

	if opts.MinVersion > 0 {
		config.MinVersion = opts.MinVersion
	}

	if opts.MaxVersion > 0 {
		config.MaxVersion = opts.MaxVersion
	}

	return &config, nil
}

func checkOpts(opts *Options) {
	if opts.Middlewares == nil {
		opts.Middlewares = DefaultMiddlewares()
	}

	if opts.Timeouts == nil {
		opts.Timeouts = &DefaultTimeoutOptions
	}
}

// The RoundTripperFunc type is an adapter to allow the use of ordinary
// functions as RoundTrippers. If f is a function with the appropriate
// signature, RountTripperFunc(f) is a RoundTripper that calls f.
type RoundTripperFunc func(req *http.Request) (*http.Response, error)

// RoundTrip implements the RoundTripper interface.
func (rt RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt(r)
}

// Middleware is an interface representing the ability to create a middleware
// that implements the http.RoundTripper interface.
type Middleware interface {
	// CreateMiddleware creates a new middleware.
	CreateMiddleware(opts *Options, next http.RoundTripper) http.RoundTripper
}

// The MiddlewareFunc type is an adapter to allow the use of ordinary
// functions as Middlewares. If f is a function with the appropriate
// signature, MiddlewareFunc(f) is a Middleware that calls f.
type MiddlewareFunc func(opts *Options, next http.RoundTripper) http.RoundTripper

// CreateMiddleware implements the Middleware interface.
func (fn MiddlewareFunc) CreateMiddleware(opts *Options, next http.RoundTripper) http.RoundTripper {
	return fn(opts, next)
}

// MiddlewareName is an interface representing the ability for a middleware to have a name.
type MiddlewareName interface {
	// MiddlewareName returns the middleware name.
	MiddlewareName() string
}

// ConfigureMiddlewareFunc function signature for configuring middleware chain.
type ConfigureMiddlewareFunc func(existingMiddleware []Middleware) []Middleware

// DefaultMiddlewares default middleware applied when creating
// new HTTP clients and no middleware is provided.
// BasicAuthenticationMiddleware and CustomHeadersMiddleware are
// the default middlewares.
func DefaultMiddlewares() []Middleware {
	return []Middleware{
		BasicAuthenticationMiddleware(),
		CustomHeadersMiddleware(),
	}
}

func roundTripperFromMiddlewares(opts *Options, middlewares []Middleware, finalRoundTripper http.RoundTripper) http.RoundTripper {
	for i, j := 0, len(middlewares)-1; i < j; i, j = i+1, j-1 {
		middlewares[i], middlewares[j] = middlewares[j], middlewares[i]
	}

	next := finalRoundTripper

	for _, m := range middlewares {
		next = m.CreateMiddleware(opts, next)
	}

	return next
}

type namedMiddleware struct {
	Name       string
	Middleware Middleware
}

// NamedMiddlewareFunc type is an adapter to allow the use of ordinary
// functions as Middleware. If f is a function with the appropriate
// signature, NamedMiddlewareFunc(f) is a Middleware that calls f.
func NamedMiddlewareFunc(name string, fn MiddlewareFunc) Middleware {
	return &namedMiddleware{
		Name:       name,
		Middleware: fn,
	}
}

func (nm *namedMiddleware) CreateMiddleware(opts *Options, next http.RoundTripper) http.RoundTripper {
	return nm.Middleware.CreateMiddleware(opts, next)
}

func (nm *namedMiddleware) MiddlewareName() string {
	return nm.Name
}
