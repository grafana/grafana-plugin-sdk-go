package marketplace

import (
	"errors"
	"flag"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

// Opts are the options for Licensing.
type Opts struct {
	// ParentPluginID is the ID of the parent plugin, for nested plugins.
	// When set, the license is looked up and validated against this ID
	// instead of the plugin's own ID. When empty, the plugin is assumed
	// not to be nested and its own ID is used.
	ParentPluginID string
}

// licensePluginID returns the plugin ID the license is issued under.
func (o Opts) licensePluginID(pluginID string) string {
	if o.ParentPluginID != "" {
		return o.ParentPluginID
	}
	return pluginID
}

// queryTimeLicenseValidation is set by Grafana with -qtlv. It is registered at package
// level because the flags are parsed before Licensing runs.
var queryTimeLicenseValidation = flag.Bool("qtlv", false,
	"Validate the marketplace license on every request, using the license sent by Grafana in the gRPC metadata, instead of once at startup")

// Licensing returns a decorator that enforces marketplace licensing. Set it as the
// Licensing field of datasource.ManageOpts or app.ManageOpts:
//
//	datasource.Manage("my-datasource", newDatasource, datasource.ManageOpts{
//		Licensing: marketplace.Licensing(marketplace.Opts{}),
//	})
//
// The license is looked up for the plugin's own ID, or for opts.ParentPluginID if set.
//
// By default the license is checked once, when the plugin starts, from the environment
// variables or license files. With a valid license, or an expired one still within its
// grace period, the plugin is served unchanged. Otherwise the plugin is served with
// handlers that only return the license error.
//
// With the -qtlv flag the license is instead validated on every request, from the
// "license" gRPC metadata that Grafana sends with it. Environment variables and license
// files are ignored, and a request without valid license metadata is refused like above:
// queries and conversions fail with the license error, and admission requests are denied
// with a 403 result carrying it. In this mode:
//   - ServeOpts.ExtraPlugins are not supported and make the decorator return an error,
//     since they bypass the handler middlewares.
//   - CollectMetrics is not checked.
//   - Each SubscribeStream and PublishStream call is checked, but RunStream is only checked
//     when the stream starts, not while it runs.
//
// Builds with the marketplace_dev tag skip both kinds of checks.
func Licensing(opts Opts) backend.ServeOptsDecorator {
	return func(pluginID string, serveOpts backend.ServeOpts) (backend.ServeOpts, error) {
		licensePluginID := opts.licensePluginID(pluginID)
		if *queryTimeLicenseValidation {
			return applyRequestTimeLicense(pluginID, licensePluginID, serveOpts)
		}
		return checkMarketplacePluginLicense(pluginID, licensePluginID, serveOpts), nil
	}
}

// applyRequestTimeLicense makes serveOpts validate the license of every request.
func applyRequestTimeLicense(pluginID, licensePluginID string, serveOpts backend.ServeOpts) (backend.ServeOpts, error) {
	return applyRequestTimeLicenseWithDeps(pluginID, licensePluginID, serveOpts, requestLicenseLoader(licensing.LoadTokenFromValue))
}

func applyRequestTimeLicenseWithDeps(
	pluginID, licensePluginID string,
	serveOpts backend.ServeOpts,
	loadToken licenseTokenLoader,
) (backend.ServeOpts, error) {
	if developmentLicenseBypass {
		backend.Logger.Warn(
			"Marketplace license checks are disabled in this development build. Do not distribute this executable.",
			"pluginId", pluginID,
		)
		return serveOpts, nil
	}
	if len(serveOpts.ExtraPlugins) > 0 {
		return backend.ServeOpts{}, errors.New("marketplace: ExtraPlugins are not supported with query-time license validation (-qtlv)")
	}

	middleware := newLicenseMiddleware(pluginID, licensePluginID, loadToken)
	// Copy instead of prepending in place, so the caller's slice is not modified.
	serveOpts.HandlerMiddlewares = append([]backend.HandlerMiddleware{middleware}, serveOpts.HandlerMiddlewares...)
	if serveOpts.QueryConversionHandler != nil {
		serveOpts.QueryConversionHandler = &licensedQueryConversionHandler{
			next: serveOpts.QueryConversionHandler,
			middleware: &licenseMiddleware{
				pluginID:        pluginID,
				licensePluginID: licensePluginID,
				loadToken:       loadToken,
			},
		}
	}
	return serveOpts, nil
}
