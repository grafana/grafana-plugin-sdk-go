package datasource

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartMCPServer_startsAndStops(t *testing.T) {
	s := mcp.NewServer(mcp.ServerOpts{Name: "test", Version: "1.0", Addr: "127.0.0.1:0"})
	require.NoError(t, startMCPServer(s))
	addr := s.ListenAddr()
	require.NotEmpty(t, addr)
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	require.NoError(t, err)
	conn.Close()
	require.NoError(t, stopMCPServer(s))
}

func TestStartMCPServer_logsAndContinuesOnError(t *testing.T) {
	occupier, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer occupier.Close()

	s := mcp.NewServer(mcp.ServerOpts{Name: "test", Version: "1.0", Addr: occupier.Addr().String()})
	// Should not panic and should not fail Manage.
	err = startMCPServer(s)
	assert.NoError(t, err) // we swallow the error and log it
	assert.Empty(t, s.ListenAddr())
}

func TestNewAutomanagementHandler_returnsHandler(t *testing.T) {
	im := NewInstanceManager(func(_ context.Context, _ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
		return struct{}{}, nil
	})
	h := NewAutomanagementHandler(im)
	require.NotNil(t, h)
	// The returned handler implements all four interfaces.
	_, isQuery := any(h).(backend.QueryDataHandler)
	_, isResource := any(h).(backend.CallResourceHandler)
	_, isHealth := any(h).(backend.CheckHealthHandler)
	assert.True(t, isQuery)
	assert.True(t, isResource)
	assert.True(t, isHealth)
}
