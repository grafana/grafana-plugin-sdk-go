package marketplace

import (
	"flag"

	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
)

var queryTimeLicenseValidation = flag.Bool("qtlv", false, "should validate marketplace license on every request")

// Manage serves a marketplace data source with automatic instance management.
// It blocks until the plugin is terminated.
// By default, it validates the license once at startup. With -qtlv, it validates
// the license supplied in incoming gRPC metadata before each request instead.
func Manage(pluginID string, instanceFactory datasource.InstanceFactoryFunc, opts datasource.ManageOpts) error {
	flag.Parse()

	if *queryTimeLicenseValidation {
		return datasource.Manage(pluginID, marketplaceInstanceFactory(instanceFactory), opts)
	}

	if err := CheckMarketplacePluginLicense(pluginID); err != nil {
		return err
	}
	return datasource.Manage(pluginID, instanceFactory, opts)
}
