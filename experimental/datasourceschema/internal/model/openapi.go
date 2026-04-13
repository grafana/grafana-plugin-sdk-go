package model

import (
	v0alpha1 "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
)

// DataSourceOpenAPIExtension mirrors the JSON shape Grafana expects for
// datasource schema metadata.
type DataSourceOpenAPIExtension struct {
	// Spec replaces the default datasource spec when provided.
	Spec map[string]any `json:"spec,omitempty"`

	// SecureValues defines which secure keys the datasource uses.
	SecureValues []SecureValueInfo `json:"secureValues"`

	// Examples are added to the POST command payload schema.
	Examples map[string]any `json:"examples,omitempty"`

	// Schemas are additional named schemas referenced by the spec.
	Schemas map[string]any `json:"schemas,omitempty"`

	// Queries defines the supported query types for this datasource.
	Queries *v0alpha1.QueryTypeDefinitionList `json:"queries,omitempty"`

	// Routes describes resource routes exposed under /resource/.
	Routes map[string]any `json:"routes,omitempty"`

	// Proxy describes proxy routes exposed under /proxy/.
	Proxy map[string]any `json:"proxy,omitempty"`
}

type SecureValueInfo struct {
	// String is the secure JSON key.
	String string `json:"string"`

	// Description describes the secure key when known.
	Description string `json:"description,omitempty"`

	// Required indicates whether the secure key must be configured.
	Required bool `json:"required,omitempty"`
}

func NewDataSourceOpenAPIExtension() DataSourceOpenAPIExtension {
	return DataSourceOpenAPIExtension{
		SecureValues: []SecureValueInfo{},
	}
}
