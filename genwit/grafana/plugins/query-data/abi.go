// Code generated by wit-bindgen-go. DO NOT EDIT.

package querydata

import (
	"github.com/grafana/grafana-plugin-sdk-go/genwit/grafana/plugins/types"
)

func lift_PluginContext(f0 uint64) (v types.PluginContext) {
	v.OrgID = (int64)(f0)
	return
}

func lift_QueryDataRequest(f0 uint64) (v types.QueryDataRequest) {
	v.PluginContext = lift_PluginContext(f0)
	return
}
