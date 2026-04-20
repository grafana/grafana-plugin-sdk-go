package schemabuilder

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	sdkapi "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
)

// SchemaBuilder is a helper function that can be used by
// backend build processes to produce static schema definitions
// This is not intended as runtime code, and is not the only way to
// produce a schema (we may also want/need to use typescript as the source)
type Builder struct {
	opts      BuilderOptions
	reflector *jsonschema.Reflector // Needed to use comments

	// discovered via reflection
	query    []sdkapi.QueryTypeDefinition
	examples sdkapi.QueryExamples

	// Explicitly configured
	settingsSchema   *pluginschema.Settings
	settingsExamples *pluginschema.SettingsExamples
	routes           *pluginschema.Routes
}

type CodePaths struct {
	// ex "github.com/grafana/github-datasource/pkg/models"
	BasePackage string

	// ex "./"
	CodePath string
}

type BuilderOptions struct {
	// The plugin type ID used in the DataSourceRef type property
	PluginID []string

	// Scan comments and enumerations
	ScanCode []CodePaths

	// explicitly define the enumeration fields
	Enums []reflect.Type
}

type QueryTypeInfo struct {
	// The management name
	Name string
	// Optional description
	Description string
	// Optional discriminators
	Discriminators []sdkapi.DiscriminatorFieldValue
	// Raw GO type used for reflection
	GoType reflect.Type
	// Add sample queries
	Examples []sdkapi.QueryExample
}

func NewSchemaBuilder(opts BuilderOptions) (*Builder, error) {
	if len(opts.PluginID) < 1 {
		return nil, fmt.Errorf("missing plugin id")
	}

	r := new(jsonschema.Reflector)
	r.DoNotReference = true
	for _, scan := range opts.ScanCode {
		if err := r.AddGoComments(scan.BasePackage, scan.CodePath); err != nil {
			return nil, err
		}
	}
	customMapper := map[reflect.Type]*jsonschema.Schema{
		reflect.TypeFor[data.Frame](): {
			Type: "object",
			Extras: map[string]any{
				"x-grafana-type": "data.DataFrame",
			},
			AdditionalProperties: jsonschema.TrueSchema,
		},
		reflect.TypeFor[sdkapi.Unstructured](): {
			Type:                 "object",
			AdditionalProperties: jsonschema.TrueSchema,
		},
		reflect.TypeFor[sdkapi.JSONSchema](): {
			Type: "object",
			Ref:  draft04,
		},
	}
	r.Mapper = func(t reflect.Type) *jsonschema.Schema {
		return customMapper[t]
	}

	if len(opts.Enums) > 0 {
		fields := []EnumField{}
		for _, scan := range opts.ScanCode {
			enums, err := findEnumFields(scan.BasePackage, scan.CodePath)
			if err != nil {
				return nil, err
			}
			fields = append(fields, enums...)
		}

		for _, etype := range opts.Enums {
			name := etype.Name()
			pack := etype.PkgPath()
			for _, f := range fields {
				if f.Name == name && f.Package == pack {
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

func (b *Builder) Reflector() *jsonschema.Reflector {
	return b.reflector
}

func (b *Builder) AddQueries(inputs []QueryTypeInfo) error {
	for _, info := range inputs {
		schema := b.reflector.ReflectFromType(info.GoType)
		if schema == nil {
			return fmt.Errorf("missing schema")
		}
		updateEnumDescriptions(schema)

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

		// Collect each example
		for _, example := range info.Examples {
			if example.Name == "" {
				return fmt.Errorf("all examples require a name: %+v", example)
			}
			example.QueryType = name
			b.examples.Examples = append(b.examples.Examples, example)
		}

		// We need to be careful to only use draft-04 so that this is possible to use
		// with kube-openapi
		schema.Version = draft04
		schema.ID = ""
		schema.Anchor = ""
		spec, err := asJSONSchema(schema)
		if err != nil {
			return err
		}

		b.query = append(b.query, sdkapi.QueryTypeDefinition{
			ObjectMeta: sdkapi.ObjectMeta{
				Name: name,
			},
			Spec: sdkapi.QueryTypeDefinitionSpec{
				Description:    info.Description,
				Discriminators: info.Discriminators,
				Schema: sdkapi.JSONSchema{
					Spec: spec,
				},
			},
		})
	}
	return nil
}

func (b *Builder) ConfigureSettings(v *pluginschema.Settings, examples *pluginschema.SettingsExamples) error {
	b.settingsSchema = v
	b.settingsExamples = examples
	return nil
}

func (b *Builder) SetRoutes(v *pluginschema.Routes) error {
	if v.IsZero() {
		v = nil
	}
	b.routes = v
	return nil
}

// Update the schema definition file
// When placed in `static/schema/query.types.json` folder of a plugin distribution,
// it can be used to advertise various query types
// If the spec contents have changed, the test will fail (but still update the output)
func (b *Builder) UpdateProviderFiles(t *testing.T, apiVersion, outdir string) {
	t.Helper()

	require.NotEmpty(t, apiVersion, "apiVersion is required")

	outfile := filepath.Join(outdir, apiVersion, "query.types.json")
	now := time.Now().UTC()
	rv := fmt.Sprintf("%d", now.UnixMilli())

	defs := sdkapi.QueryTypeDefinitionList{}
	byName := make(map[string]*sdkapi.QueryTypeDefinition)
	body, err := os.ReadFile(outfile) // #nosec G304
	if err == nil {
		err = json.Unmarshal(body, &defs)
		if err == nil {
			for i, def := range defs.Items {
				byName[def.Name] = &defs.Items[i]
			}
		}
	}
	defs.Kind = "QueryTypeDefinitionList"
	defs.APIVersion = "datasource.grafana.app/v0alpha1"

	// The updated schemas
	for _, def := range b.query {
		found, ok := byName[def.Name]
		if !ok {
			defs.ResourceVersion = rv
			def.ResourceVersion = rv
			def.CreationTimestamp = now.Format(time.RFC3339)

			defs.Items = append(defs.Items, def)
		} else {
			x := sdkapi.AsUnstructured(def.Spec)
			y := sdkapi.AsUnstructured(found.Spec)
			if diff := cmp.Diff(stripNilValues(x.Object), stripNilValues(y.Object), cmpopts.EquateEmpty()); diff != "" {
				fmt.Printf("Spec changed:\n%s\n", diff)
				found.ResourceVersion = rv
				found.Spec = def.Spec
			}
			delete(byName, def.Name)
		}
	}

	if defs.ResourceVersion == "" {
		defs.ResourceVersion = rv
	}

	if len(byName) > 0 {
		require.FailNow(t, "query type removed, manually update (for now)")
	}
	maybeUpdateFile(t, outfile, defs, body)

	outfile = filepath.Join(outdir, apiVersion, "query.examples.json")
	if len(b.examples.Examples) > 0 {
		body, _ := os.ReadFile(outfile) // #nosec G304
		maybeUpdateFile(t, outfile, b.examples, body)
	} else {
		err = os.RemoveAll(outfile)
		require.NoError(t, err)
	}

	// Now check the other files
	provider := pluginschema.NewSchemaProvider(os.DirFS(outdir), "")
	current, err := provider.Get(apiVersion)
	require.NoError(t, err)
	if current == nil {
		current = &pluginschema.PluginSchema{APIVersion: apiVersion}
	}

	// Write helper
	write := func(out []byte, name string) {
		fpath := path.Join(outdir, apiVersion, name)
		err := os.MkdirAll(filepath.Dir(fpath), 0750)
		require.NoError(t, err)
		err = os.WriteFile(fpath, out, 0600)
		require.NoError(t, err)
	}

	if b.settingsSchema != nil {
		if diff := Diff(b.settingsSchema, current.SettingsSchema); diff != "" {
			t.Errorf("settings changed (-want +got):\n%s", diff)
			out, err := json.MarshalIndent(b.settingsSchema, "", "  ")
			require.NoError(t, err)
			write(out, "settings.json")
		}
	}

	if b.settingsExamples != nil {
		if diff := Diff(b.settingsExamples, current.SettingsExamples); diff != "" {
			t.Errorf("settings examples changed (-want +got):\n%s", diff)
			out, err := json.MarshalIndent(b.settingsExamples, "", "  ")
			require.NoError(t, err)
			write(out, "settings.examples.json")
		}
	}

	if b.routes != nil {
		if diff := Diff(b.routes, current.Routes); diff != "" {
			t.Errorf("routes changed (-want +got):\n%s", diff)
			out, err := yaml.Marshal(b.routes)
			require.NoError(t, err)
			write(out, "routes.yaml")
		}
	}
}

func maybeUpdateFile(t *testing.T, outfile string, value any, existing []byte) {
	t.Helper()

	out, err := json.MarshalIndent(value, "", "  ")
	require.NoError(t, err)

	update := false
	if len(existing) > 0 {
		if !assert.JSONEq(t, string(existing), string(out)) {
			update = true
		}
	} else {
		update = true
	}
	if update {
		err := os.MkdirAll(filepath.Dir(outfile), 0750)
		require.NoError(t, err, "creating folder")
		err = os.WriteFile(outfile, out, 0600)
		require.NoError(t, err, "error writing file")
	}
}

func stripNilValues(input map[string]any) map[string]any {
	for k, v := range input {
		if v == nil {
			delete(input, k)
		} else {
			sub, ok := v.(map[string]any)
			if ok {
				stripNilValues(sub)
			}
		}
	}
	return input
}
