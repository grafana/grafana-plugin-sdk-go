package oauthtokenretriever

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type tokenRetriever struct {
	GrafanaAppURL string
	ClientID      string
	ClientSecret  string
	HTTPClient    *http.Client

	signer signer
}

// tokenPayload returns a JWT payload for the given user ID, client ID, and host.
func (t *tokenRetriever) tokenPayload(userID string) map[string]interface{} {
	iat := time.Now().Unix()
	exp := iat + 1800
	u := uuid.New()
	payload := map[string]interface{}{
		"iss": t.ClientID,
		"sub": fmt.Sprintf("user:id:%s", userID),
		"aud": t.GrafanaAppURL + "/oauth2/token",
		"exp": exp,
		"iat": iat,
		"jti": u.String(),
	}
	return payload
}

type jwtBearer struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

// retrieveJWTBearerToken returns a JWT bearer token for the given user ID.
func (t *tokenRetriever) retrieveJWTBearerToken(userID string) (string, error) {
	signed, err := t.signer.sign(t.tokenPayload(userID))
	if err != nil {
		return "", fmt.Errorf(fmt.Sprintf("Could not sign the request: %v", err))
	}

	requestParams := url.Values{}
	requestParams.Add("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	requestParams.Add("assertion", signed)
	requestParams.Add("client_id", t.ClientID)
	requestParams.Add("client_secret", t.ClientSecret)
	requestParams.Add("scope", "profile email entitlements")
	buff := bytes.NewBufferString(requestParams.Encode())

	return t.postTokenRequest(buff)
}

// retrieveSelfToken returns a JWT bearer token for the service account created with the app.
func (t *tokenRetriever) retrieveSelfToken() (string, error) {
	requestParams := url.Values{}
	requestParams.Add("grant_type", "client_credentials")
	requestParams.Add("client_id", t.ClientID)
	requestParams.Add("client_secret", t.ClientSecret)
	requestParams.Add("scope", "profile email entitlements")
	buff := bytes.NewBufferString(requestParams.Encode())

	return t.postTokenRequest(buff)
}

// postTokenRequest posts the given request body to the token endpoint and returns the access token.
func (t *tokenRetriever) postTokenRequest(buff *bytes.Buffer) (string, error) {
	req, err := http.NewRequest("POST", t.GrafanaAppURL+"/oauth2/token", buff)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var bearer jwtBearer
	err = json.Unmarshal(body, &bearer)
	if err != nil {
		return "", err
	}

	return bearer.AccessToken, nil
}

type TokenRetriever interface {
	GetExternalServiceToken(userID string) (string, error)
}

func (t *tokenRetriever) GetExternalServiceToken(userID string) (string, error) {
	if userID == "" {
		return t.retrieveSelfToken()
	}
	return t.retrieveJWTBearerToken(userID)
}

func New(httpClient *http.Client, grafanaAppURL string, externalSvcClientID string, externalSvcClientSecret string, externalSvcPrivateKey string) (TokenRetriever, error) {
	signer, err := parsePrivateKey([]byte(externalSvcPrivateKey))
	if err != nil {
		return nil, err
	}
	return &tokenRetriever{
		GrafanaAppURL: grafanaAppURL,
		ClientID:      externalSvcClientID,
		ClientSecret:  externalSvcClientSecret,
		HTTPClient:    httpClient,
		signer:        signer,
	}, nil
}
