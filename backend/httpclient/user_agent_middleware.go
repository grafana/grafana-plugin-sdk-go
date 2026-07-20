package httpclient

import (
	"fmt"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/useragent"
	"github.com/grafana/grafana-plugin-sdk-go/build/buildinfo"
)

// UserAgentMiddlewareName is the middleware name used by UserAgentMiddleware.
const UserAgentMiddlewareName = "UserAgent"

// UserAgentMiddleware sets a user agent header on the outgoing request.
func UserAgentMiddleware() Middleware {
	info, err := buildinfo.GetBuildInfo.GetInfo()

	if err != nil {
		log.DefaultLogger.Debug("failed to get plugin build info, HTTP requests will not have a user agent set", "error", err)

		return newUserAgentMiddleware("", "", false)
	}

	return newUserAgentMiddleware(info.PluginID, info.Version, true)
}

func newUserAgentMiddleware(pluginID string, version string, haveVersionInfo bool) Middleware {
	userAgentSuffix := fmt.Sprintf(" %s/%s (Grafana plugin)", pluginID, version)

	return NamedMiddlewareFunc(UserAgentMiddlewareName, func(opts Options, next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if !haveVersionInfo {
				return next.RoundTrip(req)
			}

			baseUserAgent := useragent.FromContext(req.Context()).String()

			if len(req.Header.Values("User-Agent")) == 0 {
				req.Header.Set("User-Agent", baseUserAgent+userAgentSuffix)
			}

			return next.RoundTrip(req)
		})
	})
}
