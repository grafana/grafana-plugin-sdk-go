package backend

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend/auth"
)

// withAuthentication resolves opts.Authentication from the environment, and
// sets the handler middlewares to: defaults, then the v2 token check, then the
// plugin's own. The gRPC server checks v3 requests with the same settings (see
// defaultGRPCMiddlewares).
func withAuthentication(pluginID string, opts ServeOpts, defaults func() []HandlerMiddleware) (ServeOpts, error) {
	settings, err := opts.Authentication.Resolve(pluginID)
	if err != nil {
		return opts, fmt.Errorf("authentication: %w", err)
	}
	opts.Authentication = settings

	switch {
	case settings.Authenticator != nil:
	case settings.InsecureSkipAuthentication:
		Logger.Warn("Access tokens are not checked: callers can claim any identity; use this only for local development", "pluginId", pluginID)
	case len(opts.ExtraPlugins) > 0:
		Logger.Warn("No authenticator configured: plugin protocol v3 requests will be rejected; set "+auth.EnvVarGrafanaJWKSURL+" or "+auth.EnvVarGrafanaJWKS+"; for local development, set "+auth.EnvVarInsecureSkipAuthentication+"=true", "pluginId", pluginID)
	default:
		Logger.Debug("No authenticator configured: access tokens will be ignored", "pluginId", pluginID)
	}

	middlewares := append(defaults(), newAuthenticationMiddleware(auth.NewRequestAuthenticator(settings)))
	opts.HandlerMiddlewares = append(middlewares, opts.HandlerMiddlewares...)
	return opts, nil
}

// newAuthenticationMiddleware checks the access token on every plugin protocol
// v2 call that has headers (see auth.RequestAuthenticator.AuthenticateHeader),
// and gives the handler the caller's AuthInfo. CollectMetrics has no headers,
// so it passes through.
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
