package licensing

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"

	jose "gopkg.in/go-jose/go-jose.v2"
)

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
	"ops-01":     `{"kty":"RSA","kid":"ops-01","alg":"RS512","n":"s4SrDRyR3oXgO1wsXsYqJUC44B8Zhnzi7XTqFHNiyd5i5Dqh4OPikLbRBNLrXQ-bI2y3zbrgXHJWEOEowOZplhB-KlI9CnifDBQY4SzQ_K2nLiOkNmb-V427HnYFIVH1LiSSDkAj7-flOqRKc9PiQImlwdLXwpoY1fV2yHKG0_cSxxNYK0Qfk6aO7jGOUlGFhdrW0E86SJYER-S0bUzFFuUTF4VqkK4j5SxA3oPVm6UHZInHHdyVtzj5YiZb-Nqw51WkNBVcZnOil9GYjXSPnPhKwtjDwk5JXs4PBiAIejzmQMf1Ah1IxO_jBThOlDsDbfREZaLag2VWb3sDwoPwrGnO2HzqHBHXdZeTgiesAODCTgv_tU97vNwt0XYPnrumsFapeJG2j505CWWbLDq9KiwBXnhIvzA1oiik2AXT4RwEcmQ2m-O9LcK8keSwCb7nGwY8SZn-rcJnGdsyDu45fvxxvrXo6n0R2Smy-X8fSbL50t0kj4p8pAp-nqjuMsQHaTwVhuWVTq5FZQDrR1Wv00xtcxBA4q2Yk0eKy9hNqzs50s74JwHx29AhPzMaNdwtcUEFH4guskGd-V_i97xKKe_lRvoh4jDwvnVCMWPvdZbT5UcxDI9kL4dyVYzvGdb7NonnrET5bASsp_f_4aK8zWWClbmwWJ53tjAYUwJ6eQk","e":"AQAB"}`,
	"prod-01":    `{"kty":"RSA","kid":"prod-01","alg":"RS512","n":"yXFDjzKKnKCVBYOV3x5kUPbPT4NBUzclPSi5ixO1_vg8s1lHyOhA13hdPGeZlmjpnVp0WiIbC7yS6eIDGGUdgVCQyk-xAPayupVRZOZjSFzEXgOP4rO2AyYbdTS89hhR5f6JFhoYa6LgGJVCEtlZtQJ1wi_Li8mzZm7bDLZb0MU84DnGPpALAAcHTzn7hs1fT7e-wNjtj2ceEJpTtbuW-coaOmAuOl3pxaq7YqSd4nfIehqtPqmnE9oZrtjU666oZGV9TNq3DOmatWj2BKg9ocd129S2lBo-uyfG1dyaE3vBEQQhIlhuiyKFvl5ThPESU56FNOWGoEGg0Lbs_cpz8jLtsGgna_A9_jb3k0FCa6lmxnIDJkX1Oe9BA6Jm0NQLrzhpaZXAKptF9XIeJ5MvFJGMhhNFs7cHPabQeIoDaHy0a4tfvsD6UBYAm2TpJMtfSxqtbUr0AyuBDspcEw3f0U6jGm5QgG98ApYb1LPGhS8h-NM3IBDpRlPxReCQrLvrJwIpK7enPwH4mjram-shjJm0xgbqbJNxcsPdTYPfAwzsVdOIImxV6k2irD692e_UGxn1YBz-UOG5XCGTnWV0Y_Ac_YbhojozavBO3prGCCqwJMm-O4pZcPdYhjJj9IWuGJHOE0bVisBJyiqewT1X6srSZPQPUm17rW3w4e5LxPE","e":"AQAB"}`,
}

// deniedKeyIDs are the key IDs whose tokens and validation key bundles are never accepted, for example
// because their private key leaked. It covers embedded keys and keys from validation key bundles,
// so key IDs must never be reused. The value of each key is ignored.
var deniedKeyIDs = map[string]struct{}{}

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

	if _, denied := deniedKeyIDs[signature.Protected.KeyID]; denied {
		// Like blocked licenses, report a generic error.
		return nil, errLicenseInvalid
	}
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

	backend.Logger.Debug("license token validated",
		"headers", signature.Protected,
		"keyID", jwk.KeyID,
		"algorithm", jwk.Algorithm)
	return payload, nil
}

// keySet returns the embedded keys plus the keys of every bundle in validationKeys that can be verified.
// Bundles that cannot be verified add no keys, so tokens signed by other keys still validate. They are only
// logged at debug level: Grafana can send bundles signed by roots that older plugin builds do not embed, and
// in request mode validationKeys is parsed on every request. Grafana reports bad bundles itself, with
// VerifyValidationKeys.
func keySet(validationKeys string) map[string]string {
	if validationKeys == "" {
		return embeddedKeys
	}
	keys, errs := loadKeySet(validationKeys)
	for _, err := range errs {
		backend.Logger.Debug("Ignoring marketplace license validation keys that cannot be verified", "error", err)
	}
	return keys
}

// VerifyValidationKeys reports the bundles in validationKeys that cannot be verified, and are therefore
// ignored when validating license tokens. It returns nil if validationKeys is empty.
//
// validationKeys holds one or more validation key bundles separated by commas. A bundle is a JWS whose
// payload is a JWKS of additional public keys, signed by an embedded key.
func VerifyValidationKeys(validationKeys string) error {
	_, errs := loadKeySet(validationKeys)
	return errors.Join(errs...)
}

// loadKeySet returns the embedded keys plus the keys of every bundle that can be verified, and an error for
// each bundle that cannot. Bundles are verified only with embedded keys, so a bundle cannot add keys that
// verify other bundles. A bundle key whose ID is already used keeps the existing key.
func loadKeySet(validationKeys string) (map[string]string, []error) {
	keys := maps.Clone(embeddedKeys)
	var errs []error
	for i, bundle := range strings.Split(validationKeys, ",") {
		bundle = strings.TrimSpace(bundle)
		if bundle == "" {
			continue
		}
		bundleKeys, err := verifyBundle(bundle)
		if err != nil {
			errs = append(errs, fmt.Errorf("validation key bundle %d: %w", i+1, err))
			continue
		}
		for _, key := range bundleKeys {
			rawKey, err := json.Marshal(key)
			if err != nil {
				errs = append(errs, fmt.Errorf("validation key bundle %d: %w: %w", i+1, errLicenseVerificationKey, err))
				continue
			}
			if other, exists := keys[key.KeyID]; exists {
				// If duplicates were handled as an error, a bundle could not add a key next to the embedded keys.
				backend.Logger.Debug("license validation key duplicate detected, using the existing key",
					"keyID", key.KeyID,
					"existing", other,
					"provided", string(rawKey),
				)
				continue
			}
			keys[key.KeyID] = string(rawKey)
		}
	}
	return keys, errs
}

// verifyBundle verifies a validation key bundle with the embedded keys and returns its keys.
func verifyBundle(bundle string) ([]jose.JSONWebKey, error) {
	signed, err := jose.ParseSigned(bundle)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLicenseVerificationKey, err)
	}
	jwks, err := unwrapSignedJWT(embeddedKeys, signed)
	if err != nil {
		return nil, fmt.Errorf("failed to load custom validation key: %w", err)
	}
	keySet := jose.JSONWebKeySet{}
	if err := json.Unmarshal(jwks, &keySet); err != nil {
		return nil, fmt.Errorf("%w: %w", errLicenseVerificationKey, err)
	}
	return keySet.Keys, nil
}
