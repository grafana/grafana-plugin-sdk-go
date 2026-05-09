package fromschema_test

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/fromschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type healthOnly struct{}

func (healthOnly) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return &backend.CheckHealthResult{Status: backend.HealthStatusOk, Message: "ok"}, nil
}

func TestRegisterHealthCheckTool_addsCheckHealthTool(t *testing.T) {
	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	s.BindCheckHealthHandler(healthOnly{})

	fromschema.RegisterHealthCheckTool(s)

	tools := s.Tools()
	require.Len(t, tools, 1)
	assert.Equal(t, "check_health", tools[0].Name)
}
