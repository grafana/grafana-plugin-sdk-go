package backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"

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
		handlers := Handlers{
			QueryDataHandler: handler,
		}
		handlerWithMw, err := HandlerFromMiddlewares(handlers, newHeaderMiddleware())
		require.NoError(t, err)
		adapter := newDataSDKAdapter(handlerWithMw)
		_, err = adapter.QueryData(ctx, &pluginv2.QueryDataRequest{
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
		handlers := Handlers{
			QueryDataHandler: handler,
		}
		handlerWithMw, err := HandlerFromMiddlewares(handlers, newHeaderMiddleware())
		require.NoError(t, err)
		adapter := newDataSDKAdapter(handlerWithMw)
		_, err = adapter.QueryData(ctx, &pluginv2.QueryDataRequest{
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
		handlers := Handlers{
			QueryDataHandler: QueryDataHandlerFunc(func(ctx context.Context, _ *QueryDataRequest) (*QueryDataResponse, error) {
				require.Equal(t, tid, tenant.IDFromContext(ctx))
				return NewQueryDataResponse(), nil
			}),
		}
		handlerWithMw, err := HandlerFromMiddlewares(handlers, newTenantIDMiddleware())
		require.NoError(t, err)
		a := newDataSDKAdapter(handlerWithMw)

		ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
			tenant.CtxKey: tid,
		}))

		_, err = a.QueryData(ctx, &pluginv2.QueryDataRequest{
			PluginContext: &pluginv2.PluginContext{},
		})
		require.NoError(t, err)
	})

	t.Run("Error source error from QueryData handler will be enriched with grpc status", func(t *testing.T) {
		t.Run("When error is a downstream error", func(t *testing.T) {
			adapter := newDataSDKAdapter(QueryDataHandlerFunc(
				func(_ context.Context, _ *QueryDataRequest) (*QueryDataResponse, error) {
					return nil, DownstreamError(errors.New("oh no"))
				},
			))

			_, err := adapter.QueryData(context.Background(), &pluginv2.QueryDataRequest{
				PluginContext: &pluginv2.PluginContext{},
			})
			require.Error(t, err)

			st := grpcstatus.Convert(err)
			require.NotNil(t, st)
			require.NotEmpty(t, st.Details())
			for _, detail := range st.Details() {
				errorInfo, ok := detail.(*errdetails.ErrorInfo)
				require.True(t, ok)
				require.NotNil(t, errorInfo)
				errorSource, exists := errorInfo.Metadata["errorSource"]
				require.True(t, exists)
				require.Equal(t, ErrorSourceDownstream.String(), errorSource)
			}
		})

		t.Run("When error is a plugin error", func(t *testing.T) {
			adapter := newDataSDKAdapter(QueryDataHandlerFunc(
				func(_ context.Context, _ *QueryDataRequest) (*QueryDataResponse, error) {
					return nil, PluginError(errors.New("oh no"))
				},
			))

			_, err := adapter.QueryData(context.Background(), &pluginv2.QueryDataRequest{
				PluginContext: &pluginv2.PluginContext{},
			})
			require.Error(t, err)

			st := grpcstatus.Convert(err)
			require.NotNil(t, st)
			require.NotEmpty(t, st.Details())
			for _, detail := range st.Details() {
				errorInfo, ok := detail.(*errdetails.ErrorInfo)
				require.True(t, ok)
				require.NotNil(t, errorInfo)
				errorSource, exists := errorInfo.Metadata["errorSource"]
				require.True(t, exists)
				require.Equal(t, ErrorSourcePlugin.String(), errorSource)
			}
		})

		t.Run("When error is neither a downstream or plugin error", func(t *testing.T) {
			adapter := newDataSDKAdapter(QueryDataHandlerFunc(
				func(_ context.Context, _ *QueryDataRequest) (*QueryDataResponse, error) {
					return nil, errors.New("oh no")
				},
			))

			_, err := adapter.QueryData(context.Background(), &pluginv2.QueryDataRequest{
				PluginContext: &pluginv2.PluginContext{},
			})
			require.Error(t, err)

			st := grpcstatus.Convert(err)
			require.NotNil(t, st)
			require.Empty(t, st.Details())
		})
	})
}

func TestErrorSourceFromGrpcStatusError(t *testing.T) {
	type args struct {
		ctx func() context.Context
		err func() error
	}
	type expected struct {
		src   status.Source
		found bool
	}
	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "When error is nil",
			args: args{
				ctx: context.Background,
				err: func() error { return nil },
			},
			expected: expected{
				src:   status.DefaultSource,
				found: false,
			},
		},
		{
			name: "When error is not a grpc status error",
			args: args{
				ctx: context.Background,
				err: func() error {
					return errors.New("oh no")
				},
			},
			expected: expected{
				src:   status.DefaultSource,
				found: false,
			},
		},
		{
			name: "When error is a grpc status error without error details",
			args: args{
				ctx: context.Background,
				err: func() error {
					return grpcstatus.Error(codes.Unknown, "oh no")
				},
			},
			expected: expected{
				src:   status.DefaultSource,
				found: false,
			},
		},
		{
			name: "When error is a grpc status error with error details",
			args: args{
				ctx: context.Background,
				err: func() error {
					st := grpcstatus.New(codes.Unknown, "oh no")
					st, _ = st.WithDetails(&errdetails.ErrorInfo{
						Metadata: map[string]string{
							errorSourceMetadataKey: status.SourcePlugin.String(),
						},
					})
					return st.Err()
				},
			},
			expected: expected{
				src:   status.SourcePlugin,
				found: true,
			},
		},
		{
			name: "When error is a grpc status error with error details, but context already has a source",
			args: args{
				ctx: func() context.Context {
					ctx := status.InitSource(context.Background())
					err := status.WithSource(ctx, status.SourceDownstream)
					require.NoError(t, err)
					return ctx
				},
				err: func() error {
					st := grpcstatus.New(codes.Unknown, "oh no")
					st, _ = st.WithDetails(&errdetails.ErrorInfo{
						Metadata: map[string]string{
							errorSourceMetadataKey: status.SourcePlugin.String(),
						},
					})
					return st.Err()
				},
			},
			expected: expected{
				src:   status.SourcePlugin,
				found: true,
			},
		},
		{
			name: "When error is a grpc status error with error details but no error source",
			args: args{
				ctx: context.Background,
				err: func() error {
					st := grpcstatus.New(codes.Unknown, "oh no")
					st, _ = st.WithDetails(&errdetails.ErrorInfo{
						Metadata: map[string]string{},
					})
					return st.Err()
				},
			},
			expected: expected{
				src:   status.DefaultSource,
				found: false,
			},
		},
		{
			name: "When error is a grpc status error with error details but error source is not a valid source",
			args: args{
				ctx: context.Background,
				err: func() error {
					st := grpcstatus.New(codes.Unknown, "oh no")
					st, _ = st.WithDetails(&errdetails.ErrorInfo{
						Metadata: map[string]string{
							errorSourceMetadataKey: "invalid",
						},
					})
					return st.Err()
				},
			},
			expected: expected{
				src:   status.DefaultSource,
				found: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, ok := ErrorSourceFromGrpcStatusError(tt.args.ctx(), tt.args.err())
			assert.Equal(t, tt.expected.src, src)
			assert.Equal(t, tt.expected.found, ok)
		})
	}
}

var finalRoundTripper = httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Request:    req,
		Body:       io.NopCloser(bytes.NewBufferString("")),
	}, nil
})
