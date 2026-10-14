package marketplace

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
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

// Licensing returns a decorator that enforces marketplace licensing. Set it as the
// Licensing field of datasource.ManageOpts or app.ManageOpts:
//
//	datasource.Manage("my-datasource", newDatasource, datasource.ManageOpts{
//		Licensing: marketplace.Licensing(marketplace.Opts{}),
//	})
//
// The license is checked once, when the plugin starts. With a valid license, or an
// expired one still within its grace period, the plugin is served unchanged. Otherwise
// the plugin is served with handlers that only return the license error.
//
// The license is looked up for the plugin's own ID, or for opts.ParentPluginID if set.
func Licensing(opts Opts) backend.ServeOptsDecorator {
	return func(pluginID string, serveOpts backend.ServeOpts) (backend.ServeOpts, error) {
		// TODO: PPT support (-qtlv): validate the license on every request. This will be implemented in a follow-up PR.
		return checkMarketplacePluginLicense(pluginID, opts.licensePluginID(pluginID), serveOpts), nil
	}
}
