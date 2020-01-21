package adapter

import "github.com/grafana/grafana-plugin-sdk-go/backend"

// SDKAdapter adapter between low level and SDK interfaces.
type SDKAdapter struct {
	SchemaProvider       backend.SchemaProviderFunc
	CheckHealthHandler   backend.CheckHealthHandler
	DataQueryHandler     backend.DataQueryHandler
	TransformDataHandler backend.TransformDataHandler
	schema               backend.Schema
}
