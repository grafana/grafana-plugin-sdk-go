package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/grafana/authlib/authn"
	"github.com/grafana/authlib/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authenticatorFunc func(context.Context, authn.TokenProvider) (types.AuthInfo, error)

func (f authenticatorFunc) Authenticate(ctx context.Context, provider authn.TokenProvider) (types.AuthInfo, error) {
	return f(ctx, provider)
}

func TestAuthenticateAccessToken(t *testing.T) {
	info := authn.NewAccessTokenAuthInfo(authn.Claims[authn.AccessTokenClaims]{Subject: "user:u1", Rest: authn.AccessTokenClaims{Namespace: "stacks-1"}})
	for _, tt := range []struct {
		name            string
		token           string
		authErr         error
		noIdentity      bool
		noAuthenticator bool
		want            codes.Code
	}{
		{name: "authenticated", token: "valid"},
		{name: "missing token", want: codes.Unauthenticated},
		{name: "no authenticator", token: "valid", noAuthenticator: true, want: codes.FailedPrecondition},
		{name: "invalid token", token: "valid", authErr: authn.ErrInvalidAudience, want: codes.Unauthenticated},
		{name: "authenticator failure", token: "valid", authErr: errors.New("unavailable"), want: codes.Internal},
		{name: "signing keys unavailable", token: "valid", authErr: fmt.Errorf("failed to verify access token: %w", authn.ErrFetchingSigningKey), want: codes.Internal},
		{name: "signature mismatch", token: "valid", authErr: fmt.Errorf("failed to verify access token: %w", jose.ErrCryptoFailure), want: codes.Unauthenticated},
		{name: "not yet valid", token: "valid", authErr: fmt.Errorf("failed to verify access token: %w", jwt.ErrNotValidYet), want: codes.Unauthenticated},
		{name: "issued in the future", token: "valid", authErr: fmt.Errorf("failed to verify access token: %w", jwt.ErrIssuedInTheFuture), want: codes.Unauthenticated},
		{name: "expired", token: "valid", authErr: fmt.Errorf("failed to verify access token: %w", jwt.ErrExpired), want: codes.Unauthenticated},
		{name: "no identity", token: "valid", noIdentity: true, want: codes.Unauthenticated},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var authenticator authn.Authenticator
			if !tt.noAuthenticator {
				authenticator = authenticatorFunc(func(ctx context.Context, provider authn.TokenProvider) (types.AuthInfo, error) {
					token, ok := provider.AccessToken(ctx)
					require.True(t, ok)
					require.Equal(t, tt.token, token)
					if tt.noIdentity {
						return nil, nil
					}
					return info, tt.authErr
				})
			}
			ctx, err := AuthenticateAccessToken(context.Background(), authenticator, tt.token)
			require.Equal(t, tt.want, status.Code(err))
			if tt.want != codes.OK {
				require.Nil(t, ctx)
				return
			}
			got, ok := types.AuthInfoFrom(ctx)
			require.True(t, ok)
			require.Same(t, info, got.(*authenticatedAuthInfo).AuthInfo)
			require.Equal(t, tt.token, got.GetAccessToken())
		})
	}
}

func TestAuthenticatedAuthInfoOmitsTheToken(t *testing.T) {
	info := &authenticatedAuthInfo{
		AuthInfo:    authn.NewAccessTokenAuthInfo(authn.Claims[authn.AccessTokenClaims]{Subject: "user:u1", Rest: authn.AccessTokenClaims{Namespace: "stacks-1"}}),
		accessToken: "s3cret-token",
	}
	var logged bytes.Buffer
	slog.New(slog.NewJSONHandler(&logged, nil)).Info("caller", "info", info)
	for _, formatted := range []string{fmt.Sprint(info), fmt.Sprintf("%+v", info), fmt.Sprintf("%#v", info), logged.String()} {
		require.NotContains(t, formatted, "s3cret-token")
		require.Contains(t, formatted, "stacks-1")
	}
}

func TestAuthenticateAccessTokenDoesNotExposeInternalErrors(t *testing.T) {
	_, err := AuthenticateAccessToken(context.Background(), authenticatorFunc(func(context.Context, authn.TokenProvider) (types.AuthInfo, error) {
		return nil, errors.New("fetching https://internal-signer?credential=secret failed")
	}), "valid")
	require.Equal(t, codes.Internal, status.Code(err))
	require.NotContains(t, err.Error(), "secret")
	require.NotContains(t, err.Error(), "internal-signer")
}
