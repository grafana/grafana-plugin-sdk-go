package backend

import "context"

// QueryConversionHandler is an EXPERIMENTAL service that allows converting queries between versions

type QueryConversionHandler interface {
	// ConvertQuery is called to covert queries between different versions
	ConvertQuery(ctx context.Context, req *QueryDataRequest) (*QueryConversionResponse, error)
}

type ConvertQueryFunc func(ctx context.Context, req *QueryDataRequest) (*QueryConversionResponse, error)

type QueryConversionResponse struct {
	// Converted queries. It should extend v0alpha1.Query
	QueryRequest *QueryDataRequest `json:"query"`
	// Result contains extra details into why an conversion request was denied.
	// +optional
	Result *StatusResult `json:"result,omitempty"`
}
