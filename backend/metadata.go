package backend

import "context"

type ProvideMetadataHandler interface {
	ProvideMetadata(ctx context.Context, req *ProvideMetadataRequest) (*ProvideMetadataResponse, error)
}

type ProvideMetadataHandlerFunc func(ctx context.Context, req *ProvideMetadataRequest) (*ProvideMetadataResponse, error)

type ProvideMetadataRequest struct {
	PluginContext PluginContext
}

type ProvideMetadataResponse struct {
	Metadata map[string][]string
}
