package datasource_test

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
)

func ExampleQueryTypeMux() {
	mux := datasource.NewQueryTypeMux()
	mux.HandleFunc("queryTypeA", func(_ context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		// handle queryTypeA
		return nil, nil
	})
	mux.HandleFunc("queryTypeB", func(_ context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		// handle queryTypeB
		return nil, nil
	})

	_ = datasource.ServeOpts{
		QueryDataHandler: mux,
	}
}
