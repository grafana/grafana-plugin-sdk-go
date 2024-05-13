package backend

import (
	"context"
)

// StreamHandler handles streams.
type AdmissionHandler interface {
	// ProcessInstanceSettings allows verifying the app/datasource settings before saving
	// This is a specialized form of the validation/mutation hooks that only work for instance settings
	ProcessInstanceSettings(context.Context, *ProcessInstanceSettingsRequest) (*ProcessInstanceSettingsResponse, error)
	// ValidateAdmission validates a request payload to see if it can be admitted
	ValidateAdmission(context.Context, *AdmissionRequest) (*AdmissionResponse, error)
	// MutateAdmission Verify if the input request can be admitted, and return a copy that can be saved
	MutateAdmission(context.Context, *AdmissionRequest) (*AdmissionResponse, error)
}

type ProcessInstanceSettingsRequest struct {
	PluginContext PluginContext `json:"pluginContext"`
	// When configured, this will return results in the target APIVersion format
	TargetAPIVersion string `json:"targetApiVersion"`
	// In addition to checking the payload, also check if any connection
	// parameters are successful
	CheckHealth bool `json:"checkHealth"`
}

type ProcessInstanceSettingsResponse struct {
	// Allowed indicates whether or not the admission request was permitted.
	Allowed bool `json:"allowed"`
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
	// The health response if requested
	Health *CheckHealthResult `json:"health,omitempty"`
}

// https://github.com/kubernetes/kubernetes/blob/v1.30.0/pkg/apis/admission/types.go#L158
// +enum
type AdmissionOperation int

const (
	AdmissionOperationCreate AdmissionOperation = iota
	AdmissionOperationUpdate
	AdmissionOperationDelete
	AdmissionOperationConnect
)

// AdmissionRequest contains information from a kubernetes Admission request and decoded object(s).
// NOTE: this does not (yet?) include the PluginContext
// See: https://github.com/kubernetes/kubernetes/blob/v1.30.0/pkg/apis/admission/types.go#L41
// And: https://github.com/grafana/grafana-app-sdk/blob/main/resource/admission.go#L14
type AdmissionRequest struct {
	// Operation is the type of resource operation being checked for admission control
	Operation AdmissionOperation `json:"operation,omitempty"`
	// Group is the object's group
	Group string `json:"group,omitempty"`
	// Version is the object's api version
	Version string `json:"version,omitempty"`
	// Resource is the object's resource type
	Resource string `json:"resource,omitempty"`
	// UserInfo is user information about the user making the request
	UserInfo *AdmissionUserInfo `json:"userInfo,omitempty"`
	// Object is the object in the request.  This includes the full metadata envelope.
	ObjectBytes []byte `json:"object_bytes,omitempty"`
	// OldObject is the object as it currently exists in storage. This includes the full metadata envelope.
	OldObjectBytes []byte `json:"old_object_bytes,omitempty"`
}

// See https://github.com/kubernetes/kubernetes/blob/v1.30.0/pkg/apis/admission/types.go#L118
type AdmissionResponse struct {
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

// AdmissionUserInfo contains user information for an admission request
type AdmissionUserInfo struct {
	// Username is the username of the user
	Username string `json:"username,omitempty"`
	// UID is the UID of the user in the API server's system
	UID string `json:"uid,omitempty"`
	// Groups is a list of all groups the user is a part of (if any)
	Groups []string `json:"groups,omitempty"`
	// Extra is a map of extra information, implementation-specific
	JSONExtra []byte `json:"jsonExtra,omitempty"`
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
