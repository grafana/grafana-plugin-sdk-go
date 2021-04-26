package httpclient

import (
	"time"
)

// Options defines options for creating HTTP clients.
type Options struct {
	// Timeouts timeout/connection related options.
	Timeouts *TimeoutOptions

	// BasicAuth basic authentication related options.
	BasicAuth *BasicAuthOptions

	// TLS TLS related options.
	TLS   *TLSOptions
	SigV4 *SigV4Config

	// Headers custom headers.
	Headers map[string]string

	// CustomOptions allows custom options to be provided.
	CustomOptions map[string]interface{}

	// Labels could be used by certain middlewares.
	Labels map[string]string

	// Middlewares optionally provide additional middlewares.
	Middlewares []Middleware

	// ConfigureMiddleware optionally provide a ConfigureMiddlewareFunc
	// to modify the middlewares chain.
	ConfigureMiddleware ConfigureMiddlewareFunc
}

// BasicAuthOptions basic authentication options.
type BasicAuthOptions struct {
	User     string
	Password string
}

// TimeoutOptions timeout/connection options.
type TimeoutOptions struct {
	Timeout               time.Duration
	KeepAlive             time.Duration
	TLSHandshakeTimeout   time.Duration
	ExpectContinueTimeout time.Duration
	MaxIdleConns          int
	IdleConnTimeout       time.Duration
}

// DefaultTimeoutOptions default timeout/connection options.
var DefaultTimeoutOptions = TimeoutOptions{
	Timeout:               30 * time.Second,
	KeepAlive:             30 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
}

// TLSOptions TLS options.
type TLSOptions struct {
	CACertificate      string
	ClientCertificate  string
	ClientKey          string
	InsecureSkipVerify bool
	ServerName         string

	// MinVersion configures the tls.Config.MinVersion.
	MinVersion uint16

	// MaxVersion configures the tls.Config.MaxVersion.
	MaxVersion uint16
}

// SigV4Config AWS SigV4 options.
type SigV4Config struct {
	AuthType      string
	Profile       string
	Service       string
	AccessKey     string
	SecretKey     string
	AssumeRoleARN string
	ExternalID    string
	Region        string
}
