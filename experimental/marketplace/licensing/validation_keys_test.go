package licensing

import (
	"crypto/rand"
	"crypto/rsa"
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
	for _, kid := range []string{"test-root", "test-untrusted-root", "test-child-1", "test-child-2"} {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		keys[kid] = testKey{kid: kid, priv: priv}
	}
	return keys
})

func testKeys(t *testing.T) (root, untrustedRoot, child1, child2 testKey) {
	t.Helper()
	keys := generatedTestKeys()
	return keys["test-root"], keys["test-untrusted-root"], keys["test-child-1"], keys["test-child-2"]
}

func (k testKey) publicJWK(kid string) jose.JSONWebKey {
	return jose.JSONWebKey{Key: &k.priv.PublicKey, KeyID: kid, Algorithm: string(jose.RS512), Use: "sig"}
}

func (k testKey) sign(t *testing.T, payload []byte) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS512, Key: jose.JSONWebKey{Key: k.priv, KeyID: k.kid}}, nil)
	require.NoError(t, err)
	signed, err := signer.Sign(payload)
	require.NoError(t, err)
	compact, err := signed.CompactSerialize()
	require.NoError(t, err)
	return compact
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
	root, untrustedRoot, child1, child2 := testKeys(t)
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
		"surrounding whitespace":       {token: child1, validationKeys: " " + childBundle + " , ", status: Valid},
		"malformed bundle is ignored":  {token: root, validationKeys: "not-a-bundle", status: Valid},
		"bundle from an untrusted root is ignored": {
			token: root, validationKeys: untrustedRoot.bundle(t, child1.publicJWK(child1.kid)), status: Valid,
		},
		"child key from an untrusted root": {
			token: child1, validationKeys: untrustedRoot.bundle(t, child1.publicJWK(child1.kid)), status: Invalid, err: errVerificationKeyNotFound,
		},
		"other bundles are used when one is ignored": {
			token: child1, validationKeys: "not-a-bundle," + untrustedRoot.bundle(t) + "," + childBundle, status: Valid,
		},
		"bundle keys cannot verify other bundles": {
			token: child2, validationKeys: childBundle + "," + child1.bundle(t, child2.publicJWK(child2.kid)), status: Invalid, err: errVerificationKeyNotFound,
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
	root, untrustedRoot, child1, _ := testKeys(t)
	trustEmbeddedKey(t, root)
	childBundle := root.bundle(t, child1.publicJWK(child1.kid))

	require.NoError(t, VerifyValidationKeys(""))
	require.NoError(t, VerifyValidationKeys(childBundle))

	err := VerifyValidationKeys("not-a-bundle," + childBundle + "," + untrustedRoot.bundle(t))
	require.ErrorIs(t, err, errLicenseVerificationKey)
	require.ErrorIs(t, err, errVerificationKeyNotFound)
	require.Contains(t, err.Error(), "validation key bundle 1:")
	require.NotContains(t, err.Error(), "validation key bundle 2:")
	require.Contains(t, err.Error(), "validation key bundle 3:")
}
