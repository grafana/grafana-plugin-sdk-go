// Package auth authenticates the requests a plugin serves with the access
// tokens Grafana sends on behalf of their callers. The plugin SDK applies it to
// every plugin protocol v2 and v3 request; see Settings.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/grafana/authlib/authn"
	"github.com/grafana/authlib/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// AccessTokenHeader is the header in which Grafana sends a plugin protocol
	// v2 call the caller's access token, exchanged for the plugin.
	AccessTokenHeader = "X-Access-Token"

	// AccessTokenMetadataKey is the gRPC metadata key in which Grafana sends a
	// plugin protocol v3 request the caller's access token.
	AccessTokenMetadataKey = "x-access-token"

	// IDTokenMetadataKey is the gRPC metadata key of an ID token. Plugin
	// protocol v3 requests are rejected if they carry one besides the access
	// token, since delegation has one source of identity.
	IDTokenMetadataKey = "x-id-token"
)

// AuthenticateAccessToken verifies token with authenticator, and returns ctx
// carrying the caller's AuthInfo. That AuthInfo keeps the token
// (GetAccessToken), which authlib's does not, so that clients can exchange it to
// act on behalf of the caller.
//
// A token that does not verify is a codes.Unauthenticated status error; a
// failure of the authenticator itself, such as fetching signing keys, is
// codes.Internal.
func AuthenticateAccessToken(ctx context.Context, authenticator authn.Authenticator, token string) (context.Context, error) {
	if authenticator == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin has no authenticator configured")
	}
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "an access token is required")
	}
	info, err := authenticator.Authenticate(ctx, authn.NewGRPCTokenProvider(metadata.Pairs(AccessTokenMetadataKey, token)))
	if err != nil {
		if isInvalidTokenErr(err) {
			return nil, status.Error(codes.Unauthenticated, "invalid access token")
		}
		return nil, status.Error(codes.Internal, "authentication failed")
	}
	if info == nil {
		return nil, status.Error(codes.Unauthenticated, "authenticator returned no identity")
	}
	// Keep the token only after successful authentication so downstream
	// clients can exchange it without losing the caller's identity or
	// delegation chain.
	return types.WithAuthInfo(ctx, &authenticatedAuthInfo{AuthInfo: info, accessToken: token}), nil
}

// invalidTokenErrs are errors about the token itself that authlib passes on
// from go-jose without marking them as unauthenticated.
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

// isInvalidTokenErr reports whether err is the caller's fault, rather than a
// failure of the plugin such as fetching signing keys.
func isInvalidTokenErr(err error) bool {
	if authn.IsUnauthenticatedErr(err) {
		return true
	}
	for _, target := range invalidTokenErrs {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// authenticatedAuthInfo preserves the verified token for onward delegation.
// Its formatting methods omit the token, so logging the identity is safe.
type authenticatedAuthInfo struct {
	types.AuthInfo
	accessToken string
}

func (a *authenticatedAuthInfo) GetAccessToken() string {
	return a.accessToken
}

func (a *authenticatedAuthInfo) String() string {
	return fmt.Sprintf("AuthInfo{subject: %q, namespace: %q}", a.GetSubject(), a.GetNamespace())
}

func (a *authenticatedAuthInfo) GoString() string {
	return a.String()
}

func (a *authenticatedAuthInfo) LogValue() slog.Value {
	return slog.GroupValue(slog.String("subject", a.GetSubject()), slog.String("namespace", a.GetNamespace()))
}
