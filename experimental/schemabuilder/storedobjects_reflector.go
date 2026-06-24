package schemabuilder

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
)

// StoredObjectInfo declares an input to AddStoredObjects. The Spec field's
// Go type is reflected into an OpenAPI spec.Schema.
type StoredObjectInfo struct {
	// Name is the kind name, e.g. "Watchlist".
	Name string

	// Plural URL form (lowercased), e.g. "watchlists". Optional; derived
	// from Name when empty.
	Plural string

	// Singular URL form (lowercased), e.g. "watchlist". Optional; derived
	// from Name when empty.
	Singular string

	// Scope is "Namespaced" (default) or "Cluster".
	Scope pluginschema.StoredObjectScope

	// SpecType is the Go type representing the object's spec field. The
	// type is reflected and converted to an OpenAPI spec.Schema before
	// being added to the artifact.
	SpecType reflect.Type

	// Validation, when non-empty, opts the kind into validating admission
	// for the listed operations.
	Validation []pluginschema.AdmissionOperation

	// Mutation, when non-empty, opts the kind into mutating admission for
	// the listed operations.
	Mutation []pluginschema.AdmissionOperation
}

// AddStoredObjects reflects each declared object's spec type into an
// OpenAPI schema and appends it to the schema artifact. Mirrors AddQueries.
func (b *Builder) AddStoredObjects(inputs []StoredObjectInfo) error {
	for _, info := range inputs {
		if info.Name == "" {
			return fmt.Errorf("stored object missing name")
		}
		if info.SpecType == nil {
			return fmt.Errorf("stored object %q missing SpecType", info.Name)
		}

		schema := b.reflector.ReflectFromType(info.SpecType)
		if schema == nil {
			return fmt.Errorf("stored object %q: reflection returned nil schema", info.Name)
		}
		updateEnumDescriptions(schema)

		// Stay on draft-04 so the generated schema round-trips through the
		// OpenAPI loader Grafana uses, matching what AddQueries enforces.
		schema.Version = draft04
		schema.ID = ""
		schema.Anchor = ""

		spec, err := asJSONSchema(schema)
		if err != nil {
			return fmt.Errorf("stored object %q: %w", info.Name, err)
		}

		obj := pluginschema.StoredObject{
			Name:       info.Name,
			Plural:     info.Plural,
			Singular:   info.Singular,
			Scope:      info.Scope,
			Spec:       spec,
			Validation: info.Validation,
			Mutation:   info.Mutation,
		}
		if obj.Plural == "" {
			obj.Plural = strings.ToLower(info.Name) + "s"
		}
		if obj.Singular == "" {
			obj.Singular = strings.ToLower(info.Name)
		}

		if b.storedObjects == nil {
			b.storedObjects = &pluginschema.StoredObjectList{}
		}
		b.storedObjects.Items = append(b.storedObjects.Items, obj)
	}
	return nil
}
