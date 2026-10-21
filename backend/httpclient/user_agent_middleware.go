package httpclient

import (
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/useragent"
	"github.com/grafana/grafana-plugin-sdk-go/build/buildinfo"
	"github.com/grafana/grafana-plugin-sdk-go/config"
)

// UserAgentMiddlewareName is the middleware name used by UserAgentMiddleware.
const UserAgentMiddlewareName = "UserAgent"

// UserAgentMiddleware sets a user agent header on the outgoing request.
//
// If Grafana sends a configured user agent ([plugins] user_agent), it is used as the
// prefix in place of the Grafana user agent from the request context.
func UserAgentMiddleware() Middleware {
	info, err := buildinfo.GetBuildInfo.GetInfo()

	if err != nil {
		log.DefaultLogger.Debug("failed to get plugin build info, HTTP requests will only have a user agent set if one is configured", "error", err)

		return newUserAgentMiddleware("", "", false)
	}

	return newUserAgentMiddleware(info.PluginID, info.Version, true)
}

func newUserAgentMiddleware(pluginID string, version string, haveVersionInfo bool) Middleware {
	userAgentSuffix := fmt.Sprintf(" %s/%s", pluginID, version)

	return NamedMiddlewareFunc(UserAgentMiddlewareName, func(opts Options, next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if len(req.Header.Values("User-Agent")) > 0 {
				return next.RoundTrip(req)
			}

			configured := configuredUserAgent(req)

			switch {
			case configured != "" && haveVersionInfo:
				req.Header.Set("User-Agent", configured+userAgentSuffix)
			case configured != "":
				req.Header.Set("User-Agent", configured)
			case !haveVersionInfo:
				// No build info and nothing configured: leave the header unset.
			default:
				baseUserAgent := useragent.FromContext(req.Context())
				if baseUserAgent.IsUnknown() {
					req.Header.Set("User-Agent", "Grafana"+userAgentSuffix)
				} else {
					req.Header.Set("User-Agent", baseUserAgent.String()+userAgentSuffix)
				}
			}

			return next.RoundTrip(req)
		})
	})
}

// configuredUserAgent returns the user agent configured in Grafana for this request,
// or an empty string if none is set. Values with control characters are ignored,
// since net/http rejects them as header values and the request would fail.
func configuredUserAgent(req *http.Request) string {
	ua := strings.TrimSpace(config.GrafanaConfigFromContext(req.Context()).PluginsUserAgent())
	if strings.ContainsFunc(ua, unicode.IsControl) {
		log.DefaultLogger.Debug("ignoring configured plugin user agent with control characters")
		return ""
	}
	return ua
}
