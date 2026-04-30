package marketplace

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
)

func Manage(pluginID string, instanceFactory datasource.InstanceFactoryFunc, opts datasource.ManageOpts) error {
	// TODO: implement
	/* flag.Parse() // Parse the flags so that we can check the value of -qtlv

	// If -qtlv is set, then we should check the license on every request
	if *queryTimeLicenseValidation {
		return datasource.Manage(pluginID, enterpriseInstanceFactory(instanceFactory), opts)
	} */

	// If -qtlv is not set, then we should check the license once at startup
	if err := CheckMarketplacePluginLicense(pluginID); err != nil {
		return err
	}
	return datasource.Manage(pluginID, instanceFactory, opts)
}
