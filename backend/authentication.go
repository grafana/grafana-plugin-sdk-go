package backend

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend/auth"
)

// withAuthentication resolves opts.Authentication for pluginID (see
// auth.Settings.Resolve), and returns opts with the handler middlewares from
// defaults, then the middleware that authenticates plugin protocol v2 calls,
// then opts' own middlewares. The gRPC server authenticates plugin protocol v3
// requests with the same settings (see defaultGRPCMiddlewares).
func withAuthentication(pluginID string, opts ServeOpts, defaults func() []HandlerMiddleware) (ServeOpts, error) {
	settings, err := opts.Authentication.Resolve(pluginID)
	if err != nil {
		return opts, fmt.Errorf("authentication: %w", err)
	}
	opts.Authentication = settings

	switch {
	case settings.Authenticator != nil:
	case settings.InsecureSkipAuthentication:
		// Anything that can reach the plugin can then claim any identity, so make that visible.
		Logger.Warn("requests are not verified: callers can claim any identity; use this only for local development", "pluginId", pluginID)
	case len(opts.ExtraPlugins) > 0:
		Logger.Warn("no authenticator configured: plugin protocol v3 requests will be rejected; set "+auth.EnvVarGrafanaJWKSURL+" or "+auth.EnvVarGrafanaJWKS+"; for local development, set "+auth.EnvVarInsecureSkipAuthentication+"=true", "pluginId", pluginID)
	default:
		Logger.Debug("no authenticator configured: calls with an access token are served without the caller's identity", "pluginId", pluginID)
	}

	middlewares := append(defaults(), newAuthenticationMiddleware(auth.NewRequestAuthenticator(settings)))
	opts.HandlerMiddlewares = append(middlewares, opts.HandlerMiddlewares...)
	return opts, nil
}

// newAuthenticationMiddleware authenticates every plugin protocol v2 call that
// carries request headers (QueryData, QueryChunkedData, CheckHealth,
// CallResource, and the stream calls) with the access token in its
// auth.AccessTokenHeader (see auth.RequestAuthenticator.AuthenticateHeader).
// The handler's context then carries the caller's AuthInfo. CollectMetrics has
// no headers to carry a token, and passes through unchanged.
func newAuthenticationMiddleware(a *auth.RequestAuthenticator) HandlerMiddleware {
	return HandlerMiddlewareFunc(func(next Handler) Handler {
		return &authenticationMiddleware{BaseHandler: NewBaseHandler(next), auth: a}
	})
}

type authenticationMiddleware struct {
	BaseHandler
	auth *auth.RequestAuthenticator
}

func (m *authenticationMiddleware) authenticate(ctx context.Context, pluginCtx PluginContext, headers interface{ GetHTTPHeader(string) string }) (context.Context, error) {
	return m.auth.AuthenticateHeader(ctx, pluginCtx.Namespace, headers.GetHTTPHeader(auth.AccessTokenHeader))
}

func (m *authenticationMiddleware) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	if req != nil {
		var err error
		if ctx, err = m.authenticate(ctx, req.PluginContext, req); err != nil {
			return nil, err
		}
	}
	return m.BaseHandler.QueryData(ctx, req)
}

func (m *authenticationMiddleware) QueryChunkedData(ctx context.Context, req *QueryChunkedDataRequest, w ChunkedDataWriter) error {
	if req != nil {
		var err error
		if ctx, err = m.authenticate(ctx, req.PluginContext, req); err != nil {
			return err
		}
	}
	return m.BaseHandler.QueryChunkedData(ctx, req, w)
}

func (m *authenticationMiddleware) CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
	if req != nil {
		var err error
		if ctx, err = m.authenticate(ctx, req.PluginContext, req); err != nil {
			return err
		}
	}
	return m.BaseHandler.CallResource(ctx, req, sender)
}

func (m *authenticationMiddleware) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	if req != nil {
		var err error
		if ctx, err = m.authenticate(ctx, req.PluginContext, req); err != nil {
			return nil, err
		}
	}
	return m.BaseHandler.CheckHealth(ctx, req)
}

func (m *authenticationMiddleware) SubscribeStream(ctx context.Context, req *SubscribeStreamRequest) (*SubscribeStreamResponse, error) {
	if req != nil {
		var err error
		if ctx, err = m.authenticate(ctx, req.PluginContext, req); err != nil {
			return nil, err
		}
	}
	return m.BaseHandler.SubscribeStream(ctx, req)
}

func (m *authenticationMiddleware) PublishStream(ctx context.Context, req *PublishStreamRequest) (*PublishStreamResponse, error) {
	if req != nil {
		var err error
		if ctx, err = m.authenticate(ctx, req.PluginContext, req); err != nil {
			return nil, err
		}
	}
	return m.BaseHandler.PublishStream(ctx, req)
}

func (m *authenticationMiddleware) RunStream(ctx context.Context, req *RunStreamRequest, sender *StreamSender) error {
	if req != nil {
		var err error
		if ctx, err = m.authenticate(ctx, req.PluginContext, req); err != nil {
			return err
		}
	}
	return m.BaseHandler.RunStream(ctx, req, sender)
}
