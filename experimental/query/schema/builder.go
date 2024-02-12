package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/query"
	"github.com/invopop/jsonschema"
)

type QueryTypeInfo struct {
	QueryType string
	Version   string
	GoType    reflect.Type
}

type QueryTypeBuilder struct {
	opts      BuilderOptions
	reflector *jsonschema.Reflector // Needed to use comments
	byType    map[string]*query.QueryTypeDefinition
	types     []*query.QueryTypeDefinition
}

func (b *QueryTypeBuilder) Add(info QueryTypeInfo) error {
	schema := b.reflector.ReflectFromType(info.GoType)
	if schema == nil {
		return fmt.Errorf("missing schema")
	}
	def, ok := b.byType[info.QueryType]
	if !ok {
		def = &query.QueryTypeDefinition{
			Name:     info.QueryType,
			Versions: []query.QueryTypeVersion{},
		}
		b.byType[info.QueryType] = def
		b.types = append(b.types, def)
	}
	def.Versions = append(def.Versions, query.QueryTypeVersion{
		Version: info.Version,
		Schema:  schema,
	})
	return nil
}

type BuilderOptions struct {
	// ex "github.com/invopop/jsonschema"
	BasePackage string

	// ex "./"
	CodePath string

	// queryType
	DiscriminatorField string

	// org-xyz-datasource
	PluginIDs []string
}

func NewBuilder(opts BuilderOptions, inputs ...QueryTypeInfo) (*QueryTypeBuilder, error) {
	r := new(jsonschema.Reflector)
	if err := r.AddGoComments(opts.BasePackage, opts.CodePath); err != nil {
		return nil, err
	}
	b := &QueryTypeBuilder{
		opts:      opts,
		reflector: r,
		byType:    make(map[string]*query.QueryTypeDefinition),
	}
	for _, input := range inputs {
		err := b.Add(input)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (b *QueryTypeBuilder) GetFullQuerySchema() (*jsonschema.Schema, error) {
	discriminator := b.opts.DiscriminatorField
	if discriminator == "" {
		discriminator = "queryType"
	}

	query, err := asJSONSchema(query.GetCommonJSONSchema())
	if err != nil {
		return nil, err
	}
	query.Ref = ""
	common, ok := query.Definitions["CommonQueryProperties"]
	if !ok {
		return nil, fmt.Errorf("error finding common properties")
	}
	delete(query.Definitions, "CommonQueryProperties")

	for _, t := range b.types {
		for _, v := range t.Versions {
			s, err := asJSONSchema(v.Schema)
			if err != nil {
				return nil, err
			}
			if s.Ref == "" {
				return nil, fmt.Errorf("only ref elements supported right now")
			}

			ref := strings.TrimPrefix(s.Ref, "#/$defs/")
			body := s

			// Add all types to the
			for key, def := range s.Definitions {
				if key == ref {
					body = def
				} else {
					query.Definitions[key] = def
				}
			}

			if body.Properties == nil {
				return nil, fmt.Errorf("expected properties on body")
			}

			for pair := common.Properties.Oldest(); pair != nil; pair = pair.Next() {
				body.Properties.Set(pair.Key, pair.Value)
			}
			body.Required = append(body.Required, "refId")

			if t.Name != "" {
				key := t.Name
				if v.Version != "" {
					key += "/" + v.Version
				}

				p, err := body.Properties.GetAndMoveToFront(discriminator)
				if err != nil {
					return nil, fmt.Errorf("missing discriminator field: %s", discriminator)
				}
				p.Const = key
				p.Enum = nil

				body.Required = append(body.Required, discriminator)
			}

			query.OneOf = append(query.OneOf, body)
		}
	}

	return query, nil
}

// Always creates a copy so we can modify it
func asJSONSchema(v any) (*jsonschema.Schema, error) {
	var err error
	s := &jsonschema.Schema{}
	b, ok := v.([]byte)
	if !ok {
		b, err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
	}
	err = json.Unmarshal(b, s)
	return s, err
}
