package datasource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestQueryTypeMux(t *testing.T) {
	mux := NewQueryTypeMux()
	aHandlerCalled := false
	mux.HandleFunc("a", func(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		aHandlerCalled = true
		return &backend.QueryDataResponse{
			Responses: backend.Responses{
				"A": backend.DataResponse{},
			},
		}, nil
	})
	bHandlerCalled := false
	mux.Handle("b", QueryTypeHandlerFunc(func(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		bHandlerCalled = true
		return &backend.QueryDataResponse{
			Responses: backend.Responses{
				"B": backend.DataResponse{},
			},
		}, nil
	}))

	res, err := mux.QueryData(context.Background(), &backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			backend.DataQuery{
				RefID:     "A",
				QueryType: "a",
			},
			backend.DataQuery{
				RefID:     "B",
				QueryType: "b",
			},
		},
	})

	require.NoError(t, err)
	require.True(t, aHandlerCalled)
	require.True(t, bHandlerCalled)
	require.Len(t, res.Responses, 2)
}
