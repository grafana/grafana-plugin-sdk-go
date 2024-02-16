package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SchemaBuilder is a helper function that can be used by
// backend build processes to produce static schema definitions
// This is not intended as runtime code, and is not the only way to
// produce a schema (we may also want/need to use typescript as the source)
type SchemaBuilder struct {
	opts      BuilderOptions
	reflector *jsonschema.Reflector // Needed to use comments
	query     []QueryTypeDefinition
	setting   []SettingsDefinition
}

type BuilderOptions struct {
	// ex "github.com/invopop/jsonschema"
	BasePackage string

	// ex "./"
	CodePath string

	// queryType
	DiscriminatorField string

	// explicitly define the enumeration fields
	Enums []reflect.Type
}

type QueryTypeInfo struct {
	// The management name
	Name string
	// The discriminator value (requires the field set in ops)
	Discriminator string
	// Raw GO type used for reflection
	GoType reflect.Type
	// Add sample queries
	Examples []QueryExample
}

type SettingTypeInfo struct {
	// The management name
	Name string
	// The discriminator value (requires the field set in ops)
	Discriminator string
	// Raw GO type used for reflection
	GoType reflect.Type
	// Map[string]string
	SecureGoType reflect.Type
}

func NewSchemaBuilder(opts BuilderOptions) (*SchemaBuilder, error) {
	r := new(jsonschema.Reflector)
	r.DoNotReference = true
	if err := r.AddGoComments(opts.BasePackage, opts.CodePath); err != nil {
		return nil, err
	}
	customMapper := map[reflect.Type]*jsonschema.Schema{
		reflect.TypeOf(data.Frame{}): {
			Type: "object",
			Extras: map[string]any{
				"x-grafana-type": "data.DataFrame",
			},
			AdditionalProperties: jsonschema.TrueSchema,
		},
	}
	r.Mapper = func(t reflect.Type) *jsonschema.Schema {
		return customMapper[t]
	}

	if len(opts.Enums) > 0 {
		fields, err := findEnumFields(opts.BasePackage, opts.CodePath)
		if err != nil {
			return nil, err
		}
		for _, etype := range opts.Enums {
			for _, f := range fields {
				if f.Name == etype.Name() && f.Package == etype.PkgPath() {
					enumValueDescriptions := map[string]string{}
					s := &jsonschema.Schema{
						Type: "string",
						Extras: map[string]any{
							"x-enum-description": enumValueDescriptions,
						},
					}
					for _, val := range f.Values {
						s.Enum = append(s.Enum, val.Value)
						if val.Comment != "" {
							enumValueDescriptions[val.Value] = val.Comment
						}
					}
					customMapper[etype] = s
				}
			}
		}
	}

	return &SchemaBuilder{
		opts:      opts,
		reflector: r,
	}, nil
}

func (b *SchemaBuilder) AddQueries(inputs ...QueryTypeInfo) error {
	for _, info := range inputs {
		schema := b.reflector.ReflectFromType(info.GoType)
		if schema == nil {
			return fmt.Errorf("missing schema")
		}

		b.enumify(schema)

		// used by kube-openapi
		schema.Version = "https://json-schema.org/draft-04/schema"
		schema.ID = ""
		schema.Anchor = ""

		name := info.Name
		if name == "" {
			name = info.Discriminator
			if name == "" {
				return fmt.Errorf("missing name or discriminator")
			}
		}

		if info.Discriminator != "" && b.opts.DiscriminatorField == "" {
			return fmt.Errorf("missing discriminator field")
		}

		b.query = append(b.query, QueryTypeDefinition{
			ObjectMeta: ObjectMeta{
				Name: name,
			},
			Spec: QueryTypeDefinitionSpec{
				DiscriminatorField: b.opts.DiscriminatorField,
				DiscriminatorValue: info.Discriminator,
				QuerySchema:        schema,
				Examples:           info.Examples,
			},
		})
	}
	return nil
}

func (b *SchemaBuilder) AddSettings(inputs ...SettingTypeInfo) error {
	for _, info := range inputs {
		schema := b.reflector.ReflectFromType(info.GoType)
		if schema == nil {
			return fmt.Errorf("missing schema")
		}

		b.enumify(schema)

		// used by kube-openapi
		schema.Version = "https://json-schema.org/draft-04/schema"
		schema.ID = ""
		schema.Anchor = ""

		name := info.Name
		if name == "" {
			name = info.Discriminator
			if name == "" {
				return fmt.Errorf("missing name or discriminator")
			}
		}

		if info.Discriminator != "" && b.opts.DiscriminatorField == "" {
			return fmt.Errorf("missing discriminator field")
		}

		b.setting = append(b.setting, SettingsDefinition{
			ObjectMeta: ObjectMeta{
				Name: name,
			},
			Spec: SettingsDefinitionSpec{
				DiscriminatorField: b.opts.DiscriminatorField,
				DiscriminatorValue: info.Discriminator,
				JSONDataSchema:     schema,
			},
		})
	}
	return nil
}

// whitespaceRegex is the regex for consecutive whitespaces.
var whitespaceRegex = regexp.MustCompile(`\s+`)

func (b *SchemaBuilder) enumify(s *jsonschema.Schema) {
	if len(s.Enum) > 0 && s.Extras != nil {
		extra, ok := s.Extras["x-enum-description"]
		if !ok {
			return
		}

		lookup, ok := extra.(map[string]string)
		if !ok {
			return
		}

		lines := []string{}
		if s.Description != "" {
			lines = append(lines, s.Description, "\n")
		}
		lines = append(lines, "Possible enum values:")
		for _, v := range s.Enum {
			c := lookup[v.(string)]
			c = whitespaceRegex.ReplaceAllString(c, " ")
			lines = append(lines, fmt.Sprintf(" - `%q` %s", v, c))
		}

		s.Description = strings.Join(lines, "\n")
		return
	}

	for pair := s.Properties.Oldest(); pair != nil; pair = pair.Next() {
		b.enumify(pair.Value)
	}
}

// Update the schema definition file
// When placed in `static/schema/query.schema.json` folder of a plugin distribution,
// it can be used to advertise various query types
// If the spec contents have changed, the test will fail (but still update the output)
func (b *SchemaBuilder) UpdateQueryDefinition(t *testing.T, outfile string) {
	t.Helper()

	now := time.Now().UTC()
	rv := fmt.Sprintf("%d", now.UnixMilli())

	defs := QueryTypeDefinitionList{}
	byName := make(map[string]*QueryTypeDefinition)
	body, err := os.ReadFile(outfile)
	if err == nil {
		err = json.Unmarshal(body, &defs)
		if err == nil {
			for i, def := range defs.Items {
				byName[def.ObjectMeta.Name] = &defs.Items[i]
			}
		}
	}
	defs.Kind = "QueryTypeDefinitionList"
	defs.ApiVersion = "query.grafana.app/v0alpha1"

	// The updated schemas
	for _, def := range b.query {
		found, ok := byName[def.ObjectMeta.Name]
		if !ok {
			defs.ObjectMeta.ResourceVersion = rv
			def.ObjectMeta.ResourceVersion = rv
			def.ObjectMeta.CreationTimestamp = now.Format(time.RFC3339)

			defs.Items = append(defs.Items, def)
		} else {
			var o1, o2 interface{}
			b1, _ := json.Marshal(def.Spec)
			b2, _ := json.Marshal(found.Spec)
			_ = json.Unmarshal(b1, &o1)
			_ = json.Unmarshal(b2, &o2)
			if !reflect.DeepEqual(o1, o2) {
				found.ObjectMeta.ResourceVersion = rv
				found.Spec = def.Spec
			}
			delete(byName, def.ObjectMeta.Name)
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
}

// Update the schema definition file
// When placed in `static/schema/query.schema.json` folder of a plugin distribution,
// it can be used to advertise various query types
// If the spec contents have changed, the test will fail (but still update the output)
func (b *SchemaBuilder) UpdateSettingsDefinition(t *testing.T, outfile string) {
	t.Helper()

	now := time.Now().UTC()
	rv := fmt.Sprintf("%d", now.UnixMilli())

	defs := QueryTypeDefinitionList{}
	byName := make(map[string]*QueryTypeDefinition)
	body, err := os.ReadFile(outfile)
	if err == nil {
		err = json.Unmarshal(body, &defs)
		if err == nil {
			for i, def := range defs.Items {
				byName[def.ObjectMeta.Name] = &defs.Items[i]
			}
		}
	}
	defs.Kind = "SettingsDefinitionList"
	defs.ApiVersion = "common.grafana.app/v0alpha1"

	// The updated schemas
	for _, def := range b.query {
		found, ok := byName[def.ObjectMeta.Name]
		if !ok {
			defs.ObjectMeta.ResourceVersion = rv
			def.ObjectMeta.ResourceVersion = rv
			def.ObjectMeta.CreationTimestamp = now.Format(time.RFC3339)

			defs.Items = append(defs.Items, def)
		} else {
			var o1, o2 interface{}
			b1, _ := json.Marshal(def.Spec)
			b2, _ := json.Marshal(found.Spec)
			_ = json.Unmarshal(b1, &o1)
			_ = json.Unmarshal(b2, &o2)
			if !reflect.DeepEqual(o1, o2) {
				found.ObjectMeta.ResourceVersion = rv
				found.Spec = def.Spec
			}
			delete(byName, def.ObjectMeta.Name)
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
}
