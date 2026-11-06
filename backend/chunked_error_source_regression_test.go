package backend_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/stretchr/testify/require"
)

func TestChunkedWriterErrorSource(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		source backend.ErrorSource
	}{
		{"downstream", backend.DownstreamError(errors.New("bad query")), backend.ErrorSourceDownstream},
		{"nested downstream", fmt.Errorf("wrapped: %w", backend.DownstreamError(errors.New("bad query"))), backend.ErrorSourceDownstream},
		{"plugin", backend.PluginError(errors.New("plugin failed")), backend.ErrorSourcePlugin},
		{"explicit plugin cancellation", backend.PluginError(context.Canceled), backend.ErrorSourcePlugin},
		{"canceled", context.Canceled, backend.ErrorSourceDownstream},
		{"deadline", context.DeadlineExceeded, backend.ErrorSourceDownstream},
		{"plain", errors.New("unknown"), ""},
		{"nil", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var received *pluginv2.QueryChunkedDataResponse
			writer := backend.NewChunkedDataWriter(backend.DataFrameFormat_JSON, func(chunk *pluginv2.QueryChunkedDataResponse) error { received = chunk; return nil })
			require.NoError(t, writer.WriteError(context.Background(), "A", backend.StatusInternal, tc.err))
			require.NotNil(t, received)
			require.Equal(t, string(tc.source), received.ErrorSource)
			require.Equal(t, "A", received.RefId)
			require.EqualValues(t, 500, received.Status)
			if tc.err != nil {
				require.Equal(t, tc.err.Error(), received.Error)
			} else {
				require.Empty(t, received.Error)
			}
		})
	}
}
