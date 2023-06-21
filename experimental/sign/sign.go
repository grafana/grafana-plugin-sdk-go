package sign

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
)

// JWTSigner is an interface for signing data.
type JWTSigner interface {
	// Sign returns raw signature for the given data. This method
	// will apply the hash specified for the key type to the data.
	Sign(data interface{}) (string, error)
	// Alg returns the algorithm used to create the signature.
	Alg() string
}

type rsaPrivateKey struct {
	*rsa.PrivateKey
}

func (r *rsaPrivateKey) Sign(data interface{}) (string, error) {
	return signJWT(jose.SignatureAlgorithm(r.Alg()), r.PrivateKey, data)
}

func (r *rsaPrivateKey) Alg() string {
	return "RS256"
}

type ecdsaPrivateKey struct {
	*ecdsa.PrivateKey
}

func (r *ecdsaPrivateKey) Sign(data interface{}) (string, error) {
	return signJWT(jose.SignatureAlgorithm(r.Alg()), r.PrivateKey, data)
}

func (r *ecdsaPrivateKey) Alg() string {
	return "ES256"
}

func newSignerFromKey(k interface{}) (JWTSigner, error) {
	var sshKey JWTSigner
	switch t := k.(type) {
	case *rsa.PrivateKey:
		sshKey = &rsaPrivateKey{t}
	case *ecdsa.PrivateKey:
		sshKey = &ecdsaPrivateKey{t}
	default:
		return nil, fmt.Errorf("crypto: unsupported key type %T", k)
	}
	return sshKey, nil
}

func signJWT(alg jose.SignatureAlgorithm, privateKey interface{}, payload interface{}) (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: alg, Key: privateKey}, &jose.SignerOptions{})
	if err != nil {
		return "", err
	}

	result, err := jwt.Signed(signer).Claims(payload).CompactSerialize()
	if err != nil {
		return "", err
	}
	return result, nil
}

// ParsePrivateKey parses a PEM encoded private key.
func ParsePrivateKey(pemBytes []byte) (JWTSigner, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("crypto: no key found")
	}

	var rawkey interface{}
	switch block.Type {
	case "RSA PRIVATE KEY":
		rsa, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rawkey = rsa
	case "PRIVATE KEY":
		ecdsa, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rawkey = ecdsa
	default:
		return nil, fmt.Errorf("crypto: unsupported private key type %q", block.Type)
	}
	return newSignerFromKey(rawkey)
}
