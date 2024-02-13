package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/query"
	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type QueryTypeInfo struct {
	QueryType string
	Version   string
	GoType    reflect.Type
}

type QueryTypeBuilder struct {
	t         *testing.T
	opts      BuilderOptions
	reflector *jsonschema.Reflector // Needed to use comments
	byType    map[string]*query.QueryTypeDefinitionSpec
	types     []*query.QueryTypeDefinitionSpec
}

func (b *QueryTypeBuilder) Add(info QueryTypeInfo) error {
	schema := b.reflector.ReflectFromType(info.GoType)
	if schema == nil {
		return fmt.Errorf("missing schema")
	}

	// Ignored by k8s anyway
	schema.Version = ""
	schema.ID = ""
	schema.Anchor = ""

	def, ok := b.byType[info.QueryType]
	if !ok {
		def = &query.QueryTypeDefinitionSpec{
			Name:               info.QueryType,
			DiscriminatorField: b.opts.DiscriminatorField,
			Versions:           []query.QueryTypeVersion{},
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
}

func NewBuilder(t *testing.T, opts BuilderOptions, inputs ...QueryTypeInfo) (*QueryTypeBuilder, error) {
	r := new(jsonschema.Reflector)
	r.DoNotReference = true
	if err := r.AddGoComments(opts.BasePackage, opts.CodePath); err != nil {
		return nil, err
	}
	b := &QueryTypeBuilder{
		t:         t,
		opts:      opts,
		reflector: r,
		byType:    make(map[string]*query.QueryTypeDefinitionSpec),
	}
	for _, input := range inputs {
		err := b.Add(input)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (b *QueryTypeBuilder) Write(outfile string) json.RawMessage {
	t := b.t
	t.Helper()

	now := time.Now().UTC()
	rv := fmt.Sprintf("%d", now.UnixMilli())

	defs := query.QueryTypeDefinitionList{}
	byName := make(map[string]*query.QueryTypeDefinition)
	body, err := os.ReadFile(outfile)
	if err == nil {
		err = json.Unmarshal(body, &defs)
		if err == nil {
			for i, def := range defs.Items {
				byName[def.ObjectMeta.Name] = &defs.Items[i]
			}
		}
	}

	// The updated schemas
	for _, spec := range b.types {
		found, ok := byName[spec.Name]
		if !ok {
			defs.ObjectMeta.ResourceVersion = rv
			defs.Items = append(defs.Items, query.QueryTypeDefinition{
				ObjectMeta: query.ObjectMeta{
					Name:              spec.Name,
					ResourceVersion:   rv,
					CreationTimestamp: now.Format(time.RFC3339),
				},
				Spec: *spec,
			})
		} else {
			var o1, o2 interface{}
			b1, _ := json.Marshal(spec)
			b2, _ := json.Marshal(found.Spec)
			_ = json.Unmarshal(b1, &o1)
			_ = json.Unmarshal(b2, &o2)
			if !reflect.DeepEqual(o1, o2) {
				found.ObjectMeta.ResourceVersion = rv
				found.Spec = *spec
			}
			delete(byName, spec.Name)
		}
	}

	if defs.ObjectMeta.ResourceVersion == "" {
		defs.ObjectMeta.ResourceVersion = rv
	}

	if len(byName) > 0 {
		require.FailNow(t, "query type removed, manually update (for now)")
	}

	out, err := json.MarshalIndent(defs, "", "  ")
	require.NoError(t, err)

	update := false
	if err == nil {
		if !assert.JSONEq(t, string(out), string(body)) {
			update = true
		}
	} else {
		update = true
	}
	if update {
		err = os.WriteFile(outfile, out, 0644)
		require.NoError(t, err, "error writing file")
	}
	return out
}

func (b *QueryTypeBuilder) FullQuerySchema() (*jsonschema.Schema, error) {
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
