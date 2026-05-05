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

func TestExecuteQueryTool_callsHandlerAndEncodesFrames(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	h := &fakeHandler{}
	s.BindQueryDataHandler(h)

	args := map[string]any{
		"owner":      "grafana",
		"repository": "github-datasource",
	}
	out, err := s.executeQueryTool(context.Background(), "Pull_Requests", args)
	require.NoError(t, err)
	require.NotNil(t, out)

	// Handler should have been called with a single DataQuery whose JSON body matches args.
	require.NotNil(t, h.queryCalledWith)
	require.Len(t, h.queryCalledWith.Queries, 1)
	q := h.queryCalledWith.Queries[0]
	assert.Equal(t, "Pull_Requests", q.QueryType)
	assert.Equal(t, "A", q.RefID)
	assert.JSONEq(t, `{"owner":"grafana","repository":"github-datasource"}`, string(q.JSON))
}

func TestExecuteQueryTool_errorsWhenHandlerNotBound(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	_, err := s.executeQueryTool(context.Background(), "Pull_Requests", map[string]any{})
	assert.Error(t, err)
}
