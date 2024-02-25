package schemabuilder

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/resource"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// The k8s compatible jsonschema version
const draft04 = "https://json-schema.org/draft-04/schema"

// Supported expression types
// +enum
type SchemaType string

const (
	// Single query target saved in a dashboard/panel/alert JSON
	SchemaTypeSaveModel SchemaType = "save"

	// Single query payload included in a query request
	SchemaTypeQueryPayload SchemaType = "payload"

	// Pseudo panel model including multiple targets (not mixed)
	SchemaTypePanelModel SchemaType = "panel"

	// Query request against a single data source (not mixed)
	SchemaTypeQueryRequest SchemaType = "request"
)

type QuerySchemaOptions struct {
	PluginID   []string
	QueryTypes []resource.QueryTypeDefinition
	Mode       SchemaType
}

// Given definitions for a plugin, return a valid spec
func GetQuerySchema(opts QuerySchemaOptions) (*spec.Schema, error) {
	isRequest := opts.Mode == SchemaTypeQueryPayload || opts.Mode == SchemaTypeQueryRequest
	generic, err := resource.CommonQueryPropertiesSchema()
	if err != nil {
		return nil, err
	}

	ignoreForSave := map[string]bool{"maxDataPoints": true, "intervalMs": true}
	common := make(map[string]spec.Schema)
	for key, val := range generic.Properties {
		if !isRequest && ignoreForSave[key] {
			continue //
		}
		common[key] = val
	}

	// The datasource requirement
	switch len(opts.PluginID) {
	case 0:
	case 1:
		s := common["datasource"].Properties["type"]
		s.Pattern = "xxxx"
	default:
		if opts.Mode == SchemaTypePanelModel {
			return nil, fmt.Errorf("panel model requires pluginId")
		}
		s := common["datasource"].Properties["type"]
		s.Pattern = "yyyyy"
	}

	// The types for each query type
	queryTypes := []*spec.Schema{}
	for _, qt := range opts.QueryTypes {
		node, err := asJSONSchema(qt.Spec.QuerySchema)
		if err != nil {
			return nil, fmt.Errorf("error reading query types schema: %s // %w", qt.ObjectMeta.Name, err)
		}
		if node == nil {
			return nil, fmt.Errorf("missing query schema: %s // %v", qt.ObjectMeta.Name, qt)
		}

		// Match all discriminators
		for _, d := range qt.Spec.Discriminators {
			ds, ok := node.Properties[d.Field]
			if !ok {
				ds = *spec.StringProperty()
			}
			ds.Pattern = `^` + d.Value + `$`
			node.Properties[d.Field] = ds
			node.Required = append(node.Required, d.Field)
		}

		queryTypes = append(queryTypes, node)
	}

	s := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:       []string{"object"},
			Schema:     draft04,
			Properties: make(map[string]spec.Schema),
		},
	}

	// Single node -- just union the global and local properties
	if len(queryTypes) == 1 {
		s = queryTypes[0]
		s.Schema = draft04
		for key, val := range generic.Properties {
			_, found := s.Properties[key]
			if found {
				continue
			}
			s.Properties[key] = val
		}
	} else {
		for _, qt := range queryTypes {
			qt.Required = append(qt.Required, "refId")

			for k, v := range common {
				_, found := qt.Properties[k]
				if found {
					continue
				}
				qt.Properties[k] = v
			}

			s.OneOf = append(s.OneOf, *qt)
		}
	}

	if isRequest {
		s = addRequestWrapper(s)
	}
	return s, nil
}

// moves the schema the the query slot in a request
func addRequestWrapper(s *spec.Schema) *spec.Schema {
	return &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:                 []string{"object"},
			Required:             []string{"queries"},
			AdditionalProperties: &spec.SchemaOrBool{Allows: false},
			Properties: map[string]spec.Schema{
				"from": *spec.StringProperty().WithDescription(
					"From Start time in epoch timestamps in milliseconds or relative using Grafana time units."),
				"to": *spec.StringProperty().WithDescription(
					"To end time in epoch timestamps in milliseconds or relative using Grafana time units."),
				"queries": *spec.ArrayProperty(s),
				"debug":   *spec.BoolProperty(),
				"$schema": *spec.StringProperty().WithDescription("helper"),
			},
		},
	}
}

func asJSONSchema(v any) (*spec.Schema, error) {
	s, ok := v.(*spec.Schema)
	if ok {
		return s, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	s = &spec.Schema{}
	err = json.Unmarshal(b, s)
	return s, err
}

func asGenericDataQuery(v any) (*resource.GenericDataQuery, error) {
	s, ok := v.(*resource.GenericDataQuery)
	if ok {
		return s, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	s = &resource.GenericDataQuery{}
	err = json.Unmarshal(b, s)
	return s, err
}
