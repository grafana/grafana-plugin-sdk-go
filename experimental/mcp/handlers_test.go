package mcp

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeHandler struct {
	queryCalledWith    *backend.QueryDataRequest
	resourceCalledWith *backend.CallResourceRequest
	healthCalledWith   *backend.CheckHealthRequest
}

func (f *fakeHandler) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	f.queryCalledWith = req
	return &backend.QueryDataResponse{Responses: backend.Responses{"A": backend.DataResponse{}}}, nil
}

func (f *fakeHandler) CallResource(_ context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	f.resourceCalledWith = req
	return sender.Send(&backend.CallResourceResponse{Status: 200, Body: []byte(`{"ok":true}`)})
}

func (f *fakeHandler) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	f.healthCalledWith = req
	return &backend.CheckHealthResult{Status: backend.HealthStatusOk}, nil
}

func TestServer_Bind_storesHandlers(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	h := &fakeHandler{}
	s.BindQueryDataHandler(h)
	s.BindCallResourceHandler(h)
	s.BindCheckHealthHandler(h)
	require.NotNil(t, s.queryDataHandler)
	require.NotNil(t, s.callResourceHandler)
	require.NotNil(t, s.checkHealthHandler)
	// Type assertion via accessors:
	assert.Equal(t, h, s.QueryDataHandler())
}
