package httpresource

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// New creates a new backend.CallResourceHandler adapter that provides
// support for handling resource calls using an http.Handler.
func New(handler http.Handler) backend.CallResourceHandler {
	return &httpResourceHandler{
		handler: handler,
	}
}

type httpResourceHandler struct {
	handler http.Handler
}

func (h *httpResourceHandler) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	var reqBodyReader io.Reader
	if len(req.Body) > 0 {
		reqBodyReader = bytes.NewReader(req.Body)
	}

	ctx = withPluginConfig(ctx, req.PluginConfig)
	ctx = withUser(ctx, req.User)
	reqURL, err := url.Parse(req.URL)
	if err != nil {
		return err
	}

	resourceURL := req.Path
	if reqURL.RawQuery != "" {
		resourceURL += "?" + reqURL.RawQuery
	}

	if !strings.HasPrefix(resourceURL, "/") {
		resourceURL = "/" + resourceURL
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, resourceURL, reqBodyReader)
	if err != nil {
		return err
	}

	for key, values := range req.Headers {
		httpReq.Header[key] = values
	}

	writer := newResponseWriter(sender)
	h.handler.ServeHTTP(writer, httpReq)
	writer.close()

	return nil
}

type pluginConfigKey struct{}

func withPluginConfig(ctx context.Context, cfg backend.PluginConfig) context.Context {
	return context.WithValue(ctx, pluginConfigKey{}, cfg)
}

// PluginConfigFromContext returns plugin config from context.
func PluginConfigFromContext(ctx context.Context) backend.PluginConfig {
	v := ctx.Value(pluginConfigKey{})
	if v == nil {
		return backend.PluginConfig{}
	}

	return v.(backend.PluginConfig)
}

type userKey struct{}

func withUser(ctx context.Context, cfg *backend.User) context.Context {
	return context.WithValue(ctx, userKey{}, cfg)
}

// UserFromContext returns user from context.
func UserFromContext(ctx context.Context) *backend.User {
	v := ctx.Value(userKey{})
	if v == nil {
		return nil
	}

	return v.(*backend.User)
}
