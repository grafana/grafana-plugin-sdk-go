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

// ErrNoSigningKeys means neither EnvVarGrafanaJWKSURL nor EnvVarGrafanaJWKS is set.
var ErrNoSigningKeys = errors.New("no signing keys provided: set " + EnvVarGrafanaJWKSURL + " or " + EnvVarGrafanaJWKS)

// Settings configure how a plugin checks access tokens.
type Settings struct {
	// Authenticator checks access tokens. If nil, Resolve builds one from
	// EnvVarGrafanaJWKSURL or EnvVarGrafanaJWKS.
	Authenticator authn.Authenticator

	// Audiences are accepted as token audiences, as well as the plugin ID,
	// by the authenticator Resolve builds. For example, an app's API group.
	Audiences []string

	// InsecureSkipAuthentication, when there is no Authenticator, accepts
	// tokens without checking them, so callers can claim any identity.
	// Requests without a token have no identity. Resolve also sets it from
	// EnvVarInsecureSkipAuthentication. Use it only for local development.
	InsecureSkipAuthentication bool
}

// Resolve fills in s from the environment:
//   - InsecureSkipAuthentication is set if EnvVarInsecureSkipAuthentication is "true".
//   - If Authenticator is nil, it is built with NewAuthenticatorFromEnv, for
//     pluginID and s.Audiences. It stays nil if no signing keys are set.
//
// Invalid signing keys return an error.
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

// NewAuthenticatorFromEnv builds an access token authenticator from the
// signing keys in exactly one of:
//   - GF_JWKS_URL: an http or https URL that serves the keys (JWKS).
//   - GF_JWKS: the keys themselves, as a JWKS JSON document. They must be public
//     signing keys, each with a key ID.
//
// Tokens must be signed by one of the keys and have one of audiences as an
// audience. Empty audiences are ignored; at least one non-empty audience is
// required. If neither variable is set, it returns ErrNoSigningKeys.
func NewAuthenticatorFromEnv(audiences ...string) (authn.Authenticator, error) {
	keysURL := GetEnv(EnvVarGrafanaJWKSURL)
	keysJSON := GetEnv(EnvVarGrafanaJWKS)
	if keysURL == "" && keysJSON == "" {
		return nil, ErrNoSigningKeys
	}
	if keysURL != "" && keysJSON != "" {
		return nil, fmt.Errorf("set only one of %s and %s", EnvVarGrafanaJWKSURL, EnvVarGrafanaJWKS)
	}
	// Copy the accepted audiences so callers cannot change the verifier's policy
	// by reusing their slice. Empty entries do not identify a plugin or API group.
	allowedAudiences := make([]string, 0, len(audiences))
	for _, audience := range audiences {
		if audience != "" {
			allowedAudiences = append(allowedAudiences, audience)
		}
	}
	// Without audiences, the verifier would accept any audience.
	if len(allowedAudiences) == 0 {
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

	verifier := authn.NewAccessTokenVerifier(authn.VerifierConfig{AllowedAudiences: allowedAudiences}, keys)
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
			// Anyone who can read a private key could sign tokens with it.
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
