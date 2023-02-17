package httpclient_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClientOAuth2Invalid(t *testing.T) {
	server := testGetOAuthServer(t)
	defer server.Close()
	hc, err := httpclient.New(httpclient.Options{
		AuthenticationMethod: httpclient.AuthenticationMethodOAuth2,
		Headers:              map[string]string{"h1": "v1"},
		OAuth2Options: &httpclient.OAuth2Options{
			OAuth2Type: "invalid",
			TokenURL:   server.URL + "/token",
		},
	})
	require.NotNil(t, hc)
	require.NotNil(t, err)
	assert.Equal(t, errors.New("invalid/empty oauth2 type (invalid)"), err)
}

func TestHTTPClientOAuth2ClientCredentials(t *testing.T) {
	server := testGetOAuthServer(t)
	defer server.Close()
	t.Run("valid client credentials should respond correctly", func(t *testing.T) {
		hc, err := httpclient.New(httpclient.Options{
			AuthenticationMethod: httpclient.AuthenticationMethodOAuth2,
			Headers:              map[string]string{"h1": "v1"},
			OAuth2Options: &httpclient.OAuth2Options{
				OAuth2Type: httpclient.OAuth2TypeClientCredentials,
				TokenURL:   server.URL + "/token",
			},
		})
		require.Nil(t, err)
		require.NotNil(t, hc)
		res, err := hc.Get(server.URL + "/foo")
		require.Nil(t, err)
		require.NotNil(t, res)
		if res != nil && res.Body != nil {
			defer res.Body.Close()
		}
		bodyBytes, err := io.ReadAll(res.Body)
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, `"hello world"`, string(bodyBytes))
	})
}

func TestHTTPClientOAuth2JWT(t *testing.T) {
	server := testGetOAuthServer(t)
	defer server.Close()
	t.Run("invalid private key should throw error", func(t *testing.T) {
		privateKey := testGenerateKey(t)
		hc, err := httpclient.New(httpclient.Options{
			AuthenticationMethod: httpclient.AuthenticationMethodOAuth2,
			Headers:              map[string]string{"h1": "v1"},
			OAuth2Options: &httpclient.OAuth2Options{
				OAuth2Type: httpclient.OAuth2TypeJWT,
				TokenURL:   server.URL + "/token",
				PrivateKey: privateKey,
			},
		})
		require.Nil(t, err)
		require.NotNil(t, hc)
		res, err := hc.Get(server.URL + "/foo")
		if res != nil && res.Body != nil {
			defer res.Body.Close()
		}
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "private key should be a PEM or plain PKCS1 or PKCS8; parse error: asn1: structure error"))
		require.Nil(t, res)
	})
	t.Run("valid private key should not throw error", func(t *testing.T) {
		privateKey := testGenerateKey(t)
		hc, err := httpclient.New(httpclient.Options{
			AuthenticationMethod: httpclient.AuthenticationMethodOAuth2,
			Headers:              map[string]string{"h1": "v1"},
			OAuth2Options: &httpclient.OAuth2Options{
				OAuth2Type: httpclient.OAuth2TypeJWT,
				TokenURL:   server.URL + "/token",
				PrivateKey: privateKey,
			},
		})
		require.Nil(t, err)
		require.NotNil(t, hc)
		res, err := hc.Get(server.URL + "/foo")
		require.Nil(t, err)
		require.NotNil(t, res)
		if res != nil && res.Body != nil {
			defer res.Body.Close()
		}
		bodyBytes, err := io.ReadAll(res.Body)
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, `"hello world"`, string(bodyBytes))
	})
}

func testGetOAuthServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenValue := "foo"
		if r.URL.String() != "/token" {
			if r.Header.Get("Authorization") != "Bearer foo" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `"hello world"`)
			return
		}
		if r.Header.Get("h1") != "v1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, fmt.Sprintf(`{"access_token": "%s", "refresh_token": "bar"}`, tokenValue))
	}))
}

func testGenerateKey(t *testing.T) (privateKey []byte) {
	t.Helper()
	if strings.Contains(t.Name(), "invalid_private_key") {
		return []byte("invalid private key")
	}
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		panic(err)
	}
	privateKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return privateKey
}
