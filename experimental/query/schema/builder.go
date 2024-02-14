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
	"github.com/grafana/grafana-plugin-sdk-go/experimental/query"
	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type QueryTypeInfo struct {
	// The management name
	Name string
	// The discriminator value (requires the field set in ops)
	Discriminator string
	// Raw GO type used for reflection
	GoType reflect.Type
	// Add sample queries
	Examples []query.QueryExample
}

type QueryTypeBuilder struct {
	t         *testing.T
	opts      BuilderOptions
	reflector *jsonschema.Reflector // Needed to use comments
	defs      []query.QueryTypeDefinition
}

func (b *QueryTypeBuilder) Add(info QueryTypeInfo) error {
	schema := b.reflector.ReflectFromType(info.GoType)
	if schema == nil {
		return fmt.Errorf("missing schema")
	}

	b.enumify(schema)

	// Ignored by k8s anyway
	schema.Version = ""
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

	b.defs = append(b.defs, query.QueryTypeDefinition{
		ObjectMeta: query.ObjectMeta{
			Name: name,
		},
		Spec: query.QueryTypeDefinitionSpec{
			DiscriminatorField: b.opts.DiscriminatorField,
			DiscriminatorValue: info.Discriminator,
			Schema:             schema,
			Examples:           info.Examples,
		},
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

	// explicitly define the enumeration fields
	Enums []reflect.Type
}

func NewBuilder(t *testing.T, opts BuilderOptions, inputs ...QueryTypeInfo) (*QueryTypeBuilder, error) {
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

	b := &QueryTypeBuilder{
		t:         t,
		opts:      opts,
		reflector: r,
	}
	for _, input := range inputs {
		err := b.Add(input)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

// whitespaceRegex is the regex for consecutive whitespaces.
var whitespaceRegex = regexp.MustCompile(`\s+`)

func (b *QueryTypeBuilder) enumify(s *jsonschema.Schema) {
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
	for _, def := range b.defs {
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
	return out
}
