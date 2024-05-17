package backend

import (
	"context"
)

// InstanceSettings handles streams.
type InstanceSettingsHandler interface {
	// ProcessInstanceSettings allows verifying the app/datasource settings before saving
	// This is a specialized form of the validation/mutation hooks that only work for instance settings
	ProcessInstanceSettings(context.Context, *ProcessInstanceSettingsRequest) (*ProcessInstanceSettingsResponse, error)
}

// ProcessInstanceSettingsFunc is an adapter to allow the use of
// ordinary functions as backend.ProcessInstanceSettingsFunc.
type ProcessInstanceSettingsFunc func(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error

// Operation is the type of resource operation being checked for admission control
// https://github.com/kubernetes/kubernetes/blob/v1.30.0/pkg/apis/admission/types.go#L158
type InstanceSettingsOperation int32

const (
	InstanceSettingsOperationCREATE InstanceSettingsOperation = 0
	InstanceSettingsOperationUPDATE InstanceSettingsOperation = 1
	InstanceSettingsOperationDELETE InstanceSettingsOperation = 2
)

type ProcessInstanceSettingsRequest struct {
	PluginContext PluginContext `json:"pluginContext"`
	// Operation is the type of resource operation being checked for admission control
	Operation InstanceSettingsOperation `json:"operation,omitempty"`
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
