package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// AdmissionHandler manages objects before they are sent to storage
type AdmissionHandler interface {
	ValidateAdmission(context.Context, *AdmissionRequest) (*ValidationResponse, error)
	MutateAdmission(context.Context, *AdmissionRequest) (*MutatingResponse, error)
	ConvertObject(context.Context, *ConversionRequest) (*ConversionResponse, error)
}

type ValidateAdmissionFunc func(context.Context, *AdmissionRequest) (*ValidationResponse, error)
type MutateAdmissionFunc func(context.Context, *AdmissionRequest) (*MutatingResponse, error)
type ConvertObjectFunc func(context.Context, *ConversionRequest) (*ConversionResponse, error)

// Operation is the type of resource operation being checked for admission control
// https://github.com/kubernetes/kubernetes/blob/v1.30.0/pkg/apis/admission/types.go#L158
type AdmissionRequestOperation int32

const (
	AdmissionRequestCreate AdmissionRequestOperation = 0
	AdmissionRequestUpdate AdmissionRequestOperation = 1
	AdmissionRequestDelete AdmissionRequestOperation = 2
)

// String textual representation of the operation.
func (o AdmissionRequestOperation) String() string {
	return pluginv2.AdmissionRequest_Operation(o).String()
}

// Identify the Object properties
type GroupVersionKind struct {
	Group   string `json:"group,omitempty"`
	Version string `json:"version,omitempty"`
	Kind    string `json:"kind,omitempty"`
}

// AdmissionRequest contains information from a kubernetes Admission request and decoded object(s).
// See: https://github.com/kubernetes/kubernetes/blob/v1.30.0/pkg/apis/admission/types.go#L41
// And: https://github.com/grafana/grafana-app-sdk/blob/main/resource/admission.go#L14
type AdmissionRequest struct {
	// NOTE: this may not include app or datasource instance settings depending on the request
	PluginContext PluginContext `json:"pluginContext,omitempty"`
	// The requested operation
	Operation AdmissionRequestOperation `json:"operation,omitempty"`
	// The object kind
	Kind GroupVersionKind `json:"kind,omitempty"`
	// Object is the object in the request.  This includes the full metadata envelope.
	ObjectBytes []byte `json:"object_bytes,omitempty"`
	// OldObject is the object as it currently exists in storage. This includes the full metadata envelope.
	OldObjectBytes []byte `json:"old_object_bytes,omitempty"`
}

// ConversionRequest supports converting an object from on version to another
type ConversionRequest struct {
	// NOTE: this may not include app or datasource instance settings depending on the request
	PluginContext PluginContext `json:"pluginContext,omitempty"`
	// The object kind
	Kind GroupVersionKind `json:"kind,omitempty"`
	// Object is the object in the request.  This includes the full metadata envelope.
	ObjectBytes []byte `json:"object_bytes,omitempty"`
	// Target converted version
	TargetVersion string `json:"target_version,omitempty"`
}

// Basic request to say if the validation was successful or not
type ValidationResponse struct {
	// Allowed indicates whether or not the admission request was permitted.
	Allowed bool `json:"allowed"`
}

type MutatingResponse struct {
	// Allowed indicates whether or not the admission request was permitted.
	Allowed bool `json:"allowed,omitempty"`
	// Result contains extra details into why an admission request was denied.
	// This field IS NOT consulted in any way if "Allowed" is "true".
	// +optional
	Result *StatusResult `json:"result,omitempty"`
	// AuditAnnotations is an unstructured key value map set by remote admission controller (e.g. error=image-blacklisted).
	// MutatingAdmissionWebhook and ValidatingAdmissionWebhook admission controller will prefix the keys with
	// admission webhook name (e.g. imagepolicy.example.com/error=image-blacklisted). AuditAnnotations will be provided by
	// the admission webhook to add additional context to the audit log for this request.
	// +optional
	AuditAnnotations map[string]string `json:"auditAnnotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// warnings is a list of warning messages to return to the requesting API client.
	// Warning messages describe a problem the client making the API request should correct or be aware of.
	// Limit warnings to 120 characters if possible.
	// Warnings over 256 characters and large numbers of warnings may be truncated.
	// +optional
	Warnings []string `json:"warnings,omitempty"`
	// Mutated object bytes (when requested)
	// +optional
	ObjectBytes []byte `json:"object_bytes,omitempty"`
}

type ConversionResponse struct {
	// Allowed indicates whether or not the admission request was permitted.
	Allowed bool `json:"allowed,omitempty"`
	// Result contains extra details into why an admission request was denied.
	// This field IS NOT consulted in any way if "Allowed" is "true".
	// +optional
	Result *StatusResult `json:"result,omitempty"`
	// warnings is a list of warning messages to return to the requesting API client.
	// Warning messages describe a problem the client making the API request should correct or be aware of.
	// Limit warnings to 120 characters if possible.
	// Warnings over 256 characters and large numbers of warnings may be truncated.
	// +optional
	Warnings []string `json:"warnings,omitempty"`
	// Mutated object bytes
	ObjectBytes []byte `json:"object_bytes,omitempty"`
}

type StatusResult struct {
	// Status of the operation.
	// One of: "Success" or "Failure".
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Status string `json:"status,omitempty"`
	// A human-readable description of the status of this operation.
	// +optional
	Message string `json:"message,omitempty"`
	// A machine-readable description of why this operation is in the
	// "Failure" status. If this value is empty there
	// is no information available. A Reason clarifies an HTTP status
	// code but does not override it.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Suggested HTTP return code for this status, 0 if not set.
	// +optional
	Code int32 `json:"code,omitempty"`
}
