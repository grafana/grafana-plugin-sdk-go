package marketplace

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
)

// ManageOpts are the options for Manage.
type ManageOpts struct {
	// ParentPluginID is the ID of the parent plugin, for nested plugins.
	// When set, the license is looked up and validated against this ID
	// instead of the plugin's own ID. When empty, the plugin is assumed
	// not to be nested and its own ID is used.
	ParentPluginID string

	// DatasourceOpts are passed through to datasource.Manage.
	DatasourceOpts datasource.ManageOpts
}

// licensePluginID returns the plugin ID the license is issued under.
func (o ManageOpts) licensePluginID(pluginID string) string {
	if o.ParentPluginID != "" {
		return o.ParentPluginID
	}
	return pluginID
}

// Manage checks the marketplace license and then serves the datasource plugin.
//
// pluginID is the plugin's own ID, as in its plugin.json, and is used to serve
// the plugin. opts.ParentPluginID, if set, is the ID of the parent plugin of a
// nested plugin; it is used instead of pluginID for license lookup and validation.
func Manage(pluginID string, instanceFactory datasource.InstanceFactoryFunc, opts ManageOpts) error {
	// TODO: PPT support: this will be implemented in a follow-up PR
	/* flag.Parse() // Parse the flags so that we can check the value of -qtlv

	// If -qtlv is set, then we should check the license on every request
	if *queryTimeLicenseValidation {
		return datasource.Manage(pluginID, enterpriseInstanceFactory(instanceFactory), opts.DatasourceOpts)
	} */

	// If -qtlv is not set, then we should check the license once at startup
	if err := checkMarketplacePluginLicense(pluginID, opts.licensePluginID(pluginID)); err != nil {
		return err
	}
	return datasource.Manage(pluginID, instanceFactory, opts.DatasourceOpts)
}
