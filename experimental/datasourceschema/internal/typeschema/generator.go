package typeschema

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/load"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
)

const draft04 = "https://json-schema.org/draft-04/schema#"

type Builder struct {
	load       *load.Result
	opts       SchemaOptions
	namedCache map[string]map[string]any
	inProgress map[string]struct{}
	namedDocs  map[string]string
	fieldDocs  map[string]string
	enumDocs   map[string]map[string]string
}

type SchemaOptions struct {
	IncludeRequired                bool
	RequireJSONTags                bool
	FallbackToSimpleUntaggedFields bool
	LowerCamelUntaggedFields       bool
}

func defaultSchemaOptions() SchemaOptions {
	return SchemaOptions{
		IncludeRequired: false,
	}
}

func BuildNamedTypeSchema(loadRes *load.Result, packagePath string, typeName string) (map[string]any, error) {
	return BuildNamedTypeSchemaWithOptions(loadRes, packagePath, typeName, defaultSchemaOptions())
}

func BuildNamedTypeSchemaWithOptions(loadRes *load.Result, packagePath string, typeName string, opts SchemaOptions) (map[string]any, error) {
	builder := &Builder{
		load:       loadRes,
		opts:       opts,
		namedCache: map[string]map[string]any{},
		inProgress: map[string]struct{}{},
		namedDocs:  map[string]string{},
		fieldDocs:  map[string]string{},
		enumDocs:   map[string]map[string]string{},
	}
	builder.collectDocs()

	namedType, err := builder.lookupNamedType(packagePath, typeName)
	if err != nil {
		return nil, err
	}

	schema, err := builder.schemaForType(namedType)
	if err != nil {
		return nil, err
	}
	schema["$schema"] = draft04

	return schema, nil
}

func BuildTargetSchema(loadRes *load.Result, target *model.TargetRef, opts SchemaOptions) (map[string]any, error) {
	if target == nil {
		return nil, fmt.Errorf("target is nil")
	}

	if target.TypeName != "" {
		return BuildNamedTypeSchemaWithOptions(loadRes, target.PackagePath, target.TypeName, opts)
	}

	builder := &Builder{
		load:       loadRes,
		opts:       opts,
		namedCache: map[string]map[string]any{},
		inProgress: map[string]struct{}{},
		namedDocs:  map[string]string{},
		fieldDocs:  map[string]string{},
		enumDocs:   map[string]map[string]string{},
	}
	builder.collectDocs()

	typ, err := builder.lookupTypeAtPosition(target)
	if err != nil {
		return nil, err
	}

	schema, err := builder.schemaForType(typ)
	if err != nil {
		return nil, err
	}
	schema["$schema"] = draft04

	return schema, nil
}

func (b *Builder) lookupNamedType(packagePath string, typeName string) (*types.Named, error) {
	for _, pkg := range b.load.Packages {
		if pkg.Types == nil || pkg.PkgPath != packagePath {
			continue
		}

		object := pkg.Types.Scope().Lookup(typeName)
		if object == nil {
			break
		}

		typeNameObject, ok := object.(*types.TypeName)
		if !ok {
			return nil, fmt.Errorf("%s.%s is not a named type", packagePath, typeName)
		}

		named, ok := typeNameObject.Type().(*types.Named)
		if !ok {
			return nil, fmt.Errorf("%s.%s is not a named type", packagePath, typeName)
		}

		return named, nil
	}

	return nil, fmt.Errorf("unable to find named type %s.%s", packagePath, typeName)
}

func (b *Builder) schemaForType(t types.Type) (map[string]any, error) {
	switch value := t.(type) {
	case *types.Named:
		return b.schemaForNamed(value)
	case *types.Pointer:
		return b.schemaForType(value.Elem())
	case *types.Struct:
		return b.schemaForStruct(value)
	case *types.Slice:
		itemSchema, err := b.schemaForType(value.Elem())
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"type":  "array",
			"items": itemSchema,
		}, nil
	case *types.Array:
		itemSchema, err := b.schemaForType(value.Elem())
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"type":  "array",
			"items": itemSchema,
		}, nil
	case *types.Map:
		valueSchema, err := b.schemaForType(value.Elem())
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"type":                 "object",
			"additionalProperties": valueSchema,
		}, nil
	case *types.Basic:
		return schemaForBasic(value), nil
	case *types.Interface:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": true,
		}, nil
	default:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": true,
		}, nil
	}
}

func (b *Builder) schemaForNamed(named *types.Named) (map[string]any, error) {
	key := namedKey(named)
	if schema, ok := b.namedCache[key]; ok {
		return cloneMap(schema), nil
	}
	if _, ok := b.inProgress[key]; ok {
		return map[string]any{
			"type":                 "object",
			"additionalProperties": true,
		}, nil
	}

	b.inProgress[key] = struct{}{}
	defer delete(b.inProgress, key)

	var schema map[string]any
	var err error
	if special, ok := specialNamedTypeSchema(named); ok {
		schema = special
	} else if st, ok := named.Underlying().(*types.Struct); ok {
		schema, err = b.schemaForNamedStruct(key, st)
	} else {
		schema, err = b.schemaForType(named.Underlying())
	}
	if err != nil {
		return nil, err
	}

	if enumValues := enumValuesForNamed(named); len(enumValues) > 0 {
		schema["enum"] = enumValues
		b.decorateEnumDescription(key, schema, enumValues)
	} else if description := b.namedDocs[key]; description != "" {
		schema["description"] = description
	}

	b.namedCache[key] = cloneMap(schema)

	return schema, nil
}

func (b *Builder) schemaForStruct(st *types.Struct) (map[string]any, error) {
	return b.schemaForNamedStruct("", st)
}

func (b *Builder) schemaForNamedStruct(parentKey string, st *types.Struct) (map[string]any, error) {
	properties := map[string]any{}
	required := make([]string, 0)
	allowSimpleUntaggedFields := b.opts.RequireJSONTags && b.opts.FallbackToSimpleUntaggedFields && !hasExplicitJSONFields(st)

	for index := 0; index < st.NumFields(); index++ {
		field := st.Field(index)
		if !field.Exported() && !field.Anonymous() {
			continue
		}

		jsonName, omitEmpty, skip, explicitName, hasTag := jsonField(field, st.Tag(index))
		if skip {
			continue
		}
		if b.opts.RequireJSONTags && !hasTag {
			if !allowSimpleUntaggedFields || !isSimpleJSONFieldType(field.Type()) {
				continue
			}
		}
		if jsonName == "" {
			continue
		}
		if !hasTag && b.opts.LowerCamelUntaggedFields && !explicitName {
			jsonName = lowerCamelJSONName(jsonName)
		}

		if field.Anonymous() && !explicitName {
			embeddedSchema, err := b.schemaForType(field.Type())
			if err != nil {
				return nil, err
			}
			embeddedProperties, ok := nestedMap(embeddedSchema, "properties")
			if ok {
				for key, value := range embeddedProperties {
					properties[key] = value
				}
			}
			if embeddedRequired, ok := stringSlice(embeddedSchema["required"]); ok {
				required = append(required, embeddedRequired...)
			}
			continue
		}

		fieldSchema, err := b.schemaForType(field.Type())
		if err != nil {
			return nil, err
		}
		if description := b.fieldDocs[parentKey+"."+field.Name()]; description != "" {
			fieldSchema["description"] = description
		}

		properties[jsonName] = fieldSchema
		if b.opts.IncludeRequired && !omitEmpty {
			required = append(required, jsonName)
		}
	}

	required = dedupeStrings(required)
	sort.Strings(required)

	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
	}
	if len(required) > 0 {
		items := make([]any, 0, len(required))
		for _, item := range required {
			items = append(items, item)
		}
		schema["required"] = items
	}

	return schema, nil
}

func schemaForBasic(basic *types.Basic) map[string]any {
	info := basic.Info()
	switch {
	case info&types.IsBoolean != 0:
		return map[string]any{"type": "boolean"}
	case info&types.IsInteger != 0:
		return map[string]any{"type": "integer"}
	case info&types.IsFloat != 0:
		return map[string]any{"type": "number"}
	case info&types.IsString != 0:
		return map[string]any{"type": "string"}
	default:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": true,
		}
	}
}

func (b *Builder) collectDocs() {
	for _, pkg := range b.load.Packages {
		if pkg.TypesInfo == nil {
			continue
		}

		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}

				switch genDecl.Tok {
				case token.TYPE:
					b.collectTypeDocs(pkg.PkgPath, pkg.TypesInfo, genDecl)
				case token.CONST:
					b.collectConstDocs(pkg.TypesInfo, genDecl)
				}
			}
		}
	}
}

func (b *Builder) collectTypeDocs(packagePath string, info *types.Info, genDecl *ast.GenDecl) {
	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		object, ok := info.Defs[typeSpec.Name].(*types.TypeName)
		if !ok {
			continue
		}

		named, ok := object.Type().(*types.Named)
		if !ok {
			continue
		}

		key := namedKey(named)
		if description := normalizeComment(preferComment(typeSpec.Doc, genDecl.Doc)); description != "" {
			b.namedDocs[key] = description
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		for _, field := range structType.Fields.List {
			description := normalizeComment(preferComment(field.Doc, field.Comment))
			if !shouldKeepFieldDescription(description) {
				continue
			}
			for _, name := range field.Names {
				if variable, ok := info.Defs[name].(*types.Var); ok {
					b.fieldDocs[key+"."+variable.Name()] = description
				}
			}
		}
	}
}

func (b *Builder) collectConstDocs(info *types.Info, genDecl *ast.GenDecl) {
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		description := normalizeComment(preferComment(valueSpec.Doc, valueSpec.Comment))
		if description == "" {
			continue
		}

		for _, name := range valueSpec.Names {
			constantObject, ok := info.Defs[name].(*types.Const)
			if !ok {
				continue
			}

			named, ok := constantObject.Type().(*types.Named)
			if !ok {
				continue
			}

			key := namedKey(named)
			if _, ok := b.enumDocs[key]; !ok {
				b.enumDocs[key] = map[string]string{}
			}

			switch constantObject.Val().Kind() {
			case constant.String:
				b.enumDocs[key][constant.StringVal(constantObject.Val())] = description
			case constant.Bool:
				b.enumDocs[key][fmt.Sprintf("%t", constant.BoolVal(constantObject.Val()))] = description
			case constant.Int:
				if number, ok := constant.Int64Val(constantObject.Val()); ok {
					b.enumDocs[key][fmt.Sprintf("%d", number)] = description
				}
			}
		}
	}
}

func (b *Builder) decorateEnumDescription(key string, schema map[string]any, enumValues []any) {
	lines := make([]string, 0)
	if description := b.namedDocs[key]; description != "" {
		lines = append(lines, description, "")
	}

	enumComments := b.enumDocs[key]
	if len(enumComments) == 0 {
		return
	}

	lines = append(lines, "Possible enum values:")
	for _, value := range enumValues {
		text := fmt.Sprintf("%v", value)
		comment := normalizeWhitespace(enumComments[text])
		lines = append(lines, fmt.Sprintf(" - `%q` %s", text, comment))
	}

	schema["description"] = strings.Join(lines, "\n")
}

func enumValuesForNamed(named *types.Named) []any {
	object := named.Obj()
	if object == nil || object.Pkg() == nil {
		return nil
	}

	values := make([]any, 0)
	for _, name := range object.Pkg().Scope().Names() {
		constantObject, ok := object.Pkg().Scope().Lookup(name).(*types.Const)
		if !ok || !types.Identical(constantObject.Type(), named) {
			continue
		}

		switch constantObject.Val().Kind() {
		case constant.String:
			values = append(values, constant.StringVal(constantObject.Val()))
		case constant.Bool:
			values = append(values, constant.BoolVal(constantObject.Val()))
		case constant.Int:
			number, ok := constant.Int64Val(constantObject.Val())
			if ok {
				values = append(values, number)
			}
		}
	}

	if len(values) == 0 {
		return nil
	}

	return values
}

func (b *Builder) lookupTypeAtPosition(target *model.TargetRef) (types.Type, error) {
	if target == nil || target.Expr == nil {
		return nil, fmt.Errorf("missing expression position for anonymous target")
	}

	for _, pkg := range b.load.Packages {
		if pkg.PkgPath != target.PackagePath || pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}

		for _, file := range pkg.Syntax {
			var found types.Type
			ast.Inspect(file, func(node ast.Node) bool {
				if found != nil || node == nil {
					return found == nil
				}

				expr, ok := node.(ast.Expr)
				if !ok {
					return true
				}

				pos := pkg.Fset.PositionFor(expr.Pos(), false)
				if pos.Filename != target.Expr.File || pos.Line != target.Expr.Line || pos.Column != target.Expr.Column {
					return true
				}

				found = pkg.TypesInfo.TypeOf(expr)
				return false
			})

			if found != nil {
				return found, nil
			}
		}
	}

	return nil, fmt.Errorf("unable to find target expression at %s:%d:%d", target.Expr.File, target.Expr.Line, target.Expr.Column)
}

func jsonField(field *types.Var, tag string) (name string, omitEmpty bool, skip bool, explicitName bool, hasTag bool) {
	jsonTag := reflect.StructTag(tag).Get("json")
	if jsonTag == "-" {
		return "", false, true, false, true
	}

	name = field.Name()
	if jsonTag == "" {
		return name, false, false, false, false
	}
	hasTag = true

	parts := strings.Split(jsonTag, ",")
	if len(parts) > 0 && parts[0] == "-" {
		return "", false, true, false, true
	}
	if parts[0] != "" {
		name = parts[0]
		explicitName = true
	}
	for _, part := range parts[1:] {
		if part == "omitempty" {
			omitEmpty = true
		}
	}

	return name, omitEmpty, false, explicitName, hasTag
}

func hasExplicitJSONFields(st *types.Struct) bool {
	for index := 0; index < st.NumFields(); index++ {
		field := st.Field(index)
		if !field.Exported() && !field.Anonymous() {
			continue
		}

		_, _, skip, _, hasTag := jsonField(field, st.Tag(index))
		if skip || !hasTag {
			continue
		}
		return true
	}

	return false
}

func isSimpleJSONFieldType(typ types.Type) bool {
	if typ == nil {
		return false
	}

	switch typed := typ.(type) {
	case *types.Pointer:
		return isSimpleJSONFieldType(typed.Elem())
	case *types.Named:
		if _, ok := specialNamedTypeSchema(typed); ok {
			return true
		}
		return isSimpleJSONFieldType(typed.Underlying())
	case *types.Basic:
		return true
	case *types.Slice:
		return isSimpleJSONFieldType(typed.Elem())
	case *types.Array:
		return isSimpleJSONFieldType(typed.Elem())
	case *types.Map:
		key, ok := typed.Key().Underlying().(*types.Basic)
		return ok && key.Kind() == types.String && isSimpleJSONFieldType(typed.Elem())
	default:
		return false
	}
}

func specialNamedTypeSchema(named *types.Named) (map[string]any, bool) {
	if named == nil || named.Obj() == nil || named.Obj().Pkg() == nil {
		return nil, false
	}

	switch named.Obj().Pkg().Path() + "." + named.Obj().Name() {
	case "github.com/google/uuid.UUID":
		return map[string]any{
			"type":   "string",
			"format": "uuid",
		}, true
	case "time.Time":
		return map[string]any{
			"type":   "string",
			"format": "date-time",
		}, true
	case "encoding/json.RawMessage":
		return map[string]any{
			"type":                 "object",
			"additionalProperties": true,
		}, true
	default:
		return nil, false
	}
}

func lowerCamelJSONName(name string) string {
	if name == "" {
		return ""
	}

	runes := []rune(name)
	if len(runes) == 1 {
		return strings.ToLower(name)
	}

	cutoff := 1
	for cutoff < len(runes) {
		current := runes[cutoff]
		if !unicode.IsUpper(current) {
			break
		}
		if cutoff+1 < len(runes) && unicode.IsLower(runes[cutoff+1]) {
			break
		}
		cutoff++
	}

	prefix := strings.ToLower(string(runes[:cutoff]))
	return prefix + string(runes[cutoff:])
}

func namedKey(named *types.Named) string {
	object := named.Obj()
	if object == nil {
		return named.String()
	}
	if object.Pkg() == nil {
		return object.Name()
	}

	return object.Pkg().Path() + "." + object.Name()
}

func preferComment(primary *ast.CommentGroup, fallback *ast.CommentGroup) *ast.CommentGroup {
	if primary != nil {
		return primary
	}
	return fallback
}

func normalizeComment(group *ast.CommentGroup) string {
	if group == nil {
		return ""
	}
	return normalizeWhitespace(group.Text())
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func shouldKeepFieldDescription(description string) bool {
	if description == "" {
		return false
	}
	if strings.ContainsAny(description, ".:;!?") {
		return true
	}

	words := strings.Fields(description)
	if len(words) >= 4 {
		return true
	}

	return false
}

func nestedMap(value map[string]any, keys ...string) (map[string]any, bool) {
	current := value
	for _, key := range keys {
		next, ok := current[key]
		if !ok {
			return nil, false
		}
		current, ok = next.(map[string]any)
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func stringSlice(value any) ([]string, bool) {
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, text)
	}
	return out, true
}

func dedupeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func cloneMap(value map[string]any) map[string]any {
	out := make(map[string]any, len(value))
	for key, item := range value {
		if nested, ok := item.(map[string]any); ok {
			out[key] = cloneMap(nested)
			continue
		}
		if list, ok := item.([]any); ok {
			copied := make([]any, len(list))
			copy(copied, list)
			out[key] = copied
			continue
		}
		out[key] = item
	}
	return out
}
