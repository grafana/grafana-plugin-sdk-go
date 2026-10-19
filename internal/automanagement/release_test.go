package automanagement

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type releaseSpy struct {
	instance instancemgmt.Instance
	err      error
	releases int
}

func (s *releaseSpy) Get(context.Context, backend.PluginContext) (instancemgmt.Instance, error) {
	return s.instance, s.err
}
func (s *releaseSpy) Do(context.Context, backend.PluginContext, instancemgmt.InstanceCallbackFunc) error {
	panic("unexpected Do call")
}
func (s *releaseSpy) Acquire(context.Context, backend.PluginContext) (instancemgmt.Instance, func(), error) {
	return s.instance, func() { s.releases++ }, s.err
}

type queryFunc func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error)

func (f queryFunc) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return f(ctx, req)
}

func TestQueryReleasesOnErrorAndPanic(t *testing.T) {
	t.Run("handler error", func(t *testing.T) {
		want := errors.New("query failed")
		spy := &releaseSpy{instance: queryFunc(func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
			return nil, want
		})}
		_, err := NewManager(spy).QueryData(context.Background(), &backend.QueryDataRequest{})
		require.ErrorIs(t, err, want)
		require.Equal(t, 1, spy.releases)
	})
	t.Run("handler panic", func(t *testing.T) {
		spy := &releaseSpy{instance: queryFunc(func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
			panic("query panic")
		})}
		require.Panics(t, func() { _, _ = NewManager(spy).QueryData(context.Background(), &backend.QueryDataRequest{}) })
		require.Equal(t, 1, spy.releases)
	})
	t.Run("unsupported handler", func(t *testing.T) {
		spy := &releaseSpy{instance: struct{}{}}
		_, err := NewManager(spy).QueryData(context.Background(), &backend.QueryDataRequest{})
		require.Equal(t, codes.Unimplemented, status.Code(err))
		require.Equal(t, 1, spy.releases)
	})
	t.Run("creation error", func(t *testing.T) {
		want := backend.PluginError(errors.New("creation failed"))
		spy := &releaseSpy{err: want}
		response, err := NewManager(spy).QueryData(context.Background(), &backend.QueryDataRequest{Queries: []backend.DataQuery{{RefID: "A"}}})
		require.NoError(t, err)
		require.ErrorIs(t, response.Responses["A"].Error, want)
		require.Equal(t, 0, spy.releases)
	})
}

func TestLegacyManagerFallback(t *testing.T) {
	spy := &releaseSpy{instance: queryFunc(func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		return backend.NewQueryDataResponse(), nil
	})}
	// Embedding the old interface intentionally hides the optional Acquire method.
	legacy := struct{ instancemgmt.InstanceManager }{spy}
	_, err := NewManager(legacy).QueryData(context.Background(), &backend.QueryDataRequest{})
	require.NoError(t, err)
	require.Equal(t, 0, spy.releases)
}
