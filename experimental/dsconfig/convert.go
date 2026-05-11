package dsconfig

import (
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// ToPluginSettings converts the schema to a pluginschema.Settings object.
//
// The Spec schema includes both root-level fields (url, basicAuth, etc.)
// and jsonData fields as properties. Fields targeting secureJsonData
// become SecureValues entries. Virtual fields are skipped.
func (s *DatasourceConfigSchema) ToPluginSettings() (*pluginschema.Settings, error) {
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	rootProps := make(map[string]spec.Schema)
	var rootRequired []string

	jsonDataProps := make(map[string]spec.Schema)
	var jsonDataRequired []string

	var secureValues []pluginschema.SecureValueInfo

	for _, f := range s.Fields {
		if f.Kind == VirtualField {
			continue
		}

		if f.Target == SecureJSONTarget {
			secureValues = append(secureValues, pluginschema.SecureValueInfo{
				Key:         f.Key,
				Description: f.Description,
				Required:    f.Required,
			})
			continue
		}

		if f.Target == JSONDataTarget {
			placeInSection(jsonDataProps, f)
			if f.Required && f.Section == "" {
				jsonDataRequired = append(jsonDataRequired, f.Key)
			}
		} else {
			placeInSection(rootProps, f)
			if f.Required && f.Section == "" {
				rootRequired = append(rootRequired, f.Key)
			}
		}
	}

	if len(jsonDataProps) > 0 {
		jd := spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type:       spec.StringOrArray{"object"},
				Properties: jsonDataProps,
			},
		}
		if len(jsonDataRequired) > 0 {
			jd.Required = jsonDataRequired
		}
		rootProps["jsonData"] = jd
	}

	specSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:       spec.StringOrArray{"object"},
			Properties: rootProps,
		},
	}
	if len(rootRequired) > 0 {
		specSchema.Required = rootRequired
	}

	return &pluginschema.Settings{
		Spec:         specSchema,
		SecureValues: secureValues,
	}, nil
}

// placeInSection places a field into the correct section sub-object within props.
func placeInSection(props map[string]spec.Schema, f ConfigField) {
	if f.Section == "" {
		props[f.Key] = fieldToSpecSchema(f)
		return
	}
	placeInSectionPath(props, strings.Split(f.Section, "."), f)
}

// placeInSectionPath recursively walks the section path segments,
// creating intermediate object schemas as needed, then places the
// field at the final level.
func placeInSectionPath(props map[string]spec.Schema, segments []string, f ConfigField) {
	seg := segments[0]

	section, exists := props[seg]
	if !exists {
		section = spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type:       spec.StringOrArray{"object"},
				Properties: make(map[string]spec.Schema),
			},
		}
	}
	if section.Properties == nil {
		section.Properties = make(map[string]spec.Schema)
	}

	if len(segments) == 1 {
		section.Properties[f.Key] = fieldToSpecSchema(f)
		if f.Required {
			section.Required = append(section.Required, f.Key)
		}
	} else {
		placeInSectionPath(section.Properties, segments[1:], f)
	}

	props[seg] = section
}

// fieldToSpecSchema converts a ConfigField to an OpenAPI spec.Schema.
func fieldToSpecSchema(f ConfigField) spec.Schema {
	s := spec.Schema{
		SchemaProps: spec.SchemaProps{
			Description: f.Description,
			Type:        spec.StringOrArray{valueTypeToJSONType(f.ValueType)},
		},
	}

	if f.DefaultValue != nil {
		s.Default = f.DefaultValue
	}

	if f.SemanticType != "" {
		if fmt := semanticTypeToFormat(f.SemanticType); fmt != "" {
			s.Format = fmt
		}
	}

	applyValidations(&s, f)

	if f.ValueType == ArrayType && f.Item != nil {
		itemSchema := itemSchemaToSpec(*f.Item)
		s.Items = &spec.SchemaOrArray{Schema: &itemSchema}
	}

	if f.ValueType == ObjectType && f.Item != nil && len(f.Item.Fields) > 0 {
		props := make(map[string]spec.Schema)
		var required []string
		for _, sub := range f.Item.Fields {
			props[sub.Key] = fieldToSpecSchema(sub)
			if sub.Required {
				required = append(required, sub.Key)
			}
		}
		s.Properties = props
		if len(required) > 0 {
			s.Required = required
		}
	}

	return s
}

// applyValidations maps dsconfig validation rules to JSON Schema keywords.
func applyValidations(s *spec.Schema, f ConfigField) {
	for _, v := range f.Validations {
		switch v.Type {
		case PatternValidation:
			s.Pattern = v.Pattern
		case RangeValidation:
			s.Minimum = v.Min
			s.Maximum = v.Max
		case LengthValidation:
			if v.Min != nil {
				n := int64(*v.Min)
				s.MinLength = &n
			}
			if v.Max != nil {
				n := int64(*v.Max)
				s.MaxLength = &n
			}
		case ItemCountValidation:
			if v.Min != nil {
				n := int64(*v.Min)
				s.MinItems = &n
			}
			if v.Max != nil {
				n := int64(*v.Max)
				s.MaxItems = &n
			}
		case AllowedValuesValidation:
			s.Enum = make([]any, len(v.Values))
			copy(s.Enum, v.Values)
		}
	}
}

func itemSchemaToSpec(item FieldItemSchema) spec.Schema {
	s := spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: spec.StringOrArray{valueTypeToJSONType(item.ValueType)},
		},
	}

	if item.ValueType == ObjectType && len(item.Fields) > 0 {
		props := make(map[string]spec.Schema)
		var required []string
		for _, f := range item.Fields {
			props[f.Key] = fieldToSpecSchema(f)
			if f.Required {
				required = append(required, f.Key)
			}
		}
		s.Properties = props
		if len(required) > 0 {
			s.Required = required
		}
	}

	return s
}

func valueTypeToJSONType(vt ValueType) string {
	switch vt {
	case StringType:
		return "string"
	case NumberType:
		return "number"
	case BooleanType:
		return "boolean"
	case ArrayType:
		return "array"
	case ObjectType, MapType:
		return "object"
	default:
		return "string"
	}
}

func semanticTypeToFormat(st SemanticType) string {
	switch st {
	case URLType:
		return "uri"
	case PasswordType:
		return "password"
	case HostnameType:
		return "hostname"
	case DurationType:
		return "duration"
	default:
		return ""
	}
}
