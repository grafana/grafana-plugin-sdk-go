package auth

import "os"

// Environment variables read by a plugin process. Grafana sets them when it starts the plugin.
const (
	// EnvVarAPIAccessRouterURL names the environment variable holding the URL of the API server
	// the plugin's resource clients talk to.
	EnvVarAPIAccessRouterURL = "GF_API_ACCESS_ROUTER_URL"
	// EnvVarAPIAccessCAFile names the environment variable holding the path of a CA file
	// used to verify the API server's certificate.
	EnvVarAPIAccessCAFile = "GF_API_ACCESS_CA_FILE"
	// EnvVarAPIAccessInsecureTLS names the environment variable that, when "true",
	// skips verification of the API server's certificate.
	EnvVarAPIAccessInsecureTLS = "GF_API_ACCESS_INSECURE_TLS"
	// EnvVarAPIAccessCAPToken names the environment variable holding the Cloud Access Policy (CAP)
	// token exchanged for access tokens.
	EnvVarAPIAccessCAPToken = "GF_API_ACCESS_CAP_TOKEN" //nolint:gosec // variable name, not a credential
	// EnvVarAPIAccessTokenExchangeURL names the environment variable holding the URL of the
	// token exchange service.
	EnvVarAPIAccessTokenExchangeURL = "GF_API_ACCESS_TOKEN_EXCHANGE_URL" //nolint:gosec // variable name, not a credential
	// EnvVarAPIAccessBearerToken names the environment variable holding a bearer token used
	// when token exchange is not configured.
	//
	// Deprecated: configure token exchange with EnvVarAPIAccessTokenExchangeURL and
	// EnvVarAPIAccessCAPToken instead.
	EnvVarAPIAccessBearerToken = "GF_API_ACCESS_BEARER_TOKEN" //nolint:gosec // variable name, not a credential

	// EnvVarGrafanaJWKSURL names the environment variable holding the
	// URL of the JWKS used to verify access tokens. Keys are downloaded on demand.
	EnvVarGrafanaJWKSURL = "GF_JWKS_URL"
	// EnvVarGrafanaJWKS names the environment variable holding the
	// JWKS used to verify access tokens, as a JSON document.
	EnvVarGrafanaJWKS = "GF_JWKS"

	// EnvVarInsecureSkipAuthentication names the environment variable that, when
	// "true", serves requests without verifying them if no signing keys are
	// configured (see Settings.InsecureSkipAuthentication). Grafana sets it from
	// insecure_skip_authentication = true in the plugin's [plugin.<id>]
	// settings. Use it only for local development.
	EnvVarInsecureSkipAuthentication = "GF_PLUGIN_INSECURE_SKIP_AUTHENTICATION"
)

// legacyEnvVars maps environment variable names to the names they replace.
// The legacy names are still read when the new name is unset.
var legacyEnvVars = map[string]string{ //nolint:gosec // variable names, not credentials
	EnvVarAPIAccessRouterURL:        "API_ACCESS_ROUTER_URL",
	EnvVarAPIAccessCAFile:           "API_ACCESS_CA_FILE",
	EnvVarAPIAccessInsecureTLS:      "API_ACCESS_INSECURE_TLS",
	EnvVarAPIAccessCAPToken:         "API_ACCESS_CAP_TOKEN",
	EnvVarAPIAccessTokenExchangeURL: "API_ACCESS_TOKEN_EXCHANGE_URL",
	EnvVarAPIAccessBearerToken:      "API_ACCESS_BEARER_TOKEN",
	EnvVarGrafanaJWKSURL:            "GRAFANA_JWKS_URL",
	EnvVarGrafanaJWKS:               "GRAFANA_JWKS",
}

// GetEnv returns the value of the environment variable name, one of the
// EnvVar constants, falling back to the legacy name it replaces when it is
// empty.
func GetEnv(name string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	if legacy, ok := legacyEnvVars[name]; ok {
		return os.Getenv(legacy)
	}
	return ""
}
