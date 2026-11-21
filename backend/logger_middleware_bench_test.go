package backend_test

import (
	"context"
	"io"
	"testing"

	hclog "github.com/hashicorp/go-hclog"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/handlertest"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// Set GF_PLUGIN_LOG_LEVEL to compare levels. The lines go to io.Discard, so this measures their encoding only.
func BenchmarkLoggerMiddlewareQueryData(b *testing.B) {
	output := hclog.DefaultOutput
	hclog.DefaultOutput = io.Discard
	b.Cleanup(func() { hclog.DefaultOutput = output })

	handler, err := backend.HandlerFromMiddlewares(handlertest.Handler{
		QueryDataFunc: func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
			return backend.NewQueryDataResponse(), nil
		},
	}, backend.NewLoggerMiddleware(log.New(), nil))
	if err != nil {
		b.Fatal(err)
	}
	ctx := log.WithContextualAttributes(context.Background(), []any{
		"pluginId", "example-datasource", "pluginVersion", "1.2.3", "endpoint", "queryData",
		"traceId", "4bf92f3577b34da6a3ce929d0e0e4736", "dsName", "example", "dsUid", "a1b2c3d4e5f6g7", "uname", "user@example.com",
	})
	req := &backend.QueryDataRequest{}

	for b.Loop() {
		if _, err := handler.QueryData(ctx, req); err != nil {
			b.Fatal(err)
		}
	}
}
