package httpclient

import (
	"time"
)

type Options struct {
	// Timeouts
	Timeouts *TimeoutOptions

	BasicAuth *BasicAuthOptions
	TLS       *TLSOptions
	SigV4     *SigV4Config

	// Custom headers
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

type BasicAuthOptions struct {
	User     string
	Password string
}

type TimeoutOptions struct {
	Timeout               time.Duration
	KeepAlive             time.Duration
	TLSHandshakeTimeout   time.Duration
	ExpectContinueTimeout time.Duration
	MaxIdleConns          int
	IdleConnTimeout       time.Duration
}

var DefaultTimeoutOptions = TimeoutOptions{
	Timeout:               30 * time.Second,
	KeepAlive:             30 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
}

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
