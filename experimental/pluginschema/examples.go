package pluginschema

import (
	"k8s.io/kube-openapi/pkg/spec3"

	dsV0 "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
)

type QueryTypeExample struct {
	// The query type key -- this must match the types name
	QueryType string `json:"queryType"`

	// The example
	Example dsV0.QueryExample `json:"example"`
}

type QueryExamples struct {
	Examples []QueryTypeExample `json:"examples"`
}

type SettingsExamples struct {
	// Example configuration added to the swagger documentation
	Examples map[string]*spec3.Example `json:"examples"`
}
