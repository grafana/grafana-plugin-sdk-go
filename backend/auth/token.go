package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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
// with the caller's AuthInfo. Authenticators must retain the access token in
// AuthInfo.GetAccessToken so clients can exchange it to act on the caller's
// behalf. Formatting or logging the returned AuthInfo omits the token.
//
// An invalid token returns a codes.Unauthenticated error. A failure of the
// authenticator itself, such as fetching signing keys, returns codes.Internal.
// Cancellation and timeouts return codes.Canceled and codes.DeadlineExceeded.
func AuthenticateAccessToken(ctx context.Context, authenticator authn.Authenticator, token string) (context.Context, error) {
	if authenticator == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin has no authenticator configured")
	}
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "an access token is required")
	}
	info, err := authenticator.Authenticate(ctx, authn.NewGRPCTokenProvider(metadata.Pairs(AccessTokenMetadataKey, token)))
	if err != nil {
		// Key retrieval can discard the underlying context error. Check the
		// request context as well, and never expose an authenticator's error text.
		if ctx.Err() != nil {
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		if errors.Is(err, context.Canceled) {
			return nil, status.FromContextError(context.Canceled).Err()
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, status.FromContextError(context.DeadlineExceeded).Err()
		}
		if isInvalidTokenErr(err) {
			return nil, status.Error(codes.Unauthenticated, "invalid access token")
		}
		return nil, status.Error(codes.Internal, "authentication failed")
	}
	if info == nil {
		return nil, status.Error(codes.Unauthenticated, "authenticator returned no identity")
	}
	return types.WithAuthInfo(ctx, &authenticatedAuthInfo{AuthInfo: info}), nil
}

// authenticatedAuthInfo wraps an AuthInfo to omit tokens when printing or logging.
type authenticatedAuthInfo struct {
	types.AuthInfo
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
