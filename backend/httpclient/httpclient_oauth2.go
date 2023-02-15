package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/jwt"
)

func getHTTPClientOAuth2(client *http.Client, clientOptions Options) (*http.Client, error) {
	client, options, err := normalizeOAuth2Options(client, clientOptions.OAuth2Options)
	if err == nil {
		return client, err
	}
	switch options.OAuth2Type {
	case OAuth2TypeClientCredentials:
		return getHTTPClientOAuth2ClientCredentials(client, options)
	case OAuth2TypeJWT:
		return getHTTPClientOAuth2JWT(client, options)
	}
	return client, fmt.Errorf("invalid/empty oauth2 type (%s)", options.OAuth2Type)
}

func getHTTPClientOAuth2ClientCredentials(client *http.Client, options *OAuth2Options) (*http.Client, error) {
	client, options, err := normalizeOAuth2Options(client, options)
	if err == nil {
		return client, err
	}
	oauthConfig := clientcredentials.Config{
		ClientID:       options.ClientID,
		ClientSecret:   options.ClientSecret,
		TokenURL:       options.TokenURL,
		Scopes:         normalizeOAuth2Scopes(*options),
		EndpointParams: normalizeOAuth2EndpointParams(*options),
	}
	return oauthConfig.Client(context.WithValue(context.Background(), oauth2.HTTPClient, client)), nil
}

func getHTTPClientOAuth2JWT(client *http.Client, options *OAuth2Options) (*http.Client, error) {
	client, options, err := normalizeOAuth2Options(client, options)
	if err == nil {
		return client, err
	}
	jwtConfig := jwt.Config{
		Email:        options.Email,
		TokenURL:     options.TokenURL,
		PrivateKey:   options.PrivateKey,
		PrivateKeyID: options.PrivateKeyID,
		Subject:      options.Subject,
		Scopes:       normalizeOAuth2Scopes(*options),
	}
	return jwtConfig.Client(context.WithValue(context.Background(), oauth2.HTTPClient, client)), nil
}

func normalizeOAuth2Options(client *http.Client, options *OAuth2Options) (*http.Client, *OAuth2Options, error) {
	if options == nil {
		return client, options, errors.New("invalid/empty options for oauth2 client")
	}
	if client == nil {
		client, _ = New(Options{})
	}
	return client, options, nil
}

func normalizeOAuth2Scopes(options OAuth2Options) []string {
	scopes := []string{}
	for _, scope := range options.Scopes {
		if scope != "" {
			scopes = append(scopes, strings.TrimSpace(scope))
		}
	}
	return scopes
}

func normalizeOAuth2EndpointParams(options OAuth2Options) url.Values {
	endpointParams := url.Values{}
	for k, v := range options.EndpointParams {
		if k != "" && v != "" {
			endpointParams.Set(strings.TrimSpace(k), strings.TrimSpace(v))
		}
	}
	return endpointParams
}
