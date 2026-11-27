package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/grafana/authlib/authn"
	"github.com/grafana/authlib/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const serverTestPluginID = "my-app"

// serverTestTokens returns an authenticator that accepts tokens for
// serverTestPluginID, and a function that signs one with the given namespace
// and audience.
func serverTestTokens(t *testing.T) (authn.Authenticator, func(namespace, audience string) string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: key},
		(&jose.SignerOptions{}).WithType("at+jwt").WithHeader("kid", "test"))
	require.NoError(t, err)
	keys := staticKeys{"test": {Key: &key.PublicKey, KeyID: "test", Algorithm: "ES256", Use: "sig"}}
	verifier := authn.NewAccessTokenVerifier(authn.VerifierConfig{AllowedAudiences: []string{serverTestPluginID}}, keys)
	sign := func(namespace, audience string) string {
		token, err := jwt.Signed(signer).Claims(jwt.Claims{
			Subject:  "user:u1",
			Audience: jwt.Audience{audience},
			Expiry:   jwt.NewNumericDate(time.Now().Add(time.Minute)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		}).Claims(map[string]any{"namespace": namespace}).Serialize()
		require.NoError(t, err)
		return token
	}
	return authn.NewAccessTokenAuthenticator(verifier), sign
}

func TestTrimBearer(t *testing.T) {
	for header, want := range map[string]string{
		"":             "",
		"  ":           "",
		"abc":          "abc",
		" abc ":        "abc",
		"Bearer abc":   "abc",
		"bearer abc":   "abc",
		"BEARER  abc ": "abc",
		"Bearer":       "Bearer",
		"Bearerabc":    "Bearerabc",
	} {
		require.Equal(t, want, trimBearer(header), header)
	}
}

func TestServerInterceptors(t *testing.T) {
	authenticator, sign := serverTestTokens(t)
	const v3Method = "/grafana.plugin.v3.RouteService/CallRoute"
	const v2Method = "/pluginv2.Resource/CallResource"

	incoming := func(pairs ...string) context.Context {
		return metadata.NewIncomingContext(context.Background(), metadata.Pairs(pairs...))
	}
	token := sign("stacks-1", serverTestPluginID)

	for _, tt := range []struct {
		name     string
		settings Settings
		method   string
		ctx      context.Context
		want     codes.Code
		// identity is whether the handler gets an AuthInfo.
		identity bool
	}{
		{name: "verified token", settings: Settings{Authenticator: authenticator}, method: v3Method, ctx: incoming(AccessTokenMetadataKey, token), identity: true},
		{name: "missing token", settings: Settings{Authenticator: authenticator}, method: v3Method, ctx: context.Background(), want: codes.Unauthenticated},
		{name: "invalid token", settings: Settings{Authenticator: authenticator}, method: v3Method, ctx: incoming(AccessTokenMetadataKey, "not-a-jwt"), want: codes.Unauthenticated},
		{name: "token for another audience", settings: Settings{Authenticator: authenticator}, method: v3Method, ctx: incoming(AccessTokenMetadataKey, sign("stacks-1", "other-app")), want: codes.Unauthenticated},
		{name: "two tokens", settings: Settings{Authenticator: authenticator}, method: v3Method, ctx: incoming(AccessTokenMetadataKey, token, AccessTokenMetadataKey, token), want: codes.Unauthenticated},
		{name: "a separate ID token", settings: Settings{Authenticator: authenticator}, method: v3Method, ctx: incoming(AccessTokenMetadataKey, token, idTokenMetadataKey, "id"), want: codes.Unauthenticated},
		{name: "no authenticator rejects requests", settings: Settings{}, method: v3Method, ctx: incoming(AccessTokenMetadataKey, token), want: codes.FailedPrecondition},
		{name: "insecure without a token", settings: Settings{InsecureSkipAuthentication: true}, method: v3Method, ctx: context.Background()},
		{name: "insecure with an unverified token", settings: Settings{InsecureSkipAuthentication: true}, method: v3Method, ctx: incoming(AccessTokenMetadataKey, sign("stacks-2", "other-app")), identity: true},
		{name: "insecure with a separate ID token", settings: Settings{InsecureSkipAuthentication: true}, method: v3Method, ctx: incoming(idTokenMetadataKey, "id"), want: codes.Unauthenticated},
		{name: "insecure does not apply with an authenticator", settings: Settings{Authenticator: authenticator, InsecureSkipAuthentication: true}, method: v3Method, ctx: context.Background(), want: codes.Unauthenticated},
		{name: "v2 methods are left to the handler middleware", settings: Settings{}, method: v2Method, ctx: incoming(AccessTokenMetadataKey, token)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := NewRequestAuthenticator(tt.settings)
			check := func(ctx context.Context) {
				info, ok := types.AuthInfoFrom(ctx)
				require.Equal(t, tt.identity, ok)
				if ok {
					require.NotEmpty(t, info.GetAccessToken())
				}
			}

			t.Run("unary", func(t *testing.T) {
				called := false
				_, err := a.UnaryServerInterceptor()(tt.ctx, nil, &grpc.UnaryServerInfo{FullMethod: tt.method}, func(ctx context.Context, _ any) (any, error) {
					called = true
					check(ctx)
					return nil, nil
				})
				require.Equal(t, tt.want, status.Code(err))
				require.Equal(t, tt.want == codes.OK, called)
			})

			t.Run("stream", func(t *testing.T) {
				called := false
				err := a.StreamServerInterceptor()(nil, &testServerStream{ctx: tt.ctx}, &grpc.StreamServerInfo{FullMethod: tt.method}, func(_ any, stream grpc.ServerStream) error {
					called = true
					check(stream.Context())
					return nil
				})
				require.Equal(t, tt.want, status.Code(err))
				require.Equal(t, tt.want == codes.OK, called)
			})
		})
	}
}

type testServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *testServerStream) Context() context.Context { return s.ctx }
