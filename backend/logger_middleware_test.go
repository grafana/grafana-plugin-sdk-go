package backend_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/handlertest"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const loggerMiddlewareChildEnv = "SDK_TEST_LOGGER_MIDDLEWARE_CHILD"

// The plugin process's default logger is made at package init, so each case runs in a child process.
func TestLoggerMiddlewareLevelFromEnv(t *testing.T) {
	if os.Getenv(loggerMiddlewareChildEnv) == "1" {
		logOneSuccessAndOneFailure(t)
		return
	}

	tests := []struct {
		level string
		want  []string
	}{
		{level: "", want: []string{
			"debug Plugin Request Started", "info Plugin Request Completed",
			"debug Plugin Request Started", "error Plugin Request Completed",
		}},
		{level: "warn", want: []string{"error Plugin Request Completed"}},
	}
	for _, tc := range tests {
		t.Run("level="+tc.level, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestLoggerMiddlewareLevelFromEnv$") // #nosec G204 G702
			cmd.Env = append(os.Environ(), loggerMiddlewareChildEnv+"=1", log.PluginLogLevelEnv+"="+tc.level)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			require.NoError(t, cmd.Run(), stderr.String())

			var got []string
			for line := range bytes.Lines(stderr.Bytes()) {
				var entry struct {
					Level   string `json:"@level"`
					Message string `json:"@message"`
				}
				require.NoError(t, json.Unmarshal(line, &entry), string(line))
				got = append(got, entry.Level+" "+entry.Message)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func logOneSuccessAndOneFailure(t *testing.T) {
	t.Helper()
	var fail bool
	handler, err := backend.HandlerFromMiddlewares(handlertest.Handler{
		QueryDataFunc: func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
			if fail {
				return nil, errors.New("query failed")
			}
			return backend.NewQueryDataResponse(), nil
		},
	}, backend.NewLoggerMiddleware(backend.Logger, nil))
	require.NoError(t, err)

	_, err = handler.QueryData(context.Background(), &backend.QueryDataRequest{})
	require.NoError(t, err)
	fail = true
	_, err = handler.QueryData(context.Background(), &backend.QueryDataRequest{})
	require.Error(t, err)
}
