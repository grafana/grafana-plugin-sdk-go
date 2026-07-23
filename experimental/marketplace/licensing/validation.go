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
	"test": `{"kty":"RSA","kid":"test","alg":"RS512","n":"zX2SkEP-DAPIQVHGXjcJ76HIx865tCBOKbVlquvku15lbHlExIiaF2VAfrMYobQpD2Ia2DxLQZUqYci2X1pvZzDhDtTcEpn1_pixiNGzsZXafl5gXbX0G67o-aLKdvTNtJSoN56weAJ2DgF1-P2SB6zSxHla5qOo8MQ2KaDNiq7SeE6k2qV8O6oqkF8E-uTp0Ow4kdDL-yMmohmd-vidJcJiIWmg7DFiJ-ZJ_jUBZH6PXF2QosWDVMxeZmlZZnSrqf_s4NZ8lsx_J3jE2hA8a1YUFCwwpNd_uGbo6prdUsdGZ_2ZIggNpzlwiUlmDIjxbwoTCeh99IRXo3iObuTLaqNYbxnHGk0SNgaHoy3EpOsikITJvEiuLlrnih7M76_ygfPVdE4wqQMAKkaZZ0wUDOQwuVY6bedpePF6h_iHqhzwldGxI_5Q8jTY2hJsm8bv6T6wW1Ml9WN_i-9PHOl3MZQAITVuTun-y0cxWz1ldn-Nh0SEOnbtvMNRwgoKPrN_UyaB1Dkk6AF1gx9mLpltVL4xb602vLJpfL6VEeE1Ca4CN5llFoOR8EJVhF8DBPqSnj3j2Uc3Y5ituUqRT3SZCJh-Vx_6HHWdhypJ3ULAvvK_pd4Os8tdS4JXxJxHeU-e5fJq80uNN9wyDfP8hBNMDfRLkesPswDJH-Xv7zie-M0","e":"AQAB"}`,
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
