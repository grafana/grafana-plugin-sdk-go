package backend

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/grafana/authlib/authn"
	"github.com/grafana/authlib/types"
	"github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/grafana-plugin-sdk-go/backend/auth"
)

const authTestPluginID = "my-app"

// authTestTokens returns an authenticator that accepts tokens for
// authTestPluginID, and a function that signs one with the given namespace and
// audience.
func authTestTokens(t *testing.T) (authn.Authenticator, func(namespace, audience string) string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: key},
		(&jose.SignerOptions{}).WithType("at+jwt").WithHeader("kid", "test"))
	require.NoError(t, err)
	jwks, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "test", Algorithm: "ES256", Use: "sig"}}})
	require.NoError(t, err)
	t.Setenv(auth.EnvVarGrafanaJWKSURL, "")
	t.Setenv(auth.EnvVarGrafanaJWKS, string(jwks))
	authenticator, err := auth.NewAuthenticatorFromEnv(authTestPluginID)
	require.NoError(t, err)
	t.Setenv(auth.EnvVarGrafanaJWKS, "")
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
	return authenticator, sign
}

// authRecorder is the plugin's handler: it records the context each call reached it with.
type authRecorder struct {
	ctx context.Context
	// calls is the context each call of authAllCalls reached the handler
	// with, by call name; nil when it did not reach it.
	calls map[string]context.Context
}

func (r *authRecorder) SubscribeStream(ctx context.Context, _ *SubscribeStreamRequest) (*SubscribeStreamResponse, error) {
	r.ctx = ctx
	return &SubscribeStreamResponse{}, nil
}

func (r *authRecorder) PublishStream(ctx context.Context, _ *PublishStreamRequest) (*PublishStreamResponse, error) {
	r.ctx = ctx
	return &PublishStreamResponse{}, nil
}

func (r *authRecorder) RunStream(ctx context.Context, _ *RunStreamRequest, _ *StreamSender) error {
	r.ctx = ctx
	return nil
}

func (r *authRecorder) handler() *Handlers {
	return &Handlers{
		CheckHealthHandler: CheckHealthHandlerFunc(func(ctx context.Context, _ *CheckHealthRequest) (*CheckHealthResult, error) {
			r.ctx = ctx
			return &CheckHealthResult{Status: HealthStatusOk}, nil
		}),
		QueryDataHandler: QueryDataHandlerFunc(func(ctx context.Context, _ *QueryDataRequest) (*QueryDataResponse, error) {
			r.ctx = ctx
			return &QueryDataResponse{}, nil
		}),
		QueryChunkedDataHandler: QueryChunkedDataHandlerFunc(func(ctx context.Context, _ *QueryChunkedDataRequest, _ ChunkedDataWriter) error {
			r.ctx = ctx
			return nil
		}),
		StreamHandler: r,
		CallResourceHandler: CallResourceHandlerFunc(func(ctx context.Context, _ *CallResourceRequest, _ CallResourceResponseSender) error {
			r.ctx = ctx
			return nil
		}),
	}
}

var authNopSender = CallResourceResponseSenderFunc(func(*CallResourceResponse) error { return nil })

func authResource(namespace, token string) *CallResourceRequest {
	req := &CallResourceRequest{PluginContext: PluginContext{PluginID: authTestPluginID, Namespace: namespace}, Path: "clients"}
	if token != "" {
		req.SetHTTPHeader(auth.AccessTokenHeader, token)
	}
	return req
}

// authAllCalls makes each call that carries headers, with the given token,
// against h, and returns the error each returned.
func authAllCalls(h Handler, rec *authRecorder, namespace, token string) map[string]error {
	pc := PluginContext{PluginID: authTestPluginID, Namespace: namespace}
	ctx := context.Background()
	calls := map[string]error{}

	qd := &QueryDataRequest{PluginContext: pc}
	qcd := &QueryChunkedDataRequest{PluginContext: pc}
	ss := &SubscribeStreamRequest{PluginContext: pc}
	ps := &PublishStreamRequest{PluginContext: pc}
	rs := &RunStreamRequest{PluginContext: pc}
	ch := &CheckHealthRequest{PluginContext: pc}
	if token != "" {
		qd.SetHTTPHeader(auth.AccessTokenHeader, token)
		qcd.SetHTTPHeader(auth.AccessTokenHeader, token)
		ss.SetHTTPHeader(auth.AccessTokenHeader, token)
		ps.SetHTTPHeader(auth.AccessTokenHeader, token)
		rs.SetHTTPHeader(auth.AccessTokenHeader, token)
		ch.SetHTTPHeader(auth.AccessTokenHeader, token)
	}

	call := func(name string, err error) {
		calls[name] = err
		rec.calls[name] = rec.ctx
		rec.ctx = nil
	}
	_, err := h.QueryData(ctx, qd)
	call("QueryData", err)
	call("QueryChunkedData", h.QueryChunkedData(ctx, qcd, nil))
	_, err = h.SubscribeStream(ctx, ss)
	call("SubscribeStream", err)
	_, err = h.PublishStream(ctx, ps)
	call("PublishStream", err)
	call("RunStream", h.RunStream(ctx, rs, NewStreamSender(nil)))
	_, err = h.CheckHealth(ctx, ch)
	call("CheckHealth", err)
	call("CallResource", h.CallResource(ctx, authResource(namespace, token), authNopSender))
	return calls
}

func newAuthTestHandler(t *testing.T, s auth.Settings) (Handler, *authRecorder) {
	t.Helper()
	rec := &authRecorder{calls: map[string]context.Context{}}
	h, err := HandlerFromMiddlewares(rec.handler(), newAuthenticationMiddleware(auth.NewRequestAuthenticator(s)))
	require.NoError(t, err)
	return h, rec
}

func TestAuthenticationMiddleware(t *testing.T) {
	authenticator, sign := authTestTokens(t)
	verifying := auth.Settings{Authenticator: authenticator}
	insecure := auth.Settings{InsecureSkipAuthentication: true}

	t.Run("a call without a token is let through with no identity", func(t *testing.T) {
		for name, s := range map[string]auth.Settings{"verifying": verifying, "insecure": insecure, "no authenticator": {}} {
			h, rec := newAuthTestHandler(t, s)
			for call, err := range authAllCalls(h, rec, "default", "") {
				require.NoError(t, err, name+" "+call)
				require.NotNil(t, rec.calls[call], name+" "+call)
				_, ok := types.AuthInfoFrom(rec.calls[call])
				assert.False(t, ok, name+" "+call)
			}
		}
	})

	t.Run("a verified token is the caller, and is kept for delegation", func(t *testing.T) {
		h, rec := newAuthTestHandler(t, verifying)
		token := sign("default", authTestPluginID)
		require.NoError(t, h.CallResource(context.Background(), authResource("default", token), authNopSender))
		info, ok := types.AuthInfoFrom(rec.ctx)
		require.True(t, ok)
		assert.Equal(t, "default", info.GetNamespace())
		assert.Equal(t, token, info.GetAccessToken())
	})

	t.Run("a Bearer prefix is accepted", func(t *testing.T) {
		h, rec := newAuthTestHandler(t, verifying)
		for _, prefix := range []string{"Bearer ", "bearer "} {
			require.NoError(t, h.CallResource(context.Background(), authResource("default", prefix+sign("default", authTestPluginID)), authNopSender))
			_, ok := types.AuthInfoFrom(rec.ctx)
			assert.True(t, ok, prefix)
		}
	})

	t.Run("every call that carries headers is authenticated", func(t *testing.T) {
		h, rec := newAuthTestHandler(t, verifying)

		for name, err := range authAllCalls(h, rec, "default", sign("default", authTestPluginID)) {
			require.NoError(t, err, name)
			_, ok := types.AuthInfoFrom(rec.calls[name])
			assert.True(t, ok, name)
		}
		for name, err := range authAllCalls(h, rec, "default", "not-a-jwt") {
			assert.Equal(t, codes.Unauthenticated, status.Code(err), name)
			assert.Nil(t, rec.calls[name], "%s: the handler must not run", name)
		}
		for name, err := range authAllCalls(h, rec, "stacks-1", sign("stacks-2", authTestPluginID)) {
			assert.Equal(t, codes.PermissionDenied, status.Code(err), name)
			assert.Nil(t, rec.calls[name], "%s: the handler must not run", name)
		}
	})

	t.Run("collecting metrics passes through", func(t *testing.T) {
		called := false
		h, err := HandlerFromMiddlewares(&Handlers{CollectMetricsHandler: CollectMetricsHandlerFunc(func(context.Context, *CollectMetricsRequest) (*CollectMetricsResult, error) {
			called = true
			return &CollectMetricsResult{}, nil
		})}, newAuthenticationMiddleware(auth.NewRequestAuthenticator(verifying)))
		require.NoError(t, err)
		_, err = h.CollectMetrics(context.Background(), &CollectMetricsRequest{})
		require.NoError(t, err)
		assert.True(t, called)
	})

	for name, tt := range map[string]struct {
		settings auth.Settings
		token    string
		ns       string
		code     codes.Code
	}{
		"an invalid token":                    {settings: verifying, token: "not-a-jwt", ns: "default", code: codes.Unauthenticated},
		"a token for another audience":        {settings: verifying, token: sign("default", "other-app"), ns: "default", code: codes.Unauthenticated},
		"a token for another namespace":       {settings: verifying, token: sign("stacks-2", authTestPluginID), ns: "stacks-1", code: codes.PermissionDenied},
		"a token without a namespace":         {settings: verifying, token: sign("", authTestPluginID), ns: "", code: codes.PermissionDenied},
		"a token, insecure with verification": {settings: auth.Settings{Authenticator: authenticator, InsecureSkipAuthentication: true}, token: sign("stacks-2", authTestPluginID), ns: "stacks-1", code: codes.PermissionDenied},
	} {
		t.Run(name+" is rejected", func(t *testing.T) {
			h, rec := newAuthTestHandler(t, tt.settings)
			err := h.CallResource(context.Background(), authResource(tt.ns, tt.token), authNopSender)
			assert.Equal(t, tt.code, status.Code(err))
			assert.Nil(t, rec.ctx, "the handler must not run")
		})
	}

	t.Run("a token without an authenticator is let through with no identity", func(t *testing.T) {
		h, rec := newAuthTestHandler(t, auth.Settings{})
		for name, err := range authAllCalls(h, rec, "default", sign("default", authTestPluginID)) {
			require.NoError(t, err, name)
			require.NotNil(t, rec.calls[name], name)
			_, ok := types.AuthInfoFrom(rec.calls[name])
			assert.False(t, ok, name)
		}
	})

	t.Run("insecure skip authentication parses an unverified token", func(t *testing.T) {
		h, rec := newAuthTestHandler(t, insecure)
		require.NoError(t, h.CallResource(context.Background(), authResource("stacks-1", sign("stacks-2", "other-app")), authNopSender))
		info, ok := types.AuthInfoFrom(rec.ctx)
		require.True(t, ok)
		assert.Equal(t, "stacks-2", info.GetNamespace())
	})
}

func TestWithAuthentication(t *testing.T) {
	_, sign := authTestTokens(t)

	noDefaults := func() []HandlerMiddleware { return nil }
	t.Run("authentication runs after the default middlewares and before the plugin's", func(t *testing.T) {
		var seen bool
		defaults := func() []HandlerMiddleware {
			return []HandlerMiddleware{HandlerMiddlewareFunc(func(next Handler) Handler { return next })}
		}
		opts, err := withAuthentication(authTestPluginID, ServeOpts{
			Authentication: auth.Settings{InsecureSkipAuthentication: true},
			HandlerMiddlewares: []HandlerMiddleware{HandlerMiddlewareFunc(func(next Handler) Handler {
				return &checkHealthSpy{Handler: next, check: func(ctx context.Context) {
					_, seen = types.AuthInfoFrom(ctx)
				}}
			})},
		}, defaults)
		require.NoError(t, err)
		require.Len(t, opts.HandlerMiddlewares, 3)
		_, isAuth := opts.HandlerMiddlewares[1].CreateHandlerMiddleware(&Handlers{}).(*authenticationMiddleware)
		require.True(t, isAuth, "authentication must follow the defaults")

		rec := &authRecorder{calls: map[string]context.Context{}}
		h, err := HandlerFromMiddlewares(rec.handler(), opts.HandlerMiddlewares...)
		require.NoError(t, err)
		req := &CheckHealthRequest{PluginContext: PluginContext{PluginID: authTestPluginID, Namespace: "default"}}
		req.SetHTTPHeader(auth.AccessTokenHeader, sign("default", authTestPluginID))
		_, err = h.CheckHealth(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, seen, "the plugin's middleware must see the caller's identity")
	})

	t.Run("signing keys in the environment are resolved", func(t *testing.T) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		jwks, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "k", Algorithm: "ES256", Use: "sig"}}})
		require.NoError(t, err)
		t.Setenv(auth.EnvVarGrafanaJWKS, string(jwks))
		opts, err := withAuthentication(authTestPluginID, ServeOpts{ExtraPlugins: plugin.PluginSet{}}, noDefaults)
		require.NoError(t, err)
		assert.NotNil(t, opts.Authentication.Authenticator)
	})

	t.Run("invalid signing keys fail", func(t *testing.T) {
		t.Setenv(auth.EnvVarGrafanaJWKS, "not json")
		_, err := withAuthentication(authTestPluginID, ServeOpts{}, noDefaults)
		require.ErrorContains(t, err, "authentication")
	})
}

type checkHealthSpy struct {
	Handler
	check func(context.Context)
}

func (s *checkHealthSpy) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	s.check(ctx)
	return s.Handler.CheckHealth(ctx, req)
}
