package mcp_test

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	sdkapi "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/fromschema"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDS struct{ qReq *backend.QueryDataRequest }

func (s *stubDS) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	s.qReq = req
	return &backend.QueryDataResponse{}, nil
}
func (s *stubDS) CallResource(_ context.Context, _ *backend.CallResourceRequest, _ backend.CallResourceResponseSender) error {
	return nil
}
func (s *stubDS) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return &backend.CheckHealthResult{Status: backend.HealthStatusOk}, nil
}

func TestEndToEnd_registerStartCallTool(t *testing.T) {
	// build a minimal schema with one query type and unmarshal a JSON Schema
	// the proper way (the JSONSchema type stores *spec.Schema internally).
	var js sdkapi.JSONSchema
	require.NoError(t, json.Unmarshal([]byte(`{"type":"object"}`), &js))

	schema := &pluginschema.PluginSchema{
		QueryTypes: &sdkapi.QueryTypeDefinitionList{
			Items: []sdkapi.QueryTypeDefinition{{
				ObjectMeta: sdkapi.ObjectMeta{Name: "Pull_Requests"},
				Spec: sdkapi.QueryTypeDefinitionSpec{
					Discriminators: []sdkapi.DiscriminatorFieldValue{{Field: "queryType", Value: "Pull_Requests"}},
					Schema:         js,
				},
			}},
		},
	}

	srv := mcp.NewServer(mcp.ServerOpts{Name: "test-ds", Version: "1.0", Addr: "127.0.0.1:0"})
	srv.RegisterPluginContext("test-uid", backend.PluginContext{
		DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "test-uid"},
	})
	ds := &stubDS{}
	srv.BindQueryDataHandler(ds)
	srv.BindCallResourceHandler(ds)
	srv.BindCheckHealthHandler(ds)

	fromschema.RegisterQueryTools(srv, schema)
	fromschema.RegisterHealthCheckTool(srv)

	require.NoError(t, srv.Start(context.Background()))
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	addr := srv.ListenAddr()
	require.NotEmpty(t, addr)

	// verify the listener is actually accepting TCP connections - that's a
	// sufficient Phase 1 smoke check that registration + Start wired the HTTP
	// transport up correctly. Deeper protocol-level coverage already lives in
	// server_test.go via the in-process mcptest client.
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	// confirm both registered tools are visible on the server.
	tools := srv.Tools()
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	assert.Contains(t, names, "query_Pull_Requests")
	assert.Contains(t, names, "check_health")

	// invoke the registered query tool directly to confirm the bound handler
	// is reached end-to-end through the registration path.
	var queryTool mcp.Tool
	for _, tool := range tools {
		if tool.Name == "query_Pull_Requests" {
			queryTool = tool
			break
		}
	}
	require.NotNil(t, queryTool.Handler)
	_, err = queryTool.Handler(context.Background(), map[string]any{})
	require.NoError(t, err)
	require.NotNil(t, ds.qReq, "bound QueryData handler should have been invoked")
}
