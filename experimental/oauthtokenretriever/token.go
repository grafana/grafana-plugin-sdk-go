package oauthtokenretriever

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type TokenRetriever interface {
	OnBehalfOfUser(userID string) (string, error)
	Self() (string, error)
}

type tokenRetriever struct {
	signer signer
	conf   *oauth2.Config
}

// tokenPayload returns a JWT payload for the given user ID, client ID, and host.
func (t *tokenRetriever) tokenPayload(userID string) map[string]interface{} {
	iat := time.Now().Unix()
	exp := iat + 1800
	u := uuid.New()
	payload := map[string]interface{}{
		"iss": t.conf.ClientID,
		"sub": fmt.Sprintf("user:id:%s", userID),
		"aud": t.conf.Endpoint.TokenURL,
		"exp": exp,
		"iat": iat,
		"jti": u.String(),
	}
	return payload
}

func (t *tokenRetriever) Self() (string, error) {
	tok, err := t.conf.Exchange(context.Background(), "",
		oauth2.SetAuthURLParam("grant_type", "client_credentials"),
		oauth2.SetAuthURLParam("scope", "profile email entitlements"),
	)
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

func (t *tokenRetriever) OnBehalfOfUser(userID string) (string, error) {
	signed, err := t.signer.sign(t.tokenPayload(userID))
	if err != nil {
		return "", err
	}

	tok, err := t.conf.Exchange(context.Background(), "",
		oauth2.SetAuthURLParam("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer"),
		oauth2.SetAuthURLParam("assertion", signed),
		oauth2.SetAuthURLParam("scope", "profile email entitlements"),
	)
	if err != nil {
		return "", err
	}

	return tok.AccessToken, nil
}

func New() (TokenRetriever, error) {
	// The Grafana URL is required to obtain tokens later on
	grafanaAppURL := strings.TrimRight(os.Getenv("GF_APP_URL"), "/")
	if grafanaAppURL == "" {
		// For debugging purposes only
		grafanaAppURL = "http://localhost:3000"
	}

	clientID := os.Getenv("GF_PLUGIN_APP_CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("GF_PLUGIN_APP_CLIENT_ID is required")
	}

	clientSecret := os.Getenv("GF_PLUGIN_APP_CLIENT_SECRET")
	if clientSecret == "" {
		return nil, fmt.Errorf("GF_PLUGIN_APP_CLIENT_SECRET is required")
	}

	privateKey := os.Getenv("GF_PLUGIN_APP_PRIVATE_KEY")
	if privateKey == "" {
		return nil, fmt.Errorf("GF_PLUGIN_APP_PRIVATE_KEY is required")
	}

	signer, err := parsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, err
	}

	return &tokenRetriever{
		signer: signer,
		conf: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				TokenURL:  grafanaAppURL + "/oauth2/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
	}, nil
}
