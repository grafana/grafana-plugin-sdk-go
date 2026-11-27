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
	// AccessTokenHeader is the header that carries the caller's access token on
	// a plugin protocol v2 call.
	AccessTokenHeader = "X-Access-Token"

	// AccessTokenMetadataKey is the gRPC metadata key that carries the caller's
	// access token on a plugin protocol v3 request.
	AccessTokenMetadataKey = "x-access-token"

	// idTokenMetadataKey is the gRPC metadata key for an ID token. A v3 request
	// that has one is rejected: the access token must be the only identity.
	idTokenMetadataKey = "x-id-token"
)

// AuthenticateAccessToken checks token with authenticator, and returns ctx
// with the caller's AuthInfo. Unlike authlib's AuthInfo, it keeps the token
// (GetAccessToken), so clients can exchange it to act on the caller's behalf.
//
// An invalid token returns a codes.Unauthenticated error. A failure of the
// authenticator itself, such as fetching signing keys, returns codes.Internal.
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
	return types.WithAuthInfo(ctx, &authenticatedAuthInfo{AuthInfo: info, accessToken: token}), nil
}

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

// authenticatedAuthInfo is an AuthInfo that keeps its access token. Printing
// or logging it leaves the token out.
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
