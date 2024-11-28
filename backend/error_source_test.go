package backend_test

import (
	"errors"
	"fmt"
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

func TestResponseWithOptions(t *testing.T) {
	for _, tc := range []struct {
		name            string
		err             error
		expErrorMessage string
		expErrorSource  backend.ErrorSource
	}{
		{
			name:            "plugin error",
			err:             backend.PluginError(errors.New("unknown")),
			expErrorMessage: "unknown",
			expErrorSource:  backend.ErrorSourcePlugin,
		},
		{
			name:            "downstream error",
			err:             backend.DownstreamError(errors.New("bad gateway")),
			expErrorMessage: "bad gateway",
			expErrorSource:  backend.ErrorSourceDownstream,
		},
		{
			name:            "wrapped downstream error",
			err:             fmt.Errorf("wrapped: %w", backend.DownstreamError(errors.New("inside error"))),
			expErrorMessage: "wrapped: inside error",
			expErrorSource:  backend.ErrorSourceDownstream,
		},
		{
			name:            "wrapped plugin error",
			err:             fmt.Errorf("wrapped: %w", backend.PluginError(errors.New("inside error"))),
			expErrorMessage: "wrapped: inside error",
			expErrorSource:  backend.ErrorSourcePlugin,
		},
		{
			name:            "non error source error",
			err:             errors.New("inside error"),
			expErrorMessage: "inside error",
			expErrorSource:  "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := backend.ErrorResponseWithErrorSource(tc.err)
			require.Error(t, res.Error)
			require.Equal(t, tc.expErrorMessage, res.Error.Error())
			require.Equal(t, tc.expErrorSource, res.ErrorSource)
		})
	}
}
