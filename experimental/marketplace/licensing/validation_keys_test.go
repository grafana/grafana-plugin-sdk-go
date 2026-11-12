package licensing

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	jose "gopkg.in/go-jose/go-jose.v2"
)

// testKey is an RSA key generated at test time. Only tests that call trustEmbeddedKey trust it.
type testKey struct {
	kid  string
	priv *rsa.PrivateKey
}

var generatedTestKeys = sync.OnceValue(func() map[string]testKey {
	keys := map[string]testKey{}
	for _, kid := range []string{"test-root", "test-untrusted-root", "test-child-1"} {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		keys[kid] = testKey{kid: kid, priv: priv}
	}
	return keys
})

func testKeys(t *testing.T) (root, untrustedRoot, child1 testKey) {
	t.Helper()
	keys := generatedTestKeys()
	return keys["test-root"], keys["test-untrusted-root"], keys["test-child-1"]
}

func (k testKey) publicJWK(kid string) jose.JSONWebKey {
	return jose.JSONWebKey{Key: &k.priv.PublicKey, KeyID: kid, Algorithm: string(jose.RS512), Use: "sig"}
}

func (k testKey) sign(t *testing.T, payload []byte) string {
	t.Helper()
	compact, err := k.signed(t, payload).CompactSerialize()
	require.NoError(t, err)
	return compact
}

func (k testKey) signed(t *testing.T, payload []byte) *jose.JSONWebSignature {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS512, Key: jose.JSONWebKey{Key: k.priv, KeyID: k.kid}}, nil)
	require.NoError(t, err)
	signed, err := signer.Sign(payload)
	require.NoError(t, err)
	return signed
}

// token returns a currently valid license token for testPluginID signed by k.
func (k testKey) token(t *testing.T) string {
	t.Helper()
	now := time.Now()
	claims, err := json.Marshal(map[string]any{
		"jti":  "1",
		"sub":  appURL,
		"iat":  now.Add(-time.Hour).Unix(),
		"exp":  now.Add(24 * time.Hour).Unix(),
		"nbf":  now.Add(-time.Hour).Unix(),
		"lexp": now.Add(30 * 24 * time.Hour).Unix(),
		"lid":  "1",
		"prod": []string{testProductID},
	})
	require.NoError(t, err)
	return k.sign(t, claims)
}

// bundle returns a validation key bundle with the public keys of keys, signed by k.
func (k testKey) bundle(t *testing.T, keys ...jose.JSONWebKey) string {
	t.Helper()
	jwks, err := json.Marshal(jose.JSONWebKeySet{Keys: keys})
	require.NoError(t, err)
	return k.sign(t, jwks)
}

// jsonBundle returns a JSON-serialized validation key bundle with the public keys of keys, signed by k.
func (k testKey) jsonBundle(t *testing.T, keys ...jose.JSONWebKey) string {
	t.Helper()
	jwks, err := json.Marshal(jose.JSONWebKeySet{Keys: keys})
	require.NoError(t, err)
	return k.signed(t, jwks).FullSerialize()
}

// trustEmbeddedKey adds k to the embedded keys for the duration of the test.
func trustEmbeddedKey(t *testing.T, k testKey) {
	t.Helper()
	raw, err := json.Marshal(k.publicJWK(k.kid))
	require.NoError(t, err)
	embeddedKeys[k.kid] = string(raw)
	t.Cleanup(func() { delete(embeddedKeys, k.kid) })
}

func denyKeyIDs(t *testing.T, kids ...string) {
	t.Helper()
	previous := deniedKeyIDs
	deniedKeyIDs = map[string]struct{}{}
	for _, kid := range kids {
		deniedKeyIDs[kid] = struct{}{}
	}
	t.Cleanup(func() { deniedKeyIDs = previous })
}

func TestValidationKeyBundles(t *testing.T) {
	root, untrustedRoot, child1 := testKeys(t)
	childBundle := root.bundle(t, child1.publicJWK(child1.kid))
	// impostor signs with child1's private key under the embedded root's key ID.
	impostor := testKey{kid: root.kid, priv: child1.priv}

	for name, tc := range map[string]struct {
		token          testKey
		validationKeys string
		denied         []string
		status         TokenStatus
		err            error
	}{
		"embedded key without bundles": {token: root, status: Valid},
		"child key without bundles":    {token: child1, status: Invalid, err: errVerificationKeyNotFound},
		"child key from a bundle":      {token: child1, validationKeys: childBundle, status: Valid},
		"whitespace is ignored":        {token: child1, validationKeys: " " + childBundle[:20] + "\n" + childBundle[20:] + " ", status: Valid},
		"malformed bundle is ignored":  {token: root, validationKeys: "not-a-bundle", status: Valid},
		"bundle from an untrusted root is ignored": {
			token: root, validationKeys: untrustedRoot.bundle(t, child1.publicJWK(child1.kid)), status: Valid,
		},
		"child key from an untrusted root": {
			token: child1, validationKeys: untrustedRoot.bundle(t, child1.publicJWK(child1.kid)), status: Invalid, err: errVerificationKeyNotFound,
		},
		"JSON-serialized bundle": {
			token: child1, validationKeys: " " + root.jsonBundle(t, child1.publicJWK(child1.kid)), status: Valid,
		},
		"JSON-serialized bundle from an untrusted root": {
			token: child1, validationKeys: untrustedRoot.jsonBundle(t, child1.publicJWK(child1.kid)), status: Invalid, err: errVerificationKeyNotFound,
		},
		"comma-separated bundles are ignored": {
			token: child1, validationKeys: childBundle + "," + childBundle, status: Invalid, err: errVerificationKeyNotFound,
		},
		"bundle key cannot replace an embedded key": {
			token: impostor, validationKeys: root.bundle(t, child1.publicJWK(root.kid)), status: Invalid, err: errVerifyToken,
		},
		"denied child key":    {token: child1, validationKeys: childBundle, denied: []string{child1.kid}, status: Invalid, err: errLicenseInvalid},
		"denied embedded key": {token: root, denied: []string{root.kid}, status: Invalid, err: errLicenseInvalid},
		"bundle signed by a denied key is ignored": {
			token: child1, validationKeys: childBundle, denied: []string{root.kid}, status: Invalid, err: errVerificationKeyNotFound,
		},
	} {
		t.Run(name, func(t *testing.T) {
			trustEmbeddedKey(t, root)
			denyKeyIDs(t, tc.denied...)
			token := LoadTokenFromValue(tc.token.token(t), appURL, tc.validationKeys, testPluginID)

			require.Equal(t, tc.status, token.Status, "error: %v", token.Error)
			if tc.err != nil {
				require.ErrorIs(t, token.Error, tc.err)
			} else {
				require.NoError(t, token.Error)
			}
		})
	}
}

func TestVerifyValidationKeys(t *testing.T) {
	root, untrustedRoot, child1 := testKeys(t)
	trustEmbeddedKey(t, root)
	childBundle := root.bundle(t, child1.publicJWK(child1.kid))

	require.NoError(t, VerifyValidationKeys(""))
	require.NoError(t, VerifyValidationKeys(childBundle))
	require.NoError(t, VerifyValidationKeys(root.jsonBundle(t, child1.publicJWK(child1.kid))))
	require.ErrorIs(t, VerifyValidationKeys("not-a-bundle"), errLicenseVerificationKey)
	require.ErrorIs(t, VerifyValidationKeys(childBundle+","+childBundle), errLicenseVerificationKey)
	require.ErrorIs(t, VerifyValidationKeys(untrustedRoot.bundle(t)), errVerificationKeyNotFound)
	require.ErrorIs(t, VerifyValidationKeys(untrustedRoot.jsonBundle(t)), errVerificationKeyNotFound)
}

// TestForgedValidationKeyBundles checks that a bundle is only trusted with a valid signature by an embedded key.
// Each forged bundle carries child1's public key: it must be ignored, so child1's tokens stay invalid while
// tokens signed by the embedded root still validate.
func TestForgedValidationKeyBundles(t *testing.T) {
	root, untrustedRoot, child1 := testKeys(t)
	trustEmbeddedKey(t, root)
	jwks, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{child1.publicJWK(child1.kid)}})
	require.NoError(t, err)

	sign := func(opts *jose.SignerOptions, keys ...jose.SigningKey) *jose.JSONWebSignature {
		t.Helper()
		signer, err := jose.NewMultiSigner(keys, opts)
		require.NoError(t, err)
		signed, err := signer.Sign(jwks)
		require.NoError(t, err)
		return signed
	}
	compact := func(signed *jose.JSONWebSignature) string {
		t.Helper()
		bundle, err := signed.CompactSerialize()
		require.NoError(t, err)
		return bundle
	}
	rsaKey := func(k testKey, kid string) jose.SigningKey {
		return jose.SigningKey{Algorithm: jose.RS512, Key: jose.JSONWebKey{Key: k.priv, KeyID: kid}}
	}

	// The protected header has no key ID; the embedded root's key ID is only in the unprotected header.
	var unprotectedKeyID map[string]any
	require.NoError(t, json.Unmarshal([]byte(sign(nil, jose.SigningKey{Algorithm: jose.RS512, Key: untrustedRoot.priv}).FullSerialize()), &unprotectedKeyID))
	unprotectedKeyID["header"] = map[string]string{"kid": root.kid}
	unprotectedKeyIDBundle, err := json.Marshal(unprotectedKeyID)
	require.NoError(t, err)

	b64 := base64.RawURLEncoding.EncodeToString
	for name, tc := range map[string]struct {
		bundle string
		err    error
	}{
		"self-signed": {bundle: child1.bundle(t, child1.publicJWK(child1.kid)), err: errVerificationKeyNotFound},
		"embedded root key ID, signed by another key": {
			bundle: compact(sign(nil, rsaKey(untrustedRoot, root.kid))), err: errVerifyToken,
		},
		"embedded root key ID, JSON-serialized and signed by another key": {
			bundle: sign(nil, rsaKey(untrustedRoot, root.kid)).FullSerialize(), err: errVerifyToken,
		},
		"alg none": {
			bundle: b64([]byte(`{"alg":"none","kid":"`+root.kid+`"}`)) + "." + b64(jwks) + ".", err: errParsing,
		},
		"HMAC with the embedded root's public key as secret": {
			bundle: compact(sign((&jose.SignerOptions{}).WithHeader("kid", root.kid), jose.SigningKey{Algorithm: jose.HS512, Key: []byte(embeddedKeys[root.kid])})),
			err:    errParsing,
		},
		"key ID only in the unprotected header": {bundle: string(unprotectedKeyIDBundle), err: errVerificationKeyNotFound},
		"attacker key embedded in the header": {
			bundle: compact(sign((&jose.SignerOptions{EmbedJWK: true}).WithHeader("kid", root.kid), jose.SigningKey{Algorithm: jose.RS512, Key: untrustedRoot.priv})),
			err:    errVerifyToken,
		},
		"several signatures, one by the embedded root": {
			bundle: sign(nil, rsaKey(root, root.kid), rsaKey(untrustedRoot, untrustedRoot.kid)).FullSerialize(), err: errVerifyToken,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, VerifyValidationKeys(tc.bundle), tc.err)

			token := LoadTokenFromValue(child1.token(t), appURL, tc.bundle, testPluginID)
			require.Equal(t, Invalid, token.Status)
			require.ErrorIs(t, token.Error, errVerificationKeyNotFound)

			token = LoadTokenFromValue(root.token(t), appURL, tc.bundle, testPluginID)
			require.Equal(t, Valid, token.Status, "error: %v", token.Error)
		})
	}
}

func TestValidationKeyBundleWithMalformedKeySet(t *testing.T) {
	root, _, _ := testKeys(t)
	trustEmbeddedKey(t, root)
	bundle := root.sign(t, []byte("not a JWKS"))

	require.ErrorIs(t, VerifyValidationKeys(bundle), errLicenseVerificationKey)
	token := LoadTokenFromValue(root.token(t), appURL, bundle, testPluginID)
	require.Equal(t, Valid, token.Status, "error: %v", token.Error)
}

// TestValidationKeyBundleTrustIsPerCall checks that keys from a verified bundle are not remembered: they neither
// validate later tokens without the bundle nor verify later bundles.
func TestValidationKeyBundleTrustIsPerCall(t *testing.T) {
	root, untrustedRoot, child1 := testKeys(t)
	trustEmbeddedKey(t, root)
	embeddedKeyIDs := len(embeddedKeys)

	token := LoadTokenFromValue(child1.token(t), appURL, root.bundle(t, child1.publicJWK(child1.kid)), testPluginID)
	require.Equal(t, Valid, token.Status, "error: %v", token.Error)
	require.Len(t, embeddedKeys, embeddedKeyIDs)

	token = LoadTokenFromValue(child1.token(t), appURL, "", testPluginID)
	require.Equal(t, Invalid, token.Status)
	require.ErrorIs(t, token.Error, errVerificationKeyNotFound)
	require.ErrorIs(t, VerifyValidationKeys(child1.bundle(t, untrustedRoot.publicJWK(untrustedRoot.kid))), errVerificationKeyNotFound)
}
