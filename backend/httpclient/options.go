package httpclient

import (
	"crypto/tls"
	"net/http"
	"time"
)

// ConfigureClientFunc function signature for configuring http.Client.
// Called after http.Client creation.
type ConfigureClientFunc func(opts Options, client *http.Client)

// ConfigureTransportFunc function signature for configuring http.Transport.
// Called after http.Transport creation.
type ConfigureTransportFunc func(opts Options, transport *http.Transport)

// ConfigureTLSConfigFunc function signature for configuring tls.Config.
// Called after tls.Config creation.
type ConfigureTLSConfigFunc func(opts Options, tlsConfig *tls.Config)

// AuthenticationMethod defines the type of authentication method that needs to be use.
type AuthenticationMethod string

const (
	// AuthenticationMethodOAuth2 is oauth2 type authentication.
	// Currently support client credentials and JWT type OAuth2 workflows.
	AuthenticationMethodOAuth2 AuthenticationMethod = "oauth2"
)

// Options defines options for creating HTTP clients.
type Options struct {

	// AuthenticationMethod defines the type of authentication method used by the http client.
	AuthenticationMethod AuthenticationMethod

	// OAuth2Options defines options for OAuth2 client. This will be used only if AuthenticationMethod is AuthenticationMethodOAuth.
	OAuth2Options *OAuth2Options

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

	// ConfigureClient optionally provide a ConfigureClientFunc
	// to modify the created http.Client.
	ConfigureClient ConfigureClientFunc

	// ConfigureTransport optionally provide a ConfigureTransportFunc
	// to modify the created http.Client.
	ConfigureTransport ConfigureTransportFunc

	// ConfigureTLSConfig optionally provide a ConfigureTLSConfigFunc
	// to modify the created http.Client.
	ConfigureTLSConfig ConfigureTLSConfigFunc
}

// BasicAuthOptions basic authentication options.
type BasicAuthOptions struct {
	User     string
	Password string
}

// OAuth2Type defines type of oauth2 grant type
type OAuth2Type string

const (
	OAuth2TypeClientCredentials OAuth2Type = "client_credentials"
	OAuth2TypeJWT               OAuth2Type = "jwt"
)

// OAuth2Options defines options for OAuth2 Client
type OAuth2Options struct {
	OAuth2Type     OAuth2Type
	TokenURL       string
	Scopes         []string
	ClientID       string
	ClientSecret   string
	EndpointParams map[string]string
	Subject        string
	Email          string
	PrivateKey     []byte
	PrivateKeyID   string
}

// TimeoutOptions timeout/connection options.
type TimeoutOptions struct {
	Timeout               time.Duration
	DialTimeout           time.Duration
	KeepAlive             time.Duration
	TLSHandshakeTimeout   time.Duration
	ExpectContinueTimeout time.Duration
	MaxConnsPerHost       int
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	IdleConnTimeout       time.Duration
}

// DefaultTimeoutOptions default timeout/connection options.
var DefaultTimeoutOptions = TimeoutOptions{
	Timeout:               30 * time.Second,
	DialTimeout:           10 * time.Second,
	KeepAlive:             30 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	MaxConnsPerHost:       0,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   100,
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
