package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
)

const (
	// OAuthIdentityTokenHeaderName the header name used for forwarding
	// OAuth Identity access token.
	OAuthIdentityTokenHeaderName = "Authorization"

	// OAuthIdentityIDTokenHeaderName the header name used for forwarding
	// OAuth Identity ID token.
	OAuthIdentityIDTokenHeaderName = "X-Id-Token"

	// GrafanaUserSignInTokenHeaderName the header name used for forwarding
	// the SignIn token of a Grafana User.
	// Requires idForwarded feature toggle enabled.
	GrafanaUserSignInTokenHeaderName = "X-Grafana-Id" //nolint:gosec // this is not a hardcoded credential

	// CookiesHeaderName the header name used for forwarding
	// cookies.
	CookiesHeaderName = "Cookie"

	httpHeaderPrefix = "http_"
)

// headersExcludedFromForwarding lets the plugin client negotiate response encoding
// and keeps Grafana's internal sign-in token within Grafana.
var headersExcludedFromForwarding = []string{
	"Accept-Encoding",
	GrafanaUserSignInTokenHeaderName,
}

// ForwardHTTPHeaders interface marking that forward of HTTP headers is supported.
type ForwardHTTPHeaders interface {
	// SetHTTPHeader sets the header entries associated with key to the
	// single element value. It replaces any existing values
	// associated with key. The key is case-insensitive; it is
	// canonicalized by textproto.CanonicalMIMEHeaderKey.
	SetHTTPHeader(key, value string)

	// DeleteHTTPHeader deletes the values associated with key.
	// The key is case-insensitive; it is canonicalized by
	// CanonicalHeaderKey.
	DeleteHTTPHeader(key string)

	// GetHTTPHeader gets the first value associated with the given key. If
	// there are no values associated with the key, Get returns "".
	// It is case-insensitive; textproto.CanonicalMIMEHeaderKey is
	// used to canonicalize the provided key. Get assumes that all
	// keys are stored in canonical form.
	GetHTTPHeader(key string) string

	// GetHTTPHeaders returns HTTP headers.
	GetHTTPHeaders() http.Header
}

func setHTTPHeaderInStringMap(headers map[string]string, key string, value string) {
	if headers == nil {
		headers = map[string]string{}
	}

	headers[fmt.Sprintf("%s%s", httpHeaderPrefix, key)] = value
}

func getHTTPHeadersFromStringMap(headers map[string]string) http.Header {
	httpHeaders := http.Header{}

	for k, v := range headers {
		if textproto.CanonicalMIMEHeaderKey(k) == OAuthIdentityTokenHeaderName {
			httpHeaders.Set(k, v)
		}

		if textproto.CanonicalMIMEHeaderKey(k) == OAuthIdentityIDTokenHeaderName {
			httpHeaders.Set(k, v)
		}

		if textproto.CanonicalMIMEHeaderKey(k) == CookiesHeaderName {
			httpHeaders.Set(k, v)
		}

		if after, ok := strings.CutPrefix(k, httpHeaderPrefix); ok {
			hKey := after
			httpHeaders.Set(hKey, v)
		}
	}

	return httpHeaders
}

func deleteHTTPHeaderInStringMap(headers map[string]string, key string) {
	for k := range headers {
		if textproto.CanonicalMIMEHeaderKey(k) == textproto.CanonicalMIMEHeaderKey(key) ||
			textproto.CanonicalMIMEHeaderKey(k) == textproto.CanonicalMIMEHeaderKey(fmt.Sprintf("%s%s", httpHeaderPrefix, key)) {
			delete(headers, k)
			break
		}
	}
}

// newHeaderMiddleware creates a new handler middleware that forwards HTTP headers to outgoing
// HTTP request sent using the HTTP client from the httpclient package.
func newHeaderMiddleware() HandlerMiddleware {
	return HandlerMiddlewareFunc(func(next Handler) Handler {
		return &headerMiddleware{
			BaseHandler: NewBaseHandler(next),
		}
	})
}

// headerMiddleware a handler middleware that forwards HTTP headers to outgoing
// HTTP request sent using the HTTP client from the httpclient package.
type headerMiddleware struct {
	BaseHandler
}

func (m headerMiddleware) applyHeaders(ctx context.Context, headers http.Header) context.Context {
	if len(headers) > 0 {
		if GrafanaConfigFromContext(ctx).FeatureToggles().IsEnabled(featuretoggles.PluginsFilterForwardedHeaders) {
			// Only filter the forwarding copy; handlers retain their incoming headers.
			headers = headers.Clone()
			for _, header := range headersExcludedFromForwarding {
				headers.Del(header)
			}
		}

		ctx = httpclient.WithContextualMiddleware(ctx,
			httpclient.MiddlewareFunc(func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
				if !opts.ForwardHTTPHeaders {
					return next
				}

				return httpclient.RoundTripperFunc(func(qreq *http.Request) (*http.Response, error) {
					// Only set a header if it is not already set.
					for k, v := range headers {
						if qreq.Header.Get(k) == "" {
							for _, vv := range v {
								qreq.Header.Add(k, vv)
							}
						}
					}
					return next.RoundTrip(qreq)
				})
			}))
	}
	return ctx
}

func (m *headerMiddleware) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	if req == nil {
		return m.BaseHandler.QueryData(ctx, req)
	}

	ctx = m.applyHeaders(ctx, req.GetHTTPHeaders())
	return m.BaseHandler.QueryData(ctx, req)
}

func (m *headerMiddleware) CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
	if req == nil {
		return m.BaseHandler.CallResource(ctx, req, sender)
	}

	ctx = m.applyHeaders(ctx, req.GetHTTPHeaders())
	return m.BaseHandler.CallResource(ctx, req, sender)
}

func (m *headerMiddleware) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	if req == nil {
		return m.BaseHandler.CheckHealth(ctx, req)
	}

	ctx = m.applyHeaders(ctx, req.GetHTTPHeaders())
	return m.BaseHandler.CheckHealth(ctx, req)
}
