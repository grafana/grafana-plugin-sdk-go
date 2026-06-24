package schemabuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// AdmissionEntry declares one stored-object kind for the admission
// dispatcher. The SpecType is the Go type used for the kind's spec body;
// the dispatcher decodes the incoming admission request into a new instance
// of SpecType and looks for optional Validate() and Mutate() methods.
type AdmissionEntry struct {
	// Kind is the stored object's Kind name (e.g. "Watchlist"). Must match
	// what the plugin declares in its schema artifact.
	Kind string

	// SpecType is the Go type for the kind's spec body. The dispatcher
	// creates a new pointer to this type via reflect.New, unmarshals the
	// incoming spec bytes into it, and tests for Validate()/Mutate()
	// interface conformance.
	SpecType reflect.Type
}

// AdmissionHandler returns a backend.AdmissionHandler that dispatches
// incoming admission requests by Kind to typed Validate() / Mutate() methods
// on the registered spec types. The plugin author supplies the spec type
// (and writes Validate/Mutate methods if they want admission to do
// something); the dispatcher handles envelope decoding, interface
// assertions, response marshaling.
//
// Spec types may optionally implement:
//
//	Validate() error            // called during admission validation
//	Mutate() error              // called during admission mutation; modifies receiver in place
//
// Both are optional. If neither is defined, admission is a pass-through.
func AdmissionHandler(entries ...AdmissionEntry) backend.AdmissionHandler {
	handler := &storedObjectAdmission{kinds: make(map[string]reflect.Type, len(entries))}
	for _, e := range entries {
		handler.kinds[e.Kind] = e.SpecType
	}
	return handler
}

// rawEnvelope mirrors the JSON shape Grafana sends as the admission object:
// an apiVersion + Kind + metadata + typed spec. Spec is held as raw bytes
// so the dispatcher can decode it into the per-kind Go type via reflection.
// Metadata is held as RawMessage to avoid pulling kubernetes API types into
// the plugin SDK.
type rawEnvelope struct {
	APIVersion string          `json:"apiVersion,omitempty"`
	Kind       string          `json:"kind,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	Spec       json.RawMessage `json:"spec"`
}

type storedObjectAdmission struct {
	kinds map[string]reflect.Type
}

var _ backend.AdmissionHandler = (*storedObjectAdmission)(nil)

func (h *storedObjectAdmission) ValidateAdmission(_ context.Context, req *backend.AdmissionRequest) (*backend.ValidationResponse, error) {
	specType, ok := h.kinds[req.Kind.Kind]
	if !ok {
		return admissionDenied(fmt.Sprintf("unknown kind %q", req.Kind.Kind)), nil
	}
	spec, _, err := decodeSpec(req.ObjectBytes, specType)
	if err != nil {
		return admissionDenied(fmt.Sprintf("decoding %s: %v", req.Kind.Kind, err)), nil
	}
	if v, ok := spec.(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return admissionDenied(err.Error()), nil
		}
	}
	return &backend.ValidationResponse{Allowed: true}, nil
}

func (h *storedObjectAdmission) MutateAdmission(_ context.Context, req *backend.AdmissionRequest) (*backend.MutationResponse, error) {
	specType, ok := h.kinds[req.Kind.Kind]
	if !ok {
		return mutationDenied(fmt.Sprintf("unknown kind %q", req.Kind.Kind)), nil
	}
	spec, envelope, err := decodeSpec(req.ObjectBytes, specType)
	if err != nil {
		return mutationDenied(fmt.Sprintf("decoding %s: %v", req.Kind.Kind, err)), nil
	}
	// Validate before mutating so a bad input gets rejected with a clear
	// error rather than mutated into a different bad input.
	if v, ok := spec.(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return mutationDenied(err.Error()), nil
		}
	}
	m, hasMutate := spec.(interface{ Mutate() error })
	if !hasMutate {
		return &backend.MutationResponse{Allowed: true}, nil
	}
	if err := m.Mutate(); err != nil {
		return mutationDenied(err.Error()), nil
	}
	mutatedSpec, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshaling mutated %s: %w", req.Kind.Kind, err)
	}
	envelope.Spec = mutatedSpec
	out, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshaling envelope for mutated %s: %w", req.Kind.Kind, err)
	}
	return &backend.MutationResponse{Allowed: true, ObjectBytes: out}, nil
}

// decodeSpec parses raw into the wire envelope and unmarshals the spec
// bytes into a new instance of specType. Returns the spec (as
// interface{} backed by a pointer to specType) and the envelope so callers
// can re-marshal after mutation.
func decodeSpec(raw []byte, specType reflect.Type) (interface{}, *rawEnvelope, error) {
	envelope := &rawEnvelope{}
	if err := json.Unmarshal(raw, envelope); err != nil {
		return nil, nil, err
	}
	specPtr := reflect.New(specType).Interface()
	if len(envelope.Spec) > 0 {
		if err := json.Unmarshal(envelope.Spec, specPtr); err != nil {
			return nil, nil, err
		}
	}
	return specPtr, envelope, nil
}

func admissionDenied(message string) *backend.ValidationResponse {
	return &backend.ValidationResponse{
		Allowed: false,
		Result: &backend.StatusResult{
			Status:  "Failure",
			Message: message,
			Reason:  "Invalid",
			Code:    400,
		},
	}
}

func mutationDenied(message string) *backend.MutationResponse {
	return &backend.MutationResponse{
		Allowed: false,
		Result: &backend.StatusResult{
			Status:  "Failure",
			Message: message,
			Reason:  "Invalid",
			Code:    400,
		},
	}
}
