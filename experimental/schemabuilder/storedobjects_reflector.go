package schemabuilder

import (
	"fmt"
	"reflect"
	"strings"

	"k8s.io/kube-openapi/pkg/validation/spec"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/storedobjects"
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

	// StatusType is the Go type representing the object's status
	// subresource. Optional. When set, the type is reflected into the
	// artifact's status schema and Grafana serves a /status subresource
	// for the kind.
	StatusType reflect.Type

	// Validation, when non-empty, opts the kind into validating the listed
	// write operations.
	Validation []pluginschema.Operation

	// Mutation, when non-empty, opts the kind into mutating the listed
	// write operations.
	Mutation []pluginschema.Operation

	// Events, when true, opts the kind into change-event push: Grafana
	// pushes change events for this kind to the plugin over the
	// StoredObjectEvents gRPC stream.
	Events bool
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

		spec, err := b.reflectStoredObjectSchema(info.SpecType)
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
			Events:     info.Events,
		}
		if info.StatusType != nil {
			status, err := b.reflectStoredObjectSchema(info.StatusType)
			if err != nil {
				return fmt.Errorf("stored object %q status: %w", info.Name, err)
			}
			obj.Status = status
		}
		if obj.Plural == "" {
			obj.Plural = storedobjects.PluralOf(info.Name)
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

// reflectStoredObjectSchema reflects a Go type into the OpenAPI schema shape
// stored objects carry in the artifact, applying the same draft-04
// normalization AddQueries enforces so the schema round-trips through the
// OpenAPI loader Grafana uses.
func (b *Builder) reflectStoredObjectSchema(t reflect.Type) (*spec.Schema, error) {
	schema := b.reflector.ReflectFromType(t)
	if schema == nil {
		return nil, fmt.Errorf("reflection returned nil schema")
	}
	updateEnumDescriptions(schema)

	schema.Version = draft04
	schema.ID = ""
	schema.Anchor = ""

	return asJSONSchema(schema)
}
