package licensing

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"

	jose "gopkg.in/go-jose/go-jose.v2"
)

var logger = backend.Logger

var (
	errFileNotFound            = fmt.Errorf("license token file not found")
	errLoadFailure             = fmt.Errorf("error loading license token")
	errParsing                 = fmt.Errorf("error parsing license token")
	errVerificationKeyNotFound = fmt.Errorf("license verification key not found")
	errLicenseVerificationKey  = fmt.Errorf("error loading license verification key")
	errVerifyToken             = fmt.Errorf("error verifying license token")
)

// embeddedKeys are the JWKs we use for license validation.
// They are JSON structs keyed by their "kid" field.
//
// If more keys are necessary, see the /scripts/marketplace/genjwk script in grafana-catalog-team repo. Remember to save the values somewhere safe, like the current standard password manager.
var embeddedKeys = map[string]string{
	"TEST-MKT-1": `{"kty":"RSA","kid":"TEST-MKT-1","alg":"RS512","n":"0rCuSrvARaGcLTztSLWf2IlW8PPFfN4rBJKltXjWyY8cA8Ek0xzv9oNGwcaXjHHujySsH2uOX5oyw6xReTOjGwHNlirZlbLP582YTn0CDNB5J-Yp60VK8GmwIjw3rfuc5zQYxQM8YZ1RNzRM9XNWAd6oryZB5j0I69DMCKONGZZu4SsFguES9XM2bHYAFMpYUpBmW4HLDJmVM1v2QKFAUCO4V8sdzA7SCSrselpP6zgjvcn9w8V8yqqq9wBTuPcH2FZVhP9OELfx5gZigsuPVJw4TknbP3J5MB2m_-IGtvNVVqiiw-xF6kuA761pDbWiXYeqD3Je9NYhBlykuKINfCGa212GiuRVG-DHmFn_hA1zU_vaTb7TLGI-HCILnJHH0y4Eq1fzUCZTuibDLFAnCplZ1ECdioYY2-mxRKju8736zFq7JP16ZktIt2wMMAnkIgQ1dggNxia9lYpxIjkRHk7-0FbZ5V9MY_zGopDBxJPZYxmvZU7lC3TjvGsuuS3w8DWKLM--NDkgdDBD_vLR5haz7kWDBkM9xa2bQxcVImOF_XtEl2tg_1KsCCVRPdJsFjmlhRo8OnhB4DarzgjGWnfi0KDZ7MLgelBK2Gdpl9MoJMe3erZ9LLeJp1z1vmUfIM9jovBUhDXuB3yrig6V6BUTHgLBTjD1JXhB2zJ0bdU","e":"AQAB"}`,
}

func LoadTokenFromValue(tokenStr, appUrl, validationKeys, pluginId string) *LicenseToken {
	var token LicenseToken
	token.Parse(tokenStr, appUrl, validationKeys, pluginId)
	return &token
}

func LoadTokenFromFile(tokenPath, appUrl, validationKeys, pluginId string) *LicenseToken {
	var token LicenseToken

	// Can ignore gosec G304 since tokenPath is derived from a configuration parameter
	// nolint:gosec
	dat, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			token.Status = NotFound
			token.Error = fmt.Errorf("%w: %s", errFileNotFound, tokenPath)
			return &token
		}

		token.Status = Invalid
		token.Error = fmt.Errorf("%w: %w", errLoadFailure, err)
		return &token
	}
	return LoadTokenFromValue(string(dat), appUrl, validationKeys, pluginId)
}

func unwrapSignedJWT(keys map[string]string, parsed *jose.JSONWebSignature) ([]byte, error) {
	if len(parsed.Signatures) < 1 {
		return nil, fmt.Errorf("%w: %w", errParsing, errors.New("no signature found"))
	}
	signature := parsed.Signatures[0]

	k, ok := keys[signature.Protected.KeyID]
	if !ok {
		return nil, errVerificationKeyNotFound
	}

	var jwk jose.JSONWebKey
	err := jwk.UnmarshalJSON([]byte(k))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLicenseVerificationKey, err)
	}

	if signature.Protected.Algorithm != jwk.Algorithm {
		return nil, fmt.Errorf("%w: %w", errParsing, errors.New("invalid algorithm"))
	}

	payload, err := parsed.Verify(jwk)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errVerifyToken, err)
	}

	logger.Debug("license token validated",
		"headers", signature.Protected,
		"keyID", jwk.KeyID,
		"algorithm", jwk.Algorithm)
	return payload, nil
}

func keySet(validationKeys string) (map[string]string, error) {
	if validationKeys == "" {
		return embeddedKeys, nil
	}

	keys := make(map[string]string)
	for keyID, value := range embeddedKeys {
		keys[keyID] = value
	}

	signed, err := jose.ParseSigned(validationKeys)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLicenseVerificationKey, err)
	}

	jwks, err := unwrapSignedJWT(keys, signed)
	if err != nil {
		return nil, fmt.Errorf("failed to load custom validation key: %w", err)
	}

	keySet := jose.JSONWebKeySet{}
	err = json.Unmarshal(jwks, &keySet)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLicenseVerificationKey, err)
	}

	for _, key := range keySet.Keys {
		rawKey, err := json.Marshal(key)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errLicenseVerificationKey, err)
		}
		if other, exists := keys[key.KeyID]; exists {
			// If duplicates was handled as an error, we couldn't
			// add a new key to the static keys list.
			logger.Debug("license validation key duplicate detected, using embedded",
				"keyID", key.KeyID,
				"embedded", other,
				"provided", string(rawKey),
			)
			continue
		}
		keys[key.KeyID] = string(rawKey)
	}

	return keys, nil
}
