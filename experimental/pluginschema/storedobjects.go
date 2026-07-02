package pluginschema

import (
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// StoredObjectScope is the scope of a stored object kind.
type StoredObjectScope string

const (
	// ScopeNamespaced indicates a namespaced resource. This is the default
	// when StoredObject.Scope is empty.
	ScopeNamespaced StoredObjectScope = "Namespaced"

	// ScopeCluster indicates a cluster-scoped resource.
	ScopeCluster StoredObjectScope = "Cluster"
)

// AdmissionOperation is the name of an admission operation a plugin declares
// it handles for a given stored object.
type AdmissionOperation string

const (
	AdmissionOperationCreate AdmissionOperation = "CREATE"
	AdmissionOperationUpdate AdmissionOperation = "UPDATE"
	AdmissionOperationDelete AdmissionOperation = "DELETE"
)

// StoredObject declares a typed object that the plugin persists. It is the
// counterpart to Settings for runtime-created objects: a plugin can declare
// kinds it owns, and Grafana exposes them as CRUD subresources on the
// per-plugin api-server.
//
// EXPERIMENTAL: minimum-viable shape. Versioning, conversion, RBAC, status
// subresources, and examples are intentionally not declared here. They are
// separable concerns that will be added when the design questions for each
// are settled.
type StoredObject struct {
	// Name is the kind name, e.g. "Watchlist".
	Name string `json:"name"`

	// Plural is the URL plural form, e.g. "watchlists". If empty, derived
	// from Name by lowercasing and appending "s".
	Plural string `json:"plural,omitempty"`

	// Singular is the URL singular form, e.g. "watchlist". If empty, derived
	// from Name by lowercasing.
	Singular string `json:"singular,omitempty"`

	// Scope is "Namespaced" (default) or "Cluster".
	Scope StoredObjectScope `json:"scope,omitempty"`

	// Spec is the schema for the object's spec field.
	Spec *spec.Schema `json:"spec"`

	// Status is the schema for the object's status subresource. When set,
	// Grafana serves a /status subresource for the kind so background
	// processes (e.g. a reconciler in the plugin backend) can report state
	// without contending with user writes to spec.
	Status *spec.Schema `json:"status,omitempty"`

	// Validation, when non-empty, opts the kind into validating admission for
	// the listed operations. Grafana routes those admission decisions to the
	// plugin's backend.AdmissionHandler.ValidateAdmission over gRPC.
	Validation []AdmissionOperation `json:"validation,omitempty"`

	// Mutation, when non-empty, opts the kind into mutating admission for the
	// listed operations. Grafana routes those admission decisions to the
	// plugin's backend.AdmissionHandler.MutateAdmission over gRPC.
	Mutation []AdmissionOperation `json:"mutation,omitempty"`
}

// StoredObjectList carries the declared stored objects in the plugin schema
// artifact.
type StoredObjectList struct {
	Items []StoredObject `json:"items"`
}

// IsZero returns true if the list is empty or nil.
func (s *StoredObjectList) IsZero() bool {
	if s == nil {
		return true
	}
	return len(s.Items) == 0
}
