package oauthtokenretriever

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type TokenRetriever interface {
	OnBehalfOfUser(ctx context.Context, userID string) (string, error)
	Self(ctx context.Context) (string, error)
}

type tokenRetriever struct {
	audience string
	signer   signer
	conf     *clientcredentials.Config
}

// tokenPayload returns a JWT payload for the given user ID, client ID, and host.
func (t *tokenRetriever) tokenPayload(userID string) map[string]interface{} {
	iat := time.Now().Unix()
	exp := iat + 1800
	u := uuid.New()
	payload := map[string]interface{}{
		"iss": t.conf.ClientID,
		"sub": fmt.Sprintf("user:id:%s", userID),
		"aud": []string{t.conf.TokenURL, t.audience},
		"exp": exp,
		"iat": iat,
		"jti": u.String(),
	}
	return payload
}

func (t *tokenRetriever) Self(ctx context.Context) (string, error) {
	t.conf.EndpointParams = url.Values{
		"audience": {t.audience},
	}
	tok, err := t.conf.TokenSource(ctx).Token()
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

func (t *tokenRetriever) OnBehalfOfUser(ctx context.Context, userID string) (string, error) {
	signed, err := t.signer.sign(t.tokenPayload(userID))
	if err != nil {
		return "", err
	}

	t.conf.EndpointParams = url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {signed},
	}
	tok, err := t.conf.TokenSource(ctx).Token()
	if err != nil {
		return "", err
	}

	return tok.AccessToken, nil
}

func New(settings backend.AppInstanceSettings) (TokenRetriever, error) {
	// The Grafana URL is required to obtain tokens later on
	grafanaAppURL := strings.TrimRight(os.Getenv("GF_APP_URL"), "/")
	if grafanaAppURL == "" {
		// For debugging purposes only
		grafanaAppURL = "http://localhost:3000"
	}

	privateKey := os.Getenv("GF_PLUGIN_APP_PRIVATE_KEY")
	if privateKey == "" {
		return nil, fmt.Errorf("GF_PLUGIN_APP_PRIVATE_KEY is required")
	}

	signer, err := parsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, err
	}

	useMultiTenantAuthSvc := os.Getenv("GF_USE_MULTI_TENANT_AUTH_SERVICE") == "true"
	if useMultiTenantAuthSvc {
		clientID := settings.DecryptedSecureJSONData["clientId"]
		clientSecret := settings.DecryptedSecureJSONData["clientSecret"]
		audience := os.Getenv("GF_PLUGIN_AUTH_AUDIENCE")
		if audience == "" {
			return nil, fmt.Errorf("GF_PLUGIN_AUTH_AUDIENCE is required")
		}

		authServiceURL := os.Getenv("GF_PLUGIN_AUTH_SERVICE_URL")
		if authServiceURL == "" {
			return nil, fmt.Errorf("GF_PLUGIN_AUTH_SERVICE_URL is required")
		}

		tokenEndpointURL, err := url.JoinPath(authServiceURL, "/oauth2/token")
		if err != nil {
			return nil, fmt.Errorf("failed to generate token endpoint URL: %w", err)
		}

		return &tokenRetriever{
			audience: audience,
			signer:   signer,
			conf: &clientcredentials.Config{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				TokenURL:     tokenEndpointURL,
				AuthStyle:    oauth2.AuthStyleInParams,
				Scopes:       []string{"profile", "email", "entitlements"},
			},
		}, nil
	} else {

		clientID := os.Getenv("GF_PLUGIN_APP_CLIENT_ID")
		if clientID == "" {
			return nil, fmt.Errorf("GF_PLUGIN_APP_CLIENT_ID is required")
		}

		clientSecret := os.Getenv("GF_PLUGIN_APP_CLIENT_SECRET")
		if clientSecret == "" {
			return nil, fmt.Errorf("GF_PLUGIN_APP_CLIENT_SECRET is required")
		}

		return &tokenRetriever{
			signer: signer,
			conf: &clientcredentials.Config{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				TokenURL:     grafanaAppURL + "/oauth2/token",
				AuthStyle:    oauth2.AuthStyleInParams,
				Scopes:       []string{"profile", "email", "entitlements"},
			},
		}, nil
	}
}
