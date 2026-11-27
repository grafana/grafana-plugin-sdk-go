package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/go-jose/go-jose/v4"
	"github.com/grafana/authlib/authn"
)

// ErrNoSigningKeys is returned by NewAuthenticatorFromEnv when neither
// EnvVarGrafanaJWKSURL nor EnvVarGrafanaJWKS is set.
var ErrNoSigningKeys = errors.New("no signing keys provided: set " + EnvVarGrafanaJWKSURL + " or " + EnvVarGrafanaJWKS)

// Settings configure how a plugin authenticates the requests it serves.
//
// The plugin SDK authenticates every plugin protocol v2 call that carries an
// access token in its X-Access-Token header (see AccessTokenHeader), and every
// plugin protocol v3 request, which carries one in its x-access-token gRPC
// metadata. The handler's context then carries the caller's AuthInfo (see
// types.AuthInfoFrom), which keeps the token, so the handler can act on behalf
// of the caller.
type Settings struct {
	// Authenticator verifies access tokens, for example
	// authn.NewAccessTokenAuthenticator. If nil, Resolve builds one from
	// EnvVarGrafanaJWKSURL or EnvVarGrafanaJWKS.
	Authenticator authn.Authenticator

	// Audiences are the access token audiences the authenticator Resolve builds
	// accepts, besides the plugin ID, such as the API group an app serves.
	Audiences []string

	// InsecureSkipAuthentication serves requests without verifying them when
	// there is no Authenticator. A request's access token is parsed without
	// checking its signature, audience or namespace, so handlers get the
	// identity the token claims. Requests without a token are served with no
	// identity. Resolve also sets it from EnvVarInsecureSkipAuthentication.
	// Use it only for local development.
	InsecureSkipAuthentication bool
}

// Resolve returns s completed from the environment: InsecureSkipAuthentication
// is set when EnvVarInsecureSkipAuthentication is "true", and without an
// Authenticator, one is built with NewAuthenticatorFromEnv, accepting pluginID
// and s.Audiences as audiences. Without signing keys in the environment the
// Authenticator stays nil. Invalid signing keys are an error.
func (s Settings) Resolve(pluginID string) (Settings, error) {
	if os.Getenv(EnvVarInsecureSkipAuthentication) == "true" {
		s.InsecureSkipAuthentication = true
	}
	if s.Authenticator != nil {
		return s, nil
	}
	var audiences []string
	if pluginID != "" {
		audiences = append(audiences, pluginID)
	}
	for _, audience := range s.Audiences {
		if audience != "" && audience != pluginID {
			audiences = append(audiences, audience)
		}
	}
	authenticator, err := NewAuthenticatorFromEnv(audiences...)
	switch {
	case err == nil:
		s.Authenticator = authenticator
	case !errors.Is(err, ErrNoSigningKeys):
		return s, err
	}
	return s, nil
}

// NewAuthenticatorFromEnv builds an authenticator for access tokens from
// environment variables.
//
// Picks up the signing keys from exactly one of:
//   - GF_JWKS_URL, the JWKS endpoint of the service that signs access tokens
//   - GF_JWKS, the JWKS itself, as a JSON document
//
// Access tokens must be signed by one of those keys and have one of audiences
// as an audience. The URL must be an absolute http or https URL, so in-cluster
// endpoints served over plain http work. An explicit JWKS must contain only
// public signing keys, each with a key ID. Without either variable, the error is
// ErrNoSigningKeys.
func NewAuthenticatorFromEnv(audiences ...string) (authn.Authenticator, error) {
	keysURL := GetEnv(EnvVarGrafanaJWKSURL)
	keysJSON := GetEnv(EnvVarGrafanaJWKS)
	if keysURL == "" && keysJSON == "" {
		return nil, ErrNoSigningKeys
	}
	// Two sources of trusted keys would be ambiguous.
	if keysURL != "" && keysJSON != "" {
		return nil, fmt.Errorf("set only one of %s and %s", EnvVarGrafanaJWKSURL, EnvVarGrafanaJWKS)
	}
	// An empty audience list would make the verifier skip the audience check.
	if len(audiences) == 0 {
		return nil, errors.New("a plugin ID or API group is required as the access token audience")
	}

	var keys authn.KeyRetriever
	if keysURL != "" {
		if err := validateSigningKeysURL(keysURL); err != nil {
			return nil, fmt.Errorf("invalid %s: %w", EnvVarGrafanaJWKSURL, err)
		}
		keys = authn.NewKeyRetriever(authn.KeyRetrieverConfig{SigningKeysURL: keysURL})
	} else {
		static, err := parseSigningKeys(keysJSON)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", EnvVarGrafanaJWKS, err)
		}
		keys = static
	}

	verifier := authn.NewAccessTokenVerifier(authn.VerifierConfig{AllowedAudiences: audiences}, keys)
	return authn.NewAccessTokenAuthenticator(verifier), nil
}

func validateSigningKeysURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Host == "" {
		return errors.New("an absolute URL is required")
	}
	switch u.Scheme {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
}

// staticKeys is a KeyRetriever for a fixed JWKS, indexed by key ID.
type staticKeys map[string]*jose.JSONWebKey

func (k staticKeys) Get(_ context.Context, keyID string) (*jose.JSONWebKey, error) {
	if key, ok := k[keyID]; ok {
		return key, nil
	}
	return nil, authn.ErrInvalidSigningKey
}

func parseSigningKeys(raw string) (staticKeys, error) {
	var set jose.JSONWebKeySet
	if err := json.Unmarshal([]byte(raw), &set); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}
	if len(set.Keys) == 0 {
		return nil, errors.New("JWKS has no keys")
	}
	keys := make(staticKeys, len(set.Keys))
	for i := range set.Keys {
		key := &set.Keys[i]
		switch {
		case key.KeyID == "":
			return nil, fmt.Errorf("key %d has no key ID", i)
		case !key.IsPublic():
			// A private key in the environment would let anyone who reads it sign tokens.
			return nil, fmt.Errorf("key %q is not a public key", key.KeyID)
		case !key.Valid():
			return nil, fmt.Errorf("key %q is invalid", key.KeyID)
		case key.Use != "" && key.Use != "sig":
			return nil, fmt.Errorf("key %q is not a signing key", key.KeyID)
		}
		if _, ok := keys[key.KeyID]; ok {
			return nil, fmt.Errorf("duplicate key ID %q", key.KeyID)
		}
		keys[key.KeyID] = key
	}
	return keys, nil
}
