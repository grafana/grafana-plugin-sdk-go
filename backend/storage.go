package backend

import (
	"context"
	"strconv"
)

// StorageHandler manages how objects before they are sent to storage
type StorageHandler interface {
	MutateInstanceSettings(context.Context, *InstanceSettingsAdmissionRequest) (*InstanceSettingsResponse, error)
	ValidateAdmission(context.Context, *AdmissionRequest) (*StorageResponse, error)
	MutateAdmission(context.Context, *AdmissionRequest) (*StorageResponse, error)
	ConvertObject(context.Context, *ConversionRequest) (*StorageResponse, error)
}

type MutateInstanceSettingsFunc func(context.Context, *InstanceSettingsAdmissionRequest) (*InstanceSettingsResponse, error)
type ValidateAdmissionFunc func(context.Context, *AdmissionRequest) (*StorageResponse, error)
type MutateAdmissionFunc func(context.Context, *AdmissionRequest) (*StorageResponse, error)
type ConvertObjectFunc func(context.Context, *ConversionRequest) (*StorageResponse, error)

// StorageOperation is the the storage operation
type StorageOperation int

const (
	StorageOperationCREATE StorageOperation = iota
	StorageOperationUPDATE
	StorageOperationDELETE
)

var storageOperationNames = map[int]string{
	0: "CREATE",
	1: "UPDATE",
	2: "DELETE",
}

// String textual representation of the operation.
func (hs StorageOperation) String() string {
	s, exists := storageOperationNames[int(hs)]
	if exists {
		return s
	}
	return strconv.Itoa(int(hs))
}

type InstanceSettingsAdmissionRequest struct {
	// NOTE: this may not include app or datasource instance settings depending on the request
	PluginContext PluginContext `json:"pluginContext,omitempty"`
	// The requested operation
	Operation StorageOperation `json:"operation,omitempty"`
	// Requested app instance state (not yet saved)
	AppInstanceSettings *AppInstanceSettings `json:"appInstanceSettings,omitempty"`
	// Requested data source instance state (not yet saved)
	DataSourceInstanceSettings *DataSourceInstanceSettings `json:"dataSourceInstanceSettings,omitempty"`
}

type InstanceSettingsResponse struct {
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
	// Valid app instance settings if they exist in the request
	AppInstanceSettings *AppInstanceSettings `json:"appInstanceSettings,omitempty"`
	// Valid datasource instance settings if they exist in the request
	DataSourceInstanceSettings *DataSourceInstanceSettings `json:"dataSourceInstanceSettings,omitempty"`
}

type AdmissionRequest struct {
	// NOTE: this may not include app or datasource instance settings depending on the request
	PluginContext PluginContext `protobuf:"bytes,1,opt,name=pluginContext,proto3" json:"pluginContext,omitempty"`
	// The requested operation
	Operation StorageOperation `protobuf:"varint,2,opt,name=operation,proto3,enum=pluginv2.StorageOperation" json:"operation,omitempty"`
	// Object is the object in the request.  This includes the full metadata envelope.
	// The Group+Version+Kind will be included in the payload
	ObjectBytes []byte `protobuf:"bytes,3,opt,name=object_bytes,json=objectBytes,proto3" json:"object_bytes,omitempty"`
	// OldObject is the object as it currently exists in storage. This includes the full metadata envelope.
	OldObjectBytes []byte `protobuf:"bytes,4,opt,name=old_object_bytes,json=oldObjectBytes,proto3" json:"old_object_bytes,omitempty"`
}

type ConversionObjectEnvelope int

const (
	ConversionObjectEnvelopeRESOURCE ConversionObjectEnvelope = iota
	ConversionObjectEnvelopeQUERY
)

var conversionNames = map[int]string{
	0: "RESOURCE",
	1: "QUERY",
}

// String textual representation of the operation.
func (hs ConversionObjectEnvelope) String() string {
	s, exists := conversionNames[int(hs)]
	if exists {
		return s
	}
	return strconv.Itoa(int(hs))
}

type ConversionRequest struct {
	// NOTE: this context exists because most of the middleware depends on it
	// The format conversions most likely do not depend on instance settings passed with the context
	PluginContext PluginContext `json:"pluginContext,omitempty"`
	// object metadata wrapper
	Envelope ConversionObjectEnvelope `json:"envelope,omitempty"`
	// Object is the object in the request.  This includes the full metadata envelope.
	ObjectBytes []byte `json:"object_bytes,omitempty"`
	// Target converted version
	TargetVersion string `json:"target_version,omitempty"`
}

type StorageResponse struct {
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
	AuditAnnotations map[string]string `json:"auditAnnotations,omitempty"`
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
