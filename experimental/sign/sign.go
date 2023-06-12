package sign

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// Signer is an interface for signing data.
type Signer interface {
	// Sign returns raw signature for the given data. This method
	// will apply the hash specified for the key type to the data.
	SignSHA256(data []byte) ([]byte, error)
	// Alg returns the algorithm used to create the signature.
	Alg() string
}

type rsaPrivateKey struct {
	*rsa.PrivateKey
}

func (r *rsaPrivateKey) SignSHA256(data []byte) ([]byte, error) {
	h := sha256.New()
	h.Write(data)
	d := h.Sum(nil)
	return rsa.SignPKCS1v15(rand.Reader, r.PrivateKey, crypto.SHA256, d)
}

func (r *rsaPrivateKey) Alg() string {
	return "RS256"
}

type ecdsaPrivateKey struct {
	*ecdsa.PrivateKey
}

func (r *ecdsaPrivateKey) SignSHA256(data []byte) ([]byte, error) {
	h := sha256.New()
	h.Write(data)
	d := h.Sum(nil)

	rr, s, err := ecdsa.Sign(rand.Reader, r.PrivateKey, d)
	if err != nil {
		panic(err)
	}

	keyBytes := 32

	rBytes := rr.Bytes()
	rBytesPadded := make([]byte, keyBytes)
	copy(rBytesPadded[keyBytes-len(rBytes):], rBytes)

	sBytes := s.Bytes()
	sBytesPadded := make([]byte, keyBytes)
	copy(sBytesPadded[keyBytes-len(sBytes):], sBytes)

	return append(rBytesPadded, sBytesPadded...), nil
}

func (r *ecdsaPrivateKey) Alg() string {
	return "ES256"
}

func newSignerFromKey(k interface{}) (Signer, error) {
	var sshKey Signer
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

// ParsePrivateKey parses a PEM encoded private key.
func ParsePrivateKey(pemBytes []byte) (Signer, error) {
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
