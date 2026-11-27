package auth

import (
	"errors"

	"github.com/go-jose/go-jose/v4"
	josejson "github.com/go-jose/go-jose/v4/json"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/grafana/authlib/authn"
)

// TODO: remove this compatibility fallback and use authn.IsUnauthenticatedErr
// directly once the pinned authlib version includes full JWT error classification.

// invalidTokenErrs are go-jose errors about the token itself, which authlib
// does not mark as unauthenticated.
var invalidTokenErrs = []error{
	jose.ErrCryptoFailure, // Signature mismatch, including a key of the wrong type.
	jwt.ErrNotValidYet,
	jwt.ErrIssuedInTheFuture,
	jwt.ErrExpired,
	jwt.ErrInvalidAudience,
	jwt.ErrInvalidIssuer,
	jwt.ErrInvalidSubject,
	jwt.ErrInvalidID,
	jwt.ErrInvalidClaims,
	jwt.ErrInvalidContentType,
	jwt.ErrUnmarshalAudience,
	jwt.ErrUnmarshalNumericDate,
}

// isInvalidTokenErr reports whether err is caused by a bad token, rather than
// by the plugin, such as failing to fetch signing keys.
func isInvalidTokenErr(err error) bool {
	if errors.Is(err, authn.ErrFetchingSigningKey) {
		return false
	}
	if authn.IsUnauthenticatedErr(err) {
		return true
	}
	// go-jose uses its own JSON decoder for signed claims. Invalid JSON or
	// a claim with the wrong type is a bad token, not an authenticator failure.
	if _, ok := errors.AsType[*josejson.SyntaxError](err); ok {
		return true
	}
	if _, ok := errors.AsType[*josejson.UnmarshalTypeError](err); ok {
		return true
	}
	for _, target := range invalidTokenErrs {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
