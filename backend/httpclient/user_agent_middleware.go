package httpclient

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/build/buildinfo"
)

// UserAgentMiddlewareName is the middleware name used by UserAgentMiddleware.
const UserAgentMiddlewareName = "UserAgent"

// UserAgentMiddleware sets a user agent header on the outgoing request.
func UserAgentMiddleware() Middleware {
	info, err := buildinfo.GetBuildInfo.GetInfo()

	if err != nil {
		log.DefaultLogger.Error("failed to get plugin build info", "error", err)

		info = buildinfo.Info{
			PluginID: "unknown-grafana-plugin",
			Version:  "unknown",
		}
	}

	return newUserAgentMiddleware(info.PluginID, info.Version)
}

func newUserAgentMiddleware(pluginID string, version string) Middleware {
	userAgent := pluginID + "/" + version

	return NamedMiddlewareFunc(UserAgentMiddlewareName, func(opts Options, next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Header.Set("User-Agent", userAgent)

			return next.RoundTrip(req)
		})
	})
}
