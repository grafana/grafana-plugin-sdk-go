package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// The k8s compatible jsonschema version
const draft04 = "https://json-schema.org/draft-04/schema"

// SchemaBuilder is a helper function that can be used by
// backend build processes to produce static schema definitions
// This is not intended as runtime code, and is not the only way to
// produce a schema (we may also want/need to use typescript as the source)
type Builder struct {
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

	// explicitly define the enumeration fields
	Enums []reflect.Type
}

type QueryTypeInfo struct {
	// The management name
	Name string
	// Optional discriminators
	Discriminators []DiscriminatorFieldValue
	// Raw GO type used for reflection
	GoType reflect.Type
	// Add sample queries
	Examples []QueryExample
}

type SettingTypeInfo struct {
	// The management name
	Name string
	// Optional discriminators
	Discriminators []DiscriminatorFieldValue
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
							"x-enum-dictionary": enumValueDescriptions,
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

		b.enumify(schema)

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

		b.query = append(b.query, QueryTypeDefinition{
			ObjectMeta: ObjectMeta{
				Name: name,
			},
			Spec: QueryTypeDefinitionSpec{
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

		b.enumify(schema)

		// used by kube-openapi
		schema.Version = draft04
		schema.ID = ""
		schema.Anchor = ""

		b.setting = append(b.setting, SettingsDefinition{
			ObjectMeta: ObjectMeta{
				Name: name,
			},
			Spec: SettingsDefinitionSpec{
				Discriminators: info.Discriminators,
				JSONDataSchema: schema,
			},
		})
	}
	return nil
}

// whitespaceRegex is the regex for consecutive whitespaces.
var whitespaceRegex = regexp.MustCompile(`\s+`)

func (b *Builder) enumify(s *jsonschema.Schema) {
	if len(s.Enum) > 0 && s.Extras != nil {
		extra, ok := s.Extras["x-enum-dictionary"]
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
// When placed in `static/schema/query.types.json` folder of a plugin distribution,
// it can be used to advertise various query types
// If the spec contents have changed, the test will fail (but still update the output)
func (b *Builder) UpdateQueryDefinition(t *testing.T, outdir string) QueryTypeDefinitionList {
	t.Helper()

	outfile := filepath.Join(outdir, "query.types.json")
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

	// Make sure the sample queries are actually valid
	_, err = GetExampleQueries(defs)
	require.NoError(t, err)

	// Read query info
	r := new(jsonschema.Reflector)
	r.DoNotReference = true
	err = r.AddGoComments("github.com/grafana/grafana-plugin-sdk-go/experimental/spec", "./")
	require.NoError(t, err)

	query := r.Reflect(&CommonQueryProperties{})
	query.Version = draft04 // used by kube-openapi
	query.Description = "Query properties shared by all data sources"

	// Now update the query files
	//----------------------------
	outfile = filepath.Join(outdir, "query.post.schema.json")
	schema, err := toQuerySchema(query, defs, false)
	require.NoError(t, err)

	body, _ = os.ReadFile(outfile)
	maybeUpdateFile(t, outfile, schema, body)

	// Now update the query files
	//----------------------------
	outfile = filepath.Join(outdir, "query.save.schema.json")
	schema, err = toQuerySchema(query, defs, true)
	require.NoError(t, err)

	body, _ = os.ReadFile(outfile)
	maybeUpdateFile(t, outfile, schema, body)
	return defs
}

// Update the schema definition file
// When placed in `static/schema/query.schema.json` folder of a plugin distribution,
// it can be used to advertise various query types
// If the spec contents have changed, the test will fail (but still update the output)
func (b *Builder) UpdateSettingsDefinition(t *testing.T, outfile string) SettingsDefinitionList {
	t.Helper()

	now := time.Now().UTC()
	rv := fmt.Sprintf("%d", now.UnixMilli())

	defs := SettingsDefinitionList{}
	byName := make(map[string]*SettingsDefinition)
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

// Converts a set of queries into a single real schema (merged with the common properties)
func toQuerySchema(generic *jsonschema.Schema, defs QueryTypeDefinitionList, saveModel bool) (*jsonschema.Schema, error) {
	descr := "Query model (the payload sent to /ds/query)"
	if saveModel {
		descr = "Save model (the payload saved in dashboards and alerts)"
	}

	ignoreForSave := map[string]bool{"maxDataPoints": true, "intervalMs": true, "timeRange": true}
	definitions := make(jsonschema.Definitions)
	common := make(map[string]*jsonschema.Schema)
	for pair := generic.Properties.Oldest(); pair != nil; pair = pair.Next() {
		if saveModel && ignoreForSave[pair.Key] {
			continue //
		}
		definitions[pair.Key] = pair.Value
		common[pair.Key] = &jsonschema.Schema{Ref: "#/definitions/" + pair.Key}
	}

	// The types for each query type
	queryTypes := []*jsonschema.Schema{}
	for _, qt := range defs.Items {
		node, err := asJSONSchema(qt.Spec.QuerySchema)
		node.Version = ""
		if err != nil {
			return nil, fmt.Errorf("error reading query types schema: %s // %w", qt.ObjectMeta.Name, err)
		}
		if node == nil {
			return nil, fmt.Errorf("missing query schema: %s // %v", qt.ObjectMeta.Name, qt)
		}

		// Match all discriminators
		for _, d := range qt.Spec.Discriminators {
			ds, ok := node.Properties.Get(d.Field)
			if !ok {
				ds = &jsonschema.Schema{Type: "string"}
				node.Properties.Set(d.Field, ds)
			}
			ds.Pattern = `^` + d.Value + `$`
			node.Required = append(node.Required, d.Field)
		}

		queryTypes = append(queryTypes, node)
	}

	// Single node -- just union the global and local properties
	if len(queryTypes) == 1 {
		node := queryTypes[0]
		node.Version = draft04
		node.Description = descr
		node.Definitions = definitions
		for pair := generic.Properties.Oldest(); pair != nil; pair = pair.Next() {
			_, found := node.Properties.Get(pair.Key)
			if found {
				continue
			}
			node.Properties.Set(pair.Key, pair.Value)
		}
		return node, nil
	}

	s := &jsonschema.Schema{
		Type:        "object",
		Version:     draft04,
		Properties:  jsonschema.NewProperties(),
		Definitions: definitions,
		Description: descr,
	}

	for _, qt := range queryTypes {
		qt.Required = append(qt.Required, "refId")

		for k, v := range common {
			_, found := qt.Properties.Get(k)
			if found {
				continue
			}
			qt.Properties.Set(k, v)
		}

		s.OneOf = append(s.OneOf, qt)
	}
	return s, nil
}

func asJSONSchema(v any) (*jsonschema.Schema, error) {
	s, ok := v.(*jsonschema.Schema)
	if ok {
		return s, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	s = &jsonschema.Schema{}
	err = json.Unmarshal(b, s)
	return s, err
}

func asGenericDataQuery(v any) (*GenericDataQuery, error) {
	s, ok := v.(*GenericDataQuery)
	if ok {
		return s, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	s = &GenericDataQuery{}
	err = json.Unmarshal(b, s)
	return s, err
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

func GetExampleQueries(defs QueryTypeDefinitionList) (QueryRequest[GenericDataQuery], error) {
	rsp := QueryRequest[GenericDataQuery]{
		Queries: []GenericDataQuery{},
	}
	for _, def := range defs.Items {
		for _, sample := range def.Spec.Examples {
			if sample.SaveModel != nil {
				q, err := asGenericDataQuery(sample.SaveModel)
				if err != nil {
					return rsp, fmt.Errorf("invalid sample save query [%s], in %s // %w",
						sample.Name, def.ObjectMeta.Name, err)
				}
				q.RefID = string(rune('A' + len(rsp.Queries)))
				rsp.Queries = append(rsp.Queries, *q)
			}
		}
	}
	return rsp, nil
}
