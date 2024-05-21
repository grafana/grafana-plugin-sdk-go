package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// AdmissionHandler manages how objects before they are sent to storage
type AdmissionHandler interface {
	ValidateAdmission(context.Context, *AdmissionRequest) (*AdmissionResponse, error)
	MutateAdmission(context.Context, *AdmissionRequest) (*AdmissionResponse, error)
	ConvertObject(context.Context, *ConversionRequest) (*AdmissionResponse, error)
}

type ValidateAdmissionFunc func(context.Context, *AdmissionRequest) (*AdmissionResponse, error)
type MutateAdmissionFunc func(context.Context, *AdmissionRequest) (*AdmissionResponse, error)
type ConvertObjectFunc func(context.Context, *ConversionRequest) (*AdmissionResponse, error)

// Operation is the type of resource operation being checked for admission control
// https://github.com/kubernetes/kubernetes/blob/v1.30.0/pkg/apis/admission/types.go#L158
type AdmissionRequest_Operation int32

const (
	AdmissionRequest_CREATE AdmissionRequest_Operation = 0
	AdmissionRequest_UPDATE AdmissionRequest_Operation = 1
	AdmissionRequest_DELETE AdmissionRequest_Operation = 2
)

// String textual representation of the operation.
func (o AdmissionRequest_Operation) String() string {
	return pluginv2.AdmissionRequest_Operation(o).String()
}

// Identify the Object properties
// EG, the union of: metav1.GroupVersionKind and metav1.GroupVersionResource
type GroupVersionKindResource struct {
	Group    string `protobuf:"bytes,1,opt,name=group,proto3" json:"group,omitempty"`
	Version  string `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
	Kind     string `protobuf:"bytes,3,opt,name=kind,proto3" json:"kind,omitempty"`
	Resource string `protobuf:"bytes,4,opt,name=resource,proto3" json:"resource,omitempty"`
}

// AdmissionRequest contains information from a kubernetes Admission request and decoded object(s).
// See: https://github.com/kubernetes/kubernetes/blob/v1.30.0/pkg/apis/admission/types.go#L41
// And: https://github.com/grafana/grafana-app-sdk/blob/main/resource/admission.go#L14
// NOTE: this does not include a plugin context
type AdmissionRequest struct {
	// NOTE: this may not include app or datasource instance settings depending on the request
	PluginContext PluginContext `protobuf:"bytes,1,opt,name=pluginContext,proto3" json:"pluginContext,omitempty"`
	// The requested operation
	Operation AdmissionRequest_Operation `protobuf:"varint,2,opt,name=operation,proto3,enum=pluginv2.AdmissionRequest_Operation" json:"operation,omitempty"`
	// The object kind
	Kind *GroupVersionKindResource `protobuf:"bytes,3,opt,name=kind,proto3" json:"kind,omitempty"`
	// Object is the object in the request.  This includes the full metadata envelope.
	ObjectBytes []byte `protobuf:"bytes,4,opt,name=object_bytes,json=objectBytes,proto3" json:"object_bytes,omitempty"`
	// OldObject is the object as it currently exists in storage. This includes the full metadata envelope.
	OldObjectBytes []byte `protobuf:"bytes,5,opt,name=old_object_bytes,json=oldObjectBytes,proto3" json:"old_object_bytes,omitempty"`
}

// ConversionRequest supports converting an object from on version to another
type ConversionRequest struct {
	// NOTE: this may not include app or datasource instance settings depending on the request
	PluginContext PluginContext `protobuf:"bytes,1,opt,name=pluginContext,proto3" json:"pluginContext,omitempty"`
	// The object kind
	Kind *GroupVersionKindResource `protobuf:"bytes,2,opt,name=kind,proto3" json:"kind,omitempty"`
	// Object is the object in the request.  This includes the full metadata envelope.
	ObjectBytes []byte `protobuf:"bytes,3,opt,name=object_bytes,json=objectBytes,proto3" json:"object_bytes,omitempty"`
	// Target converted version
	TargetVersion string `protobuf:"bytes,4,opt,name=target_version,json=targetVersion,proto3" json:"target_version,omitempty"`
}

// See https://github.com/kubernetes/kubernetes/blob/v1.30.0/pkg/apis/admission/types.go#L118
type AdmissionResponse struct {
	// Allowed indicates whether or not the admission request was permitted.
	Allowed bool `protobuf:"varint,1,opt,name=allowed,proto3" json:"allowed,omitempty"`
	// Result contains extra details into why an admission request was denied.
	// This field IS NOT consulted in any way if "Allowed" is "true".
	// +optional
	Result *StatusResult `protobuf:"bytes,2,opt,name=result,proto3" json:"result,omitempty"`
	// AuditAnnotations is an unstructured key value map set by remote admission controller (e.g. error=image-blacklisted).
	// MutatingAdmissionWebhook and ValidatingAdmissionWebhook admission controller will prefix the keys with
	// admission webhook name (e.g. imagepolicy.example.com/error=image-blacklisted). AuditAnnotations will be provided by
	// the admission webhook to add additional context to the audit log for this request.
	// +optional
	AuditAnnotations map[string]string `protobuf:"bytes,3,rep,name=auditAnnotations,proto3" json:"auditAnnotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// warnings is a list of warning messages to return to the requesting API client.
	// Warning messages describe a problem the client making the API request should correct or be aware of.
	// Limit warnings to 120 characters if possible.
	// Warnings over 256 characters and large numbers of warnings may be truncated.
	// +optional
	Warnings []string `protobuf:"bytes,4,rep,name=warnings,proto3" json:"warnings,omitempty"`
	// Mutated object bytes (when requested)
	// +optional
	ObjectBytes []byte `protobuf:"bytes,5,opt,name=object_bytes,json=objectBytes,proto3" json:"object_bytes,omitempty"`
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
