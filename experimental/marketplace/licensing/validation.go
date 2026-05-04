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
	errFileNotFound = func(tokenPath string) error {
		return fmt.Errorf("license token file not found: %s", tokenPath)
	}
	errLoadFailure = func(err error) error {
		return fmt.Errorf("error loading license token: %s", err)
	}
	errParsing = func(err error) error {
		return fmt.Errorf("error parsing license token: %s", err)
	}
	errVerificationKeyNotFound = fmt.Errorf("license verification key not found")
	errLicenseVerificationKey  = func(err error) error {
		return fmt.Errorf("error loading license verification key: %s", err)
	}
	errVerifyToken = func(err error) error {
		return fmt.Errorf("error verifying license token: %s", err)
	}
)

// embeddedKeys are the JWKs we use for license validation.
// They are JSON structs keyed by their "kid" field.
//
// If more keys are necessary, see the /scripts/licensing/genjwk script in grafana-enterprise. Remember to save the values somewhere safe, like the current standard password manager.
var embeddedKeys = map[string]string{
	"test": `{"kty":"RSA","kid":"test","alg":"RS512","n":"vEozbbwXShnXoz4kUKycW-awmd2wS3-Gy3peiH0jOn1wzwZJN9Nj2hkZ68BxuHFISMSKTrgC-vEF99kq6CocbxNl-xe5DWR-md4XFjGK1MCsINMp20UUgMCKy6pAQDYrZYT0JXeiDPrSnJbCDwTS6TrRscD10prNo54gS54yDSY5ds53W8O0TnbdjR-VPa5X91kqOzApZTJ0s40XNtHQSETylLD4N7j1BuSYFRm0xmodsSOFIE1Jl4ALuyugptM0F9np7qcLRvwLyHX4qRzBv0ua-9zOZSjIB3hBw4O8ViDYS0MAR92llPgtBngPZ1OZ4hyK09do7gNRXxFdURX9GHm5Lbf01f0SFDzYgXTVZV6wcN4NuSn823owvkcLoeeyIgY2MKYJxDHZCW5dLnNSkHkrOBxTGjTEiL_dX-M4NwqRh5wBZyvqNufQCJWcp-1Ft_zicYsNJNTU7mBG3rBKlMU_ZsMzr2QJjVUIyI0W7nHhI0ymtLWyHxmqHubxOhI7HuhT8dUBFj36K12vx24KxAKz2Vt17j9xw221KiP2q0R31qUYnzS4vIiR7Agz8BIp9XP8MR5GEhS2SQ9syK77bx_YqSbR3u2nofhmNh_5Hm8sZ46SeCdjUcl46Dv4aIiFcaXnpfqTf7d0iMA3ZCPRVIpQX6cRkMDJF-SMIS-Kovs","e":"AQAB"}`,
}

func LoadToken(tokenPath, appUrl, validationKeys, pluginId string) *LicenseToken {
	token := &LicenseToken{}

	// Can ignore gosec G304 since tokenPath is derived from a configuration parameter
	// nolint:gosec
	dat, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			token.Status = NotFound
			token.Error = errFileNotFound(tokenPath)
			return token
		}

		token.Status = Invalid
		token.Error = errLoadFailure(err)
		return token
	}

	token.Parse(string(dat), appUrl, validationKeys, pluginId)
	return token
}

func (token *LicenseToken) Parse(tokenStr, appUrl, validationKeys, pluginId string) bool {
	token.Raw = tokenStr

	parsed, err := jose.ParseSigned(token.Raw)
	if err != nil {
		token.Status = Invalid
		token.Error = errParsing(err)
		return false
	}

	keys, err := keySet(validationKeys)
	if err != nil {
		token.Status = Invalid
		token.Error = err
		return false
	}

	payload, err := unwrapSignedJWT(keys, parsed)
	if err != nil {
		token.Status = Invalid
		token.Error = err
		return false
	}

	err = json.Unmarshal(payload, &token)
	if err != nil {
		token.Status = Invalid
		token.Error = errParsing(err)
		return false
	}

	// Handle tokens with missing or invalid "update_days" field
	if token.UpdateDays < 1 {
		token.UpdateDays = 1
	}

	logger.Debug("license token parsed", "token", token)
	return token.Validate(appUrl, pluginId)
}

func unwrapSignedJWT(keys map[string]string, parsed *jose.JSONWebSignature) ([]byte, error) {
	if len(parsed.Signatures) < 1 {
		return nil, errParsing(errors.New("no signature found"))
	}
	signature := parsed.Signatures[0]

	k, ok := keys[signature.Protected.KeyID]
	if !ok {
		return nil, errVerificationKeyNotFound
	}

	var jwk jose.JSONWebKey
	err := jwk.UnmarshalJSON([]byte(k))
	if err != nil {
		return nil, errLicenseVerificationKey(err)
	}

	if signature.Protected.Algorithm != jwk.Algorithm {
		return nil, errParsing(errors.New("invalid algorithm"))
	}

	payload, err := parsed.Verify(jwk)
	if err != nil {
		return nil, errVerifyToken(err)
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
		return nil, errLicenseVerificationKey(err)
	}

	jwks, err := unwrapSignedJWT(keys, signed)
	if err != nil {
		return nil, fmt.Errorf("failed to load custom validation key: %w", err)
	}

	keySet := jose.JSONWebKeySet{}
	err = json.Unmarshal(jwks, &keySet)
	if err != nil {
		return nil, errLicenseVerificationKey(err)
	}

	for _, key := range keySet.Keys {
		rawKey, err := json.Marshal(key)
		if err != nil {
			return nil, errLicenseVerificationKey(err)
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
