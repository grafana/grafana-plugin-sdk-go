package schemabuilder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/resource"
	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/kube-openapi/pkg/validation/strfmt"
	"k8s.io/kube-openapi/pkg/validation/validate"
)

// SchemaBuilder is a helper function that can be used by
// backend build processes to produce static schema definitions
// This is not intended as runtime code, and is not the only way to
// produce a schema (we may also want/need to use typescript as the source)
type Builder struct {
	opts      BuilderOptions
	reflector *jsonschema.Reflector // Needed to use comments
	query     []resource.QueryTypeDefinition
	setting   []resource.SettingsDefinition
}

type BuilderOptions struct {
	// The plugin type ID used in the DataSourceRef type property
	PluginID []string

	// ex "github.com/grafana/github-datasource/pkg/models"
	BasePackage string

	// ex "./"
	CodePath string

	// explicitly define the enumeration fields
	Enums []reflect.Type
}

type QueryTypeInfo struct {
	// The management name
	Name string
	// Optional discriminators
	Discriminators []resource.DiscriminatorFieldValue
	// Raw GO type used for reflection
	GoType reflect.Type
	// Add sample queries
	Examples []resource.QueryExample
}

type SettingTypeInfo struct {
	// The management name
	Name string
	// Optional discriminators
	Discriminators []resource.DiscriminatorFieldValue
	// Raw GO type used for reflection
	GoType reflect.Type
	// Map[string]string
	SecureGoType reflect.Type
}

func NewSchemaBuilder(opts BuilderOptions) (*Builder, error) {
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

	return &Builder{
		opts:      opts,
		reflector: r,
	}, nil
}

func (b *Builder) AddQueries(inputs ...QueryTypeInfo) error {
	for _, info := range inputs {
		schema := b.reflector.ReflectFromType(info.GoType)
		if schema == nil {
			return fmt.Errorf("missing schema")
		}

		UpdateEnumDescriptions(schema)

		name := info.Name
		if name == "" {
			for _, dis := range info.Discriminators {
				if name != "" {
					name += "-"
				}
				name += dis.Value
			}
			if name == "" {
				return fmt.Errorf("missing name or discriminators")
			}
		}

		// We need to be careful to only use draft-04 so that this is possible to use
		// with kube-openapi
		schema.Version = draft04
		schema.ID = ""
		schema.Anchor = ""

		b.query = append(b.query, resource.QueryTypeDefinition{
			ObjectMeta: resource.ObjectMeta{
				Name: name,
			},
			Spec: resource.QueryTypeDefinitionSpec{
				Discriminators: info.Discriminators,
				QuerySchema:    schema,
				Examples:       info.Examples,
			},
		})
	}
	return nil
}

func (b *Builder) AddSettings(inputs ...SettingTypeInfo) error {
	for _, info := range inputs {
		name := info.Name
		if name == "" {
			return fmt.Errorf("missing name")
		}

		schema := b.reflector.ReflectFromType(info.GoType)
		if schema == nil {
			return fmt.Errorf("missing schema")
		}

		UpdateEnumDescriptions(schema)

		// used by kube-openapi
		schema.Version = draft04
		schema.ID = ""
		schema.Anchor = ""

		b.setting = append(b.setting, resource.SettingsDefinition{
			ObjectMeta: resource.ObjectMeta{
				Name: name,
			},
			Spec: resource.SettingsDefinitionSpec{
				Discriminators: info.Discriminators,
				JSONDataSchema: schema,
			},
		})
	}
	return nil
}

// Update the schema definition file
// When placed in `static/schema/query.types.json` folder of a plugin distribution,
// it can be used to advertise various query types
// If the spec contents have changed, the test will fail (but still update the output)
func (b *Builder) UpdateQueryDefinition(t *testing.T, outdir string) resource.QueryTypeDefinitionList {
	t.Helper()

	outfile := filepath.Join(outdir, "query.types.json")
	now := time.Now().UTC()
	rv := fmt.Sprintf("%d", now.UnixMilli())

	defs := resource.QueryTypeDefinitionList{}
	byName := make(map[string]*resource.QueryTypeDefinition)
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
	defs.APIVersion = "query.grafana.app/v0alpha1"

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
	maybeUpdateFile(t, outfile, defs, body)

	// Update the query save model schema
	//------------------------------------
	outfile = filepath.Join(outdir, "query.schema.json")
	schema, err := GetQuerySchema(QuerySchemaOptions{
		QueryTypes: defs.Items,
		Mode:       SchemaTypePanelModel,
	})
	require.NoError(t, err)

	body, _ = os.ReadFile(outfile)
	maybeUpdateFile(t, outfile, schema, body)

	// Update the request payload schema
	//------------------------------------
	outfile = filepath.Join(outdir, "query.request.schema.json")
	schema, err = GetQuerySchema(QuerySchemaOptions{
		QueryTypes: defs.Items,
		Mode:       SchemaTypeQueryRequest,
	})
	require.NoError(t, err)

	body, _ = os.ReadFile(outfile)
	maybeUpdateFile(t, outfile, schema, body)

	// Verify that the example queries actually validate
	//------------------------------------
	request, err := GetExampleQueries(defs)
	require.NoError(t, err)

	outfile = filepath.Join(outdir, "query.request.examples.json")
	body, _ = os.ReadFile(outfile)
	maybeUpdateFile(t, outfile, request, body)

	validator := validate.NewSchemaValidator(schema, nil, "", strfmt.Default)
	result := validator.Validate(request)
	if result.HasErrorsOrWarnings() {
		body, err = json.MarshalIndent(result, "", "  ")
		require.NoError(t, err)
		fmt.Printf("Validation: %s\n", string(body))
		require.Fail(t, "validation failed")
	}
	require.True(t, result.MatchCount > 0, "must have some rules")
	return defs
}

// Update the schema definition file
// When placed in `static/schema/query.schema.json` folder of a plugin distribution,
// it can be used to advertise various query types
// If the spec contents have changed, the test will fail (but still update the output)
func (b *Builder) UpdateSettingsDefinition(t *testing.T, outfile string) resource.SettingsDefinitionList {
	t.Helper()

	now := time.Now().UTC()
	rv := fmt.Sprintf("%d", now.UnixMilli())

	defs := resource.SettingsDefinitionList{}
	byName := make(map[string]*resource.SettingsDefinition)
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
	defs.APIVersion = "common.grafana.app/v0alpha1"

	// The updated schemas
	for _, def := range b.setting {
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
		require.FailNow(t, "settings type removed, manually update (for now)")
	}
	return defs
}

func maybeUpdateFile(t *testing.T, outfile string, value any, body []byte) {
	t.Helper()

	out, err := json.MarshalIndent(value, "", "  ")
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
		err = os.WriteFile(outfile, out, 0600)
		require.NoError(t, err, "error writing file")
	}
}
