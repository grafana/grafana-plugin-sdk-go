package auth

import "os"

// Environment variables that Grafana sets when it starts a plugin.
const (
	// EnvVarAPIAccessRouterURL is the URL of the API server the plugin's clients call.
	EnvVarAPIAccessRouterURL = "GF_API_ACCESS_ROUTER_URL"
	// EnvVarAPIAccessCAFile is the path of a CA file for the API server's certificate.
	EnvVarAPIAccessCAFile = "GF_API_ACCESS_CA_FILE"
	// EnvVarAPIAccessInsecureTLS, if "true", skips checking the API server's certificate.
	EnvVarAPIAccessInsecureTLS = "GF_API_ACCESS_INSECURE_TLS"
	// EnvVarAPIAccessCAPToken is the Cloud Access Policy (CAP) token, exchanged for access tokens.
	EnvVarAPIAccessCAPToken = "GF_API_ACCESS_CAP_TOKEN" //nolint:gosec // variable name, not a credential
	// EnvVarAPIAccessTokenExchangeURL is the URL of the token exchange service.
	EnvVarAPIAccessTokenExchangeURL = "GF_API_ACCESS_TOKEN_EXCHANGE_URL" //nolint:gosec // variable name, not a credential
	// EnvVarAPIAccessBearerToken is a bearer token, used when token exchange is not configured.
	//
	// Deprecated: configure token exchange with EnvVarAPIAccessTokenExchangeURL and
	// EnvVarAPIAccessCAPToken instead.
	EnvVarAPIAccessBearerToken = "GF_API_ACCESS_BEARER_TOKEN" //nolint:gosec // variable name, not a credential

	// EnvVarGrafanaJWKSURL is the URL of the keys (JWKS) that check access tokens.
	EnvVarGrafanaJWKSURL = "GF_JWKS_URL"
	// EnvVarGrafanaJWKS is the keys (JWKS) that check access tokens, as JSON.
	EnvVarGrafanaJWKS = "GF_JWKS"

	// EnvVarInsecureSkipAuthentication, if "true", accepts tokens without
	// checking them when no signing keys are set (see
	// Settings.InsecureSkipAuthentication). Grafana sets it from
	// insecure_skip_authentication in the plugin's [plugin.<id>] settings.
	// Use it only for local development.
	EnvVarInsecureSkipAuthentication = "GF_PLUGIN_INSECURE_SKIP_AUTHENTICATION"
)

// legacyEnvVars maps each variable to the older name it replaces, which is
// still read when the new one is empty.
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

// GetEnv returns the value of the variable name, one of the EnvVar constants,
// or of its older name if it is empty.
func GetEnv(name string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	if legacy, ok := legacyEnvVars[name]; ok {
		return os.Getenv(legacy)
	}
	return ""
}
