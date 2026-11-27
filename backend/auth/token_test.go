package auth

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

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
		{name: "authenticator canceled", token: "valid", authErr: fmt.Errorf("secret: %w", context.Canceled), want: codes.Canceled},
		{name: "authenticator timed out", token: "valid", authErr: fmt.Errorf("secret: %w", context.DeadlineExceeded), want: codes.DeadlineExceeded},
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
				require.NotContains(t, err.Error(), "secret")
				require.Nil(t, ctx)
				return
			}
			got, ok := types.AuthInfoFrom(ctx)
			require.True(t, ok)
			require.Same(t, info, got.(*authenticatedAuthInfo).AuthInfo)
			require.Equal(t, info.GetAccessToken(), got.GetAccessToken())
		})
	}
}

func TestAuthenticateAccessTokenMalformedClaims(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: key},
		(&jose.SignerOptions{}).WithType("at+jwt").WithHeader("kid", "test"))
	require.NoError(t, err)
	verifier := authn.NewAccessTokenVerifier(authn.VerifierConfig{AllowedAudiences: []string{"my-app"}},
		staticKeys{"test": {Key: &key.PublicKey, KeyID: "test"}})
	authenticator := authn.NewAccessTokenAuthenticator(verifier)
	for _, claims := range []string{
		`{"aud":"my-app",`,
		`{"aud":"my-app","sub":123}`,
		`{"aud":"my-app","namespace":123}`,
		`{"aud":"my-app","scopes":"invalid"}`,
	} {
		t.Run(claims, func(t *testing.T) {
			signed, err := signer.Sign([]byte(claims))
			require.NoError(t, err)
			token, err := signed.CompactSerialize()
			require.NoError(t, err)
			ctx, err := AuthenticateAccessToken(context.Background(), authenticator, token)
			require.Equal(t, codes.Unauthenticated, status.Code(err))
			require.Nil(t, ctx)
			require.NotContains(t, err.Error(), claims)
		})
	}
}

func TestAuthenticateAccessTokenContextError(t *testing.T) {
	for _, timedOut := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		want := codes.Canceled
		if timedOut {
			cancel()
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			want = codes.DeadlineExceeded
		}
		authenticator := authenticatorFunc(func(context.Context, authn.TokenProvider) (types.AuthInfo, error) {
			cancel()
			// The JWKS retriever does not wrap the underlying context error.
			return nil, authn.ErrFetchingSigningKey
		})
		got, err := AuthenticateAccessToken(ctx, authenticator, "valid")
		cancel()
		require.Equal(t, want, status.Code(err))
		require.Nil(t, got)
	}
}

func TestAuthenticatedAuthInfoOmitsTheToken(t *testing.T) {
	authenticator, sign := serverTestTokens(t)
	token := sign("stacks-1", serverTestPluginID)
	ctx, err := AuthenticateAccessToken(context.Background(), authenticator, token)
	require.NoError(t, err)
	info, ok := types.AuthInfoFrom(ctx)
	require.True(t, ok)
	require.Equal(t, token, info.GetAccessToken(), "authlib retains the token for delegation")
	var logged bytes.Buffer
	slog.New(slog.NewJSONHandler(&logged, nil)).Info("caller", "info", info)
	for _, formatted := range []string{fmt.Sprint(info), fmt.Sprintf("%+v", info), fmt.Sprintf("%#v", info), logged.String()} {
		require.NotContains(t, formatted, token)
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
