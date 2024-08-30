package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/grafana/grafana-plugin-sdk-go/internal/tenant"
)

type fakeDataHandlerWithOAuth struct {
	cli     *http.Client
	svr     *httptest.Server
	lastReq *http.Request
}

func newFakeDataHandlerWithOAuth() *fakeDataHandlerWithOAuth {
	settings := DataSourceInstanceSettings{}
	opts, err := settings.HTTPClientOptions(context.Background())
	if err != nil {
		panic("http client options: " + err.Error())
	}
	cli, err := httpclient.New(opts)
	if err != nil {
		panic("httpclient new: " + err.Error())
	}

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	return &fakeDataHandlerWithOAuth{
		cli: cli,
		svr: svr,
	}
}

func (f *fakeDataHandlerWithOAuth) QueryData(ctx context.Context, _ *QueryDataRequest) (*QueryDataResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", f.svr.URL, nil)
	if err != nil {
		return nil, err
	}
	f.lastReq = httpReq

	res, err := f.cli.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return &QueryDataResponse{}, nil
}

func TestQueryData(t *testing.T) {
	t.Run("When forward HTTP headers enabled should forward headers", func(t *testing.T) {
		ctx := context.Background()
		handler := newFakeDataHandlerWithOAuth()
		adapter := newDataSDKAdapter(handler, nil)
		_, err := adapter.QueryData(ctx, &pluginv2.QueryDataRequest{
			Headers: map[string]string{
				"Authorization": "Bearer 123",
			},
			PluginContext: &pluginv2.PluginContext{},
		})
		require.NoError(t, err)

		middlewares := httpclient.ContextualMiddlewareFromContext(handler.lastReq.Context())
		require.Len(t, middlewares, 1)

		reqClone := handler.lastReq.Clone(handler.lastReq.Context())
		// clean up headers to be sure they are injected
		reqClone.Header = http.Header{}

		res, err := middlewares[0].CreateMiddleware(httpclient.Options{ForwardHTTPHeaders: true}, finalRoundTripper).RoundTrip(reqClone)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
		require.Len(t, reqClone.Header, 1)
		require.Equal(t, "Bearer 123", reqClone.Header.Get("Authorization"))
	})

	t.Run("When forward HTTP headers disable should not forward headers", func(t *testing.T) {
		ctx := context.Background()
		handler := newFakeDataHandlerWithOAuth()
		adapter := newDataSDKAdapter(handler, nil)
		_, err := adapter.QueryData(ctx, &pluginv2.QueryDataRequest{
			Headers: map[string]string{
				"Authorization": "Bearer 123",
			},
			PluginContext: &pluginv2.PluginContext{},
		})
		require.NoError(t, err)

		middlewares := httpclient.ContextualMiddlewareFromContext(handler.lastReq.Context())
		require.Len(t, middlewares, 1)

		reqClone := handler.lastReq.Clone(handler.lastReq.Context())
		// clean up headers to be sure they are injected
		reqClone.Header = http.Header{}

		res, err := middlewares[0].CreateMiddleware(httpclient.Options{ForwardHTTPHeaders: false}, finalRoundTripper).RoundTrip(reqClone)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
		require.Empty(t, reqClone.Header)
	})

	t.Run("When tenant information is attached to incoming context, it is propagated from adapter to handler", func(t *testing.T) {
		tid := "123456"
		a := newDataSDKAdapter(QueryDataHandlerFunc(func(ctx context.Context, _ *QueryDataRequest) (*QueryDataResponse, error) {
			require.Equal(t, tid, tenant.IDFromContext(ctx))
			return NewQueryDataResponse(), nil
		}), nil)

		ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
			tenant.CtxKey: tid,
		}))
		_, err := a.QueryData(ctx, &pluginv2.QueryDataRequest{
			PluginContext: &pluginv2.PluginContext{},
		})
		require.NoError(t, err)
	})

	t.Run("TestQueryDataResponse", func(t *testing.T) {
		someErr := errors.New("oops")

		for _, tc := range []struct {
			name              string
			queryDataResponse *QueryDataResponse
			expErrorSource    ErrorSource
		}{
			{
				name: `single downstream error should be "downstream" error source`,
				queryDataResponse: &QueryDataResponse{
					Responses: map[string]DataResponse{
						"A": {Error: someErr, ErrorSource: ErrorSourceDownstream},
					},
				},
				expErrorSource: ErrorSourceDownstream,
			},
			{
				name: `single plugin error should be "plugin" error source`,
				queryDataResponse: &QueryDataResponse{
					Responses: map[string]DataResponse{
						"A": {Error: someErr, ErrorSource: ErrorSourcePlugin},
					},
				},
				expErrorSource: ErrorSourcePlugin,
			},
			{
				name: `multiple downstream errors should be "downstream" error source`,
				queryDataResponse: &QueryDataResponse{
					Responses: map[string]DataResponse{
						"A": {Error: someErr, ErrorSource: ErrorSourceDownstream},
						"B": {Error: someErr, ErrorSource: ErrorSourceDownstream},
					},
				},
				expErrorSource: ErrorSourceDownstream,
			},
			{
				name: `single plugin error mixed with downstream errors should be "plugin" error source`,
				queryDataResponse: &QueryDataResponse{
					Responses: map[string]DataResponse{
						"A": {Error: someErr, ErrorSource: ErrorSourceDownstream},
						"B": {Error: someErr, ErrorSource: ErrorSourcePlugin},
						"C": {Error: someErr, ErrorSource: ErrorSourceDownstream},
					},
				},
				expErrorSource: ErrorSourcePlugin,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var actualCtx context.Context
				a := newDataSDKAdapter(QueryDataHandlerFunc(func(ctx context.Context, _ *QueryDataRequest) (*QueryDataResponse, error) {
					actualCtx = ctx
					return tc.queryDataResponse, nil
				}), nil)
				_, err := a.QueryData(context.Background(), &pluginv2.QueryDataRequest{
					PluginContext: &pluginv2.PluginContext{},
				})
				require.NoError(t, err)
				ss := errorSourceFromContext(actualCtx)
				require.Equal(t, tc.expErrorSource, ss)
			})
		}
	})

	t.Run("When conversionHandler is defined", func(t *testing.T) {
		oldQuery := &pluginv2.DataQuery{
			TimeRange: &pluginv2.TimeRange{},
			Json:      []byte(`{"old":"value"}`),
		}
		a := newDataSDKAdapter(QueryDataHandlerFunc(func(_ context.Context, q *QueryDataRequest) (*QueryDataResponse, error) {
			require.Len(t, q.Queries, 1)
			// Assert that the query has been converted
			require.Equal(t, string(`{"new":"value"}`), string(q.Queries[0].JSON))
			return &QueryDataResponse{}, nil
		}), ConvertObjectsFunc(func(_ context.Context, req *ConversionRequest) (*ConversionResponse, error) {
			// Validate that the request is a query
			require.Equal(t, "query", req.TargetVersion.Group)
			require.Equal(t, "v0alpha1", req.TargetVersion.Version)
			require.Len(t, req.Objects, 1)
			// Parse the object and change the JSON
			q := &DataQuery{}
			require.NoError(t, json.Unmarshal(req.Objects[0].Raw, &q))
			require.Equal(t, string(`{"old":"value"}`), string(q.JSON))
			q.JSON = []byte(`{"new":"value"}`)
			b, err := json.Marshal(q)
			require.NoError(t, err)
			return &ConversionResponse{Objects: []RawObject{{Raw: b}}}, nil
		}))
		_, err := a.QueryData(context.Background(), &pluginv2.QueryDataRequest{
			PluginContext: &pluginv2.PluginContext{},
			Queries:       []*pluginv2.DataQuery{oldQuery},
		})
		require.NoError(t, err)
	})
}

var finalRoundTripper = httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Request:    req,
		Body:       io.NopCloser(bytes.NewBufferString("")),
	}, nil
})
