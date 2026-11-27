package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/grafana/authlib/authn"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestNewAuthenticatorFromEnvVerifiesSignedAccessTokens(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	jwks := testJWKS(t, jose.JSONWebKey{Key: &key.PublicKey, KeyID: "test", Algorithm: string(jose.ES256), Use: "sig"})

	for _, tt := range []struct {
		name string
		env  func(t *testing.T)
	}{
		{name: "downloaded from URL", env: func(t *testing.T) {
			t.Helper()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(jwks))
			}))
			t.Cleanup(server.Close)
			t.Setenv(EnvVarGrafanaJWKSURL, server.URL)
		}},
		{name: "explicit JWKS", env: func(t *testing.T) {
			t.Helper()
			t.Setenv(EnvVarGrafanaJWKS, jwks)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.env(t)
			settings, err := Settings{Audiences: []string{"example.grafana.app"}}.Resolve("example-app")
			require.NoError(t, err)
			authenticator := settings.Authenticator
			require.NotNil(t, authenticator)

			authenticate := func(token string) error {
				_, err := authenticator.Authenticate(context.Background(), authn.NewGRPCTokenProvider(metadata.Pairs(AccessTokenMetadataKey, token)))
				return err
			}

			// Either the API group or the plugin ID is an accepted audience.
			for _, audience := range []string{"example.grafana.app", "example-app"} {
				info, err := authenticator.Authenticate(context.Background(), authn.NewGRPCTokenProvider(metadata.Pairs(AccessTokenMetadataKey, signTestToken(t, key, "test", audience))))
				require.NoError(t, err, audience)
				require.Equal(t, "stacks-1", info.GetNamespace())
			}

			err = authenticate(signTestToken(t, key, "test", "other.grafana.app"))
			require.True(t, authn.IsUnauthenticatedErr(err), "wrong audience: %v", err)

			err = authenticate(signTestToken(t, key, "unknown", "example.grafana.app"))
			require.True(t, authn.IsUnauthenticatedErr(err), "unknown key ID: %v", err)

			otherKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			require.NoError(t, err)
			require.Error(t, authenticate(signTestToken(t, otherKey, "test", "example.grafana.app")), "a token signed by another key must be rejected")
		})
	}
}

func TestNewAuthenticatorFromEnvConfiguration(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	public := jose.JSONWebKey{Key: &key.PublicKey, KeyID: "a", Algorithm: string(jose.ES256), Use: "sig"}
	validJWKS := testJWKS(t, public)
	withKID := func(k jose.JSONWebKey, kid string) jose.JSONWebKey { k.KeyID = kid; return k }
	encryption := public
	encryption.Use = "enc"

	audiences := []string{"example.grafana.app"}
	for _, tt := range []struct {
		name      string
		url       string
		jwks      string
		audiences []string
		wantErr   string
	}{
		{name: "https", url: "https://auth.example.com/v1/jwks", audiences: audiences},
		{name: "http on localhost", url: "http://localhost:3000/jwks", audiences: audiences},
		{name: "http on loopback IP", url: "http://127.0.0.1:3000/jwks", audiences: audiences},
		{name: "http on IPv6 loopback", url: "http://[::1]:3000/jwks", audiences: audiences},
		{name: "explicit JWKS", jwks: validJWKS, audiences: audiences},
		{name: "explicit JWKS with several keys", jwks: testJWKS(t, public, withKID(public, "b")), audiences: audiences},
		{name: "missing", audiences: audiences, wantErr: "set GF_JWKS_URL or GF_JWKS"},
		{name: "both sources", url: "https://auth.example.com/v1/jwks", jwks: validJWKS, audiences: audiences, wantErr: "set only one of"},
		{name: "http on remote host", url: "http://api-lb.auth.svc.cluster.local./v1/keys", audiences: audiences},
		{name: "relative", url: "/jwks", audiences: audiences, wantErr: "absolute URL"},
		{name: "file URL", url: "file:///etc/jwks", audiences: audiences, wantErr: "absolute URL"},
		{name: "ftp scheme", url: "ftp://auth.example.com/jwks", audiences: audiences, wantErr: "unsupported scheme"},
		{name: "malformed JWKS", jwks: "not json", audiences: audiences, wantErr: "decode JWKS"},
		{name: "empty JWKS", jwks: `{"keys":[]}`, audiences: audiences, wantErr: "JWKS has no keys"},
		{name: "key without ID", jwks: testJWKS(t, withKID(public, "")), audiences: audiences, wantErr: "has no key ID"},
		{name: "private key", jwks: testJWKS(t, jose.JSONWebKey{Key: key, KeyID: "a", Algorithm: string(jose.ES256)}), audiences: audiences, wantErr: "not a public key"},
		{name: "encryption key", jwks: testJWKS(t, encryption), audiences: audiences, wantErr: "not a signing key"},
		{name: "duplicate key ID", jwks: testJWKS(t, public, public), audiences: audiences, wantErr: "duplicate key ID"},
		{name: "no audience", url: "https://auth.example.com/v1/jwks", wantErr: "a plugin ID or API group is required"},
		{name: "empty audience", jwks: validJWKS, audiences: []string{""}, wantErr: "a plugin ID or API group is required"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(EnvVarGrafanaJWKSURL, tt.url)
			t.Setenv(EnvVarGrafanaJWKS, tt.jwks)
			authenticator, err := NewAuthenticatorFromEnv(tt.audiences...)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Nil(t, authenticator)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, authenticator)
		})
	}

	t.Setenv(EnvVarGrafanaJWKSURL, "")
	t.Setenv(EnvVarGrafanaJWKS, "")
	_, err = NewAuthenticatorFromEnv(audiences...)
	require.ErrorIs(t, err, ErrNoSigningKeys)

	// The legacy names are still read.
	t.Setenv("GRAFANA_JWKS", validJWKS)
	authenticator, err := NewAuthenticatorFromEnv(audiences...)
	require.NoError(t, err)
	require.NotNil(t, authenticator)
}

func TestNewAuthenticatorFromEnvOwnsAudiences(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	t.Setenv(EnvVarGrafanaJWKSURL, "")
	t.Setenv(EnvVarGrafanaJWKS, testJWKS(t, jose.JSONWebKey{Key: &key.PublicKey, KeyID: "test"}))
	audiences := []string{"example-app", ""}
	authenticator, err := NewAuthenticatorFromEnv(audiences...)
	require.NoError(t, err)
	require.Equal(t, []string{"example-app", ""}, audiences, "the caller's slice must not be modified")
	audiences[0] = "other-app"

	for _, audience := range []string{"example-app", "other-app", ""} {
		_, err := AuthenticateAccessToken(context.Background(), authenticator, signTestToken(t, key, "test", audience))
		if audience == "example-app" {
			require.NoError(t, err)
		} else {
			require.Equal(t, codes.Unauthenticated, status.Code(err), audience)
		}
	}
}

func TestSettingsResolve(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	validJWKS := testJWKS(t, jose.JSONWebKey{Key: &key.PublicKey, KeyID: "a", Algorithm: string(jose.ES256), Use: "sig"})

	t.Run("no signing keys leaves the authenticator unset", func(t *testing.T) {
		t.Setenv(EnvVarGrafanaJWKSURL, "")
		t.Setenv(EnvVarGrafanaJWKS, "")
		t.Setenv(EnvVarInsecureSkipAuthentication, "")
		s, err := Settings{}.Resolve("example-app")
		require.NoError(t, err)
		require.Nil(t, s.Authenticator)
		require.False(t, s.InsecureSkipAuthentication)
	})

	t.Run("signing keys build the authenticator", func(t *testing.T) {
		t.Setenv(EnvVarGrafanaJWKS, validJWKS)
		s, err := Settings{}.Resolve("example-app")
		require.NoError(t, err)
		require.NotNil(t, s.Authenticator)
	})

	t.Run("invalid signing keys are an error", func(t *testing.T) {
		t.Setenv(EnvVarGrafanaJWKS, "not json")
		_, err := Settings{}.Resolve("example-app")
		require.ErrorContains(t, err, "decode JWKS")
	})

	t.Run("signing keys without an audience are an error", func(t *testing.T) {
		t.Setenv(EnvVarGrafanaJWKS, validJWKS)
		_, err := Settings{}.Resolve("")
		require.ErrorContains(t, err, "a plugin ID or API group is required")
	})

	t.Run("an explicit authenticator is kept", func(t *testing.T) {
		t.Setenv(EnvVarGrafanaJWKS, "not json")
		explicit := authn.NewAccessTokenAuthenticator(authn.NewUnsafeAccessTokenVerifier(authn.VerifierConfig{}))
		s, err := Settings{Authenticator: explicit}.Resolve("example-app")
		require.NoError(t, err)
		require.Same(t, explicit, s.Authenticator)
	})

	t.Run("the environment can skip authentication", func(t *testing.T) {
		t.Setenv(EnvVarInsecureSkipAuthentication, "true")
		s, err := Settings{}.Resolve("example-app")
		require.NoError(t, err)
		require.True(t, s.InsecureSkipAuthentication)
	})
}

func TestGetEnv(t *testing.T) {
	t.Setenv(EnvVarAPIAccessRouterURL, "")
	t.Setenv("API_ACCESS_ROUTER_URL", "https://legacy")
	require.Equal(t, "https://legacy", GetEnv(EnvVarAPIAccessRouterURL))

	t.Setenv(EnvVarAPIAccessRouterURL, "https://current")
	require.Equal(t, "https://current", GetEnv(EnvVarAPIAccessRouterURL))

	t.Setenv("GF_SOME_OTHER_VAR", "value")
	require.Equal(t, "value", GetEnv("GF_SOME_OTHER_VAR"))
}

func testJWKS(t *testing.T, keys ...jose.JSONWebKey) string {
	t.Helper()
	raw, err := json.Marshal(jose.JSONWebKeySet{Keys: keys})
	require.NoError(t, err)
	return string(raw)
}

func signTestToken(t *testing.T, key *ecdsa.PrivateKey, keyID, audience string) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: key}, (&jose.SignerOptions{}).WithType(jose.ContentType(authn.TokenTypeAccess)).WithHeader("kid", keyID))
	require.NoError(t, err)
	token, err := jwt.Signed(signer).Claims(jwt.Claims{Subject: "access-policy:grafana", Audience: jwt.Audience{audience}, Expiry: jwt.NewNumericDate(time.Now().Add(time.Hour))}).
		Claims(authn.AccessTokenClaims{Namespace: "stacks-1"}).Serialize()
	require.NoError(t, err)
	return token
}
