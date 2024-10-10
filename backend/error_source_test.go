package backend_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestErrorSource(t *testing.T) {
	var s backend.ErrorSource
	require.False(t, s.IsValid())
	require.Equal(t, "plugin", s.String())
	require.True(t, backend.ErrorSourceDownstream.IsValid())
	require.Equal(t, "downstream", backend.ErrorSourceDownstream.String())
	require.True(t, backend.ErrorSourcePlugin.IsValid())
	require.Equal(t, "plugin", backend.ErrorSourcePlugin.String())
}
