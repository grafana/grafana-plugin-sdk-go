package example

import (
	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/spec"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
)

func newSchema() *pluginschema.PluginSchema {
	schema := pluginschema.PluginSchema{
		TargetAPIVersion: "v0alpha1",
		SettingsSchema: &pluginschema.Settings{
			Spec: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Description: "Test data does not require any explicit configuration",
				},
			},

			SecureValues: []pluginschema.SecureValueInfo{
				{
					Key:         "aaa",
					Description: "describe aaa",
					Required:    true,
				}, {
					Key:         "bbb",
					Description: "describe bbb",
				},
			},
		},

		SettingsExamples: &pluginschema.SettingsExamples{
			Examples: map[string]*spec3.Example{
				"": {
					ExampleProps: spec3.ExampleProps{
						Description: "a sample",
						Value:       "invalid",
					},
				},
			},
		},

		Routes: &pluginschema.Routes{},
	}
	p := schema.SettingsSchema.Spec
	p.Required = []string{"title"}
	p.AdditionalProperties = &spec.SchemaOrBool{Allows: false}
	p.Properties = map[string]spec.Schema{
		"title": *spec.StringProperty().WithDescription("display name"),
		"url":   *spec.StringProperty().WithDescription("not used"),
	}
	p.Example = map[string]any{
		"url": "http://xxxx",
	}

	schema.Routes.Register("/hello", spec3.PathProps{
		Description: "world",
	})
	schema.Routes.Register("/routes", spec3.PathProps{
		Description: "more",
	})

	return &schema
}
