package backend

import (
	"context"
)

// InstanceSettings handles instance settings storage.
type InstanceSettingsHandler interface {
	CreateInstanceSettings(context.Context, *CreateInstanceSettingsRequest) (*InstanceSettingsResponse, error)
	UpdateInstanceSettings(context.Context, *UpdateInstanceSettingsRequest) (*InstanceSettingsResponse, error)
}

type CreateInstanceSettingsFunc func(context.Context, *CreateInstanceSettingsRequest) (*InstanceSettingsResponse, error)
type UpdateInstanceSettingsFunc func(context.Context, *UpdateInstanceSettingsRequest) (*InstanceSettingsResponse, error)

type CreateInstanceSettingsRequest struct {
	// The unique identifier of the plugin the request is targeted for.
	PluginID string `json:"pluginId,omitempty"`
	// Requested app instance state (not yet saved)
	AppInstanceSettings *AppInstanceSettings `json:"appInstanceSettings,omitempty"`
	// Requested data source instance state (not yet saved)
	DataSourceInstanceSettings *DataSourceInstanceSettings `json:"dataSourceInstanceSettings,omitempty"`
}

type UpdateInstanceSettingsRequest struct {
	// The currently saved properties
	PluginContext PluginContext `json:"pluginContext,omitempty"`
	// Requested new sate of the app plugins settings
	AppInstanceSettings *AppInstanceSettings `json:"appInstanceSettings,omitempty"`
	// Requested new sate of the datasource plugins settings
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
