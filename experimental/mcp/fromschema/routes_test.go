package fromschema_test

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/fromschema"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

type resourceOnly struct{ lastReq *backend.CallResourceRequest }

func (r *resourceOnly) CallResource(_ context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	r.lastReq = req
	return sender.Send(&backend.CallResourceResponse{Status: 200, Body: []byte(`{"ok":true}`)})
}

func TestRegisterRouteTools_addsToolsForResourceRoutes(t *testing.T) {
	schema := &pluginschema.PluginSchema{
		Routes: &pluginschema.Routes{
			Paths: map[string]*spec3.Path{
				"/resources/labels": {
					PathProps: spec3.PathProps{
						Get: &spec3.Operation{
							OperationProps: spec3.OperationProps{
								Summary: "List GitHub labels",
								Parameters: []*spec3.Parameter{
									{ParameterProps: spec3.ParameterProps{Name: "owner", In: "query", Required: true, Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: spec.StringOrArray{"string"}}}}},
									{ParameterProps: spec3.ParameterProps{Name: "repository", In: "query", Required: true, Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: spec.StringOrArray{"string"}}}}},
								},
							},
						},
					},
				},
			},
		},
	}

	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	r := &resourceOnly{}
	s.BindCallResourceHandler(r)
	fromschema.RegisterRouteTools(s, schema)

	tools := s.Tools()
	require.Len(t, tools, 1)
	// tool name strips the /resources/ prefix - this matches the path that
	// CallResourceHandler actually sees at runtime.
	assert.Equal(t, "get_labels", tools[0].Name)
	assert.Equal(t, "List GitHub labels", tools[0].Description)
}

func TestRegisterRouteTools_skipsProxyRoutes(t *testing.T) {
	schema := &pluginschema.PluginSchema{
		Routes: &pluginschema.Routes{
			Paths: map[string]*spec3.Path{
				"/proxy/foo": {PathProps: spec3.PathProps{Get: &spec3.Operation{}}},
			},
		},
	}
	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	fromschema.RegisterRouteTools(s, schema)
	assert.Empty(t, s.Tools())
}

func TestRegisterRouteTools_handlerCallsBoundCallResource(t *testing.T) {
	schema := &pluginschema.PluginSchema{
		Routes: &pluginschema.Routes{
			Paths: map[string]*spec3.Path{
				"/resources/labels": {PathProps: spec3.PathProps{Get: &spec3.Operation{}}},
			},
		},
	}
	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	r := &resourceOnly{}
	s.BindCallResourceHandler(r)
	fromschema.RegisterRouteTools(s, schema)

	_, err := s.Tools()[0].Handler(context.Background(), map[string]any{})
	require.NoError(t, err)
	require.NotNil(t, r.lastReq)
	assert.Equal(t, "GET", r.lastReq.Method)
	// CallResourceHandler sees the path WITHOUT the /resources/ prefix
	// (Grafana strips it before forwarding; we mirror that behavior).
	assert.Equal(t, "/labels", r.lastReq.Path)
}
