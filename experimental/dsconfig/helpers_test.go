package dsconfig_test

import (
	"github.com/grafana/grafana-plugin-sdk-go/experimental/dsconfig"
)

func ptr[T any](v T) *T { return &v }

func validStorageField(id, key string) dsconfig.ConfigField {
	return dsconfig.ConfigField{
		ID:        id,
		Key:       key,
		ValueType: dsconfig.StringType,
		Target:    dsconfig.JSONDataTarget,
	}
}

func minimalSchema(fields ...dsconfig.ConfigField) *dsconfig.DatasourceConfigSchema {
	if len(fields) == 0 {
		fields = append(fields, dsconfig.ConfigField{
			ID:        "url",
			Key:       "url",
			ValueType: dsconfig.StringType,
			Target:    dsconfig.RootTarget,
		})
	}
	return &dsconfig.DatasourceConfigSchema{
		SchemaVersion: "v1",
		PluginType:    "test",
		PluginName:    "Test",
		Fields:        fields,
	}
}
