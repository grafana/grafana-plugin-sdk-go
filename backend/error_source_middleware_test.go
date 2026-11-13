package backend_test

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/handlertest"
	"github.com/stretchr/testify/require"
)

func TestErrorSourceMiddleware(t *testing.T) {
	someErr := errors.New("oops")
	downstreamErr := backend.DownstreamError(someErr)

	t.Run("Handlers return errors", func(t *testing.T) {
		for _, tc := range []struct {
			name           string
			err            error
			expErrorSource backend.ErrorSource
		}{
			{
				name:           `no downstream error`,
				err:            someErr,
				expErrorSource: backend.ErrorSourcePlugin,
			},
			{
				name:           `downstream error`,
				err:            downstreamErr,
				expErrorSource: backend.ErrorSourceDownstream,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				cdt := handlertest.NewHandlerMiddlewareTest(t,
					handlertest.WithMiddlewares(
						backend.NewErrorSourceMiddleware(),
					),
				)
				setupHandlersWithError(cdt, tc.err)

				_, err := cdt.MiddlewareHandler.QueryData(context.Background(), &backend.QueryDataRequest{})
				require.Error(t, err)
				ss := backend.ErrorSourceFromContext(cdt.QueryDataCtx)
				require.Equal(t, tc.expErrorSource, ss)

				_, err = cdt.MiddlewareHandler.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
				require.Error(t, err)
				ss = backend.ErrorSourceFromContext(cdt.CheckHealthCtx)
				require.Equal(t, tc.expErrorSource, ss)

				err = cdt.MiddlewareHandler.CallResource(context.Background(), &backend.CallResourceRequest{}, backend.CallResourceResponseSenderFunc(func(_ *backend.CallResourceResponse) error { return nil }))
				require.Error(t, err)
				ss = backend.ErrorSourceFromContext(cdt.CallResourceCtx)
				require.Equal(t, tc.expErrorSource, ss)

				_, err = cdt.MiddlewareHandler.CollectMetrics(context.Background(), &backend.CollectMetricsRequest{})
				require.Error(t, err)
				ss = backend.ErrorSourceFromContext(cdt.CollectMetricsCtx)
				require.Equal(t, tc.expErrorSource, ss)

				_, err = cdt.MiddlewareHandler.SubscribeStream(context.Background(), &backend.SubscribeStreamRequest{})
				require.Error(t, err)
				ss = backend.ErrorSourceFromContext(cdt.SubscribeStreamCtx)
				require.Equal(t, tc.expErrorSource, ss)

				err = cdt.MiddlewareHandler.RunStream(context.Background(), &backend.RunStreamRequest{}, backend.NewStreamSender(nil))
				require.Error(t, err)
				ss = backend.ErrorSourceFromContext(cdt.RunStreamCtx)
				require.Equal(t, tc.expErrorSource, ss)

				_, err = cdt.MiddlewareHandler.PublishStream(context.Background(), &backend.PublishStreamRequest{})
				require.Error(t, err)
				ss = backend.ErrorSourceFromContext(cdt.PublishStreamCtx)
				require.Equal(t, tc.expErrorSource, ss)

				_, err = cdt.MiddlewareHandler.ValidateAdmission(context.Background(), &backend.AdmissionRequest{})
				require.Error(t, err)
				ss = backend.ErrorSourceFromContext(cdt.ValidateAdmissionCtx)
				require.Equal(t, tc.expErrorSource, ss)

				_, err = cdt.MiddlewareHandler.MutateAdmission(context.Background(), &backend.AdmissionRequest{})
				require.Error(t, err)
				ss = backend.ErrorSourceFromContext(cdt.MutateAdmissionCtx)
				require.Equal(t, tc.expErrorSource, ss)

				_, err = cdt.MiddlewareHandler.ConvertObjects(context.Background(), &backend.ConversionRequest{})
				require.Error(t, err)
				ss = backend.ErrorSourceFromContext(cdt.ConvertObjectCtx)
				require.Equal(t, tc.expErrorSource, ss)
			})
		}
	})

	t.Run("QueryData response with errors", func(t *testing.T) {
		for _, tc := range []struct {
			name              string
			queryDataResponse *backend.QueryDataResponse
			expErrorSource    backend.ErrorSource
		}{
			{
				name:              `no error should be "plugin" error source`,
				queryDataResponse: nil,
				expErrorSource:    backend.ErrorSourcePlugin,
			},
			{
				name: `single downstream error should be "downstream" error source`,
				queryDataResponse: &backend.QueryDataResponse{
					Responses: map[string]backend.DataResponse{
						"A": {Error: someErr, ErrorSource: backend.ErrorSourceDownstream},
					},
				},
				expErrorSource: backend.ErrorSourceDownstream,
			},
			{
				name: `single plugin error should be "plugin" error source`,
				queryDataResponse: &backend.QueryDataResponse{
					Responses: map[string]backend.DataResponse{
						"A": {Error: someErr, ErrorSource: backend.ErrorSourcePlugin},
					},
				},
				expErrorSource: backend.ErrorSourcePlugin,
			},
			{
				name: `multiple downstream errors should be "downstream" error source`,
				queryDataResponse: &backend.QueryDataResponse{
					Responses: map[string]backend.DataResponse{
						"A": {Error: someErr, ErrorSource: backend.ErrorSourceDownstream},
						"B": {Error: someErr, ErrorSource: backend.ErrorSourceDownstream},
					},
				},
				expErrorSource: backend.ErrorSourceDownstream,
			},
			{
				name: `single plugin error mixed with downstream errors should be "plugin" error source`,
				queryDataResponse: &backend.QueryDataResponse{
					Responses: map[string]backend.DataResponse{
						"A": {Error: someErr, ErrorSource: backend.ErrorSourceDownstream},
						"B": {Error: someErr, ErrorSource: backend.ErrorSourcePlugin},
						"C": {Error: someErr, ErrorSource: backend.ErrorSourceDownstream},
					},
				},
				expErrorSource: backend.ErrorSourcePlugin,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				cdt := handlertest.NewHandlerMiddlewareTest(t,
					handlertest.WithMiddlewares(
						backend.NewErrorSourceMiddleware(),
					),
				)
				cdt.TestHandler.QueryDataFunc = func(ctx context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
					cdt.QueryDataCtx = ctx
					return tc.queryDataResponse, nil
				}

				_, _ = cdt.MiddlewareHandler.QueryData(context.Background(), &backend.QueryDataRequest{})

				ss := backend.ErrorSourceFromContext(cdt.QueryDataCtx)
				require.Equal(t, tc.expErrorSource, ss)
			})
		}
	})

	t.Run("QueryData response without valid error source error should set error source", func(t *testing.T) {
		cdt := handlertest.NewHandlerMiddlewareTest(t,
			handlertest.WithMiddlewares(
				backend.NewErrorSourceMiddleware(),
			),
		)

		cdt.TestHandler.QueryDataFunc = func(ctx context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
			cdt.QueryDataCtx = ctx
			return &backend.QueryDataResponse{
				Responses: map[string]backend.DataResponse{
					"A": {Error: someErr},
					"B": {Error: downstreamErr},
				},
			}, nil
		}

		resp, _ := cdt.MiddlewareHandler.QueryData(context.Background(), &backend.QueryDataRequest{})
		require.Equal(t, backend.ErrorSourcePlugin, resp.Responses["A"].ErrorSource)
		require.Equal(t, backend.ErrorSourceDownstream, resp.Responses["B"].ErrorSource)
	})

	t.Run("CheckHealth result sets error source", func(t *testing.T) {
		for _, tc := range []struct {
			name           string
			status         backend.HealthStatus
			responseSource backend.ErrorSource
			markDownstream bool
			expectedWire   backend.ErrorSource
			expectedCtx    backend.ErrorSource
		}{
			{
				name:           "explicit downstream source",
				status:         backend.HealthStatusError,
				responseSource: backend.ErrorSourceDownstream,
				expectedWire:   backend.ErrorSourceDownstream,
				expectedCtx:    backend.ErrorSourceDownstream,
			},
			{
				name:           "explicit plugin source overrides downstream marked on ctx",
				status:         backend.HealthStatusError,
				responseSource: backend.ErrorSourcePlugin,
				markDownstream: true,
				expectedWire:   backend.ErrorSourcePlugin,
				expectedCtx:    backend.ErrorSourcePlugin,
			},
			{
				name:         "untagged error result stays empty on the wire",
				status:       backend.HealthStatusError,
				expectedWire: "",
				expectedCtx:  backend.ErrorSourcePlugin,
			},
			{
				name:           "untagged error result takes downstream from ctx",
				status:         backend.HealthStatusError,
				markDownstream: true,
				expectedWire:   backend.ErrorSourceDownstream,
				expectedCtx:    backend.ErrorSourceDownstream,
			},
			{
				name:           "unknown result is not treated as an error",
				status:         backend.HealthStatusUnknown,
				responseSource: backend.ErrorSourceDownstream,
				expectedWire:   backend.ErrorSourceDownstream,
				expectedCtx:    backend.ErrorSourcePlugin,
			},
			{
				name:           "ok result is not treated as an error",
				status:         backend.HealthStatusOk,
				responseSource: backend.ErrorSourceDownstream,
				expectedWire:   backend.ErrorSourceDownstream,
				expectedCtx:    backend.ErrorSourcePlugin,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				cdt := handlertest.NewHandlerMiddlewareTest(t,
					handlertest.WithMiddlewares(backend.NewErrorSourceMiddleware()),
				)
				cdt.TestHandler.CheckHealthFunc = func(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
					cdt.CheckHealthCtx = ctx
					if tc.markDownstream {
						require.NoError(t, backend.WithDownstreamErrorSource(ctx))
					}
					return &backend.CheckHealthResult{
						Status:      tc.status,
						ErrorSource: tc.responseSource,
					}, nil
				}

				resp, err := cdt.MiddlewareHandler.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
				require.NoError(t, err)
				require.Equal(t, tc.expectedWire, resp.ErrorSource)
				require.Equal(t, tc.expectedCtx, backend.ErrorSourceFromContext(cdt.CheckHealthCtx))
			})
		}
	})

	t.Run("CheckHealth untagged error result does not fail without an initialized error source", func(t *testing.T) {
		mw := backend.NewErrorSourceMiddleware().CreateHandlerMiddleware(&handlertest.Handler{
			CheckHealthFunc: func(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
				return &backend.CheckHealthResult{Status: backend.HealthStatusError}, nil
			},
		})

		resp, err := mw.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
		require.NoError(t, err)
		require.Equal(t, backend.ErrorSource(""), resp.ErrorSource)
	})

	t.Run("CallResource response sets error source", func(t *testing.T) {
		for _, tc := range []struct {
			name           string
			status         int
			responseSource backend.ErrorSource
			markDownstream bool
			expectedWire   backend.ErrorSource
			expectedCtx    backend.ErrorSource
		}{
			{
				name:         "untagged server error stays empty on the wire",
				status:       500,
				expectedWire: "",
				expectedCtx:  backend.ErrorSourcePlugin,
			},
			{
				name:         "untagged client error stays empty on the wire",
				status:       400,
				expectedWire: "",
				expectedCtx:  backend.ErrorSourcePlugin,
			},
			{
				name:           "untagged error takes downstream from ctx",
				status:         502,
				markDownstream: true,
				expectedWire:   backend.ErrorSourceDownstream,
				expectedCtx:    backend.ErrorSourceDownstream,
			},
			{
				name:           "success response is not tagged even when ctx is downstream",
				status:         200,
				markDownstream: true,
				expectedWire:   "",
				expectedCtx:    backend.ErrorSourceDownstream,
			},
			{
				name:           "explicit downstream source is kept",
				status:         502,
				responseSource: backend.ErrorSourceDownstream,
				expectedWire:   backend.ErrorSourceDownstream,
				expectedCtx:    backend.ErrorSourceDownstream,
			},
			{
				name:           "explicit plugin source is kept",
				status:         500,
				responseSource: backend.ErrorSourcePlugin,
				expectedWire:   backend.ErrorSourcePlugin,
				expectedCtx:    backend.ErrorSourcePlugin,
			},
			{
				name:           "explicit plugin source overrides downstream marked on ctx",
				status:         500,
				responseSource: backend.ErrorSourcePlugin,
				markDownstream: true,
				expectedWire:   backend.ErrorSourcePlugin,
				expectedCtx:    backend.ErrorSourcePlugin,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				cdt := handlertest.NewHandlerMiddlewareTest(t,
					handlertest.WithMiddlewares(backend.NewErrorSourceMiddleware()),
				)
				cdt.TestHandler.CallResourceFunc = func(ctx context.Context, _ *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
					cdt.CallResourceCtx = ctx
					if tc.markDownstream {
						require.NoError(t, backend.WithDownstreamErrorSource(ctx))
					}
					return sender.Send(&backend.CallResourceResponse{
						Status:      tc.status,
						ErrorSource: tc.responseSource,
					})
				}

				var resp *backend.CallResourceResponse
				err := cdt.MiddlewareHandler.CallResource(context.Background(), &backend.CallResourceRequest{}, backend.CallResourceResponseSenderFunc(func(r *backend.CallResourceResponse) error {
					resp = r
					return nil
				}))
				require.NoError(t, err)
				require.Equal(t, tc.expectedWire, resp.ErrorSource)
				require.Equal(t, tc.expectedCtx, backend.ErrorSourceFromContext(cdt.CallResourceCtx))
			})
		}
	})

	t.Run("CallResource plugin response takes precedence over downstream responses", func(t *testing.T) {
		cdt := handlertest.NewHandlerMiddlewareTest(t,
			handlertest.WithMiddlewares(backend.NewErrorSourceMiddleware()),
		)
		cdt.TestHandler.CallResourceFunc = func(ctx context.Context, _ *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
			cdt.CallResourceCtx = ctx
			require.NoError(t, sender.Send(&backend.CallResourceResponse{Status: 500, ErrorSource: backend.ErrorSourceDownstream}))
			return sender.Send(&backend.CallResourceResponse{Status: 500, ErrorSource: backend.ErrorSourcePlugin})
		}

		err := cdt.MiddlewareHandler.CallResource(context.Background(), &backend.CallResourceRequest{}, backend.CallResourceResponseSenderFunc(func(_ *backend.CallResourceResponse) error {
			return nil
		}))
		require.NoError(t, err)
		require.Equal(t, backend.ErrorSourcePlugin, backend.ErrorSourceFromContext(cdt.CallResourceCtx))
	})

	t.Run("CallResource terminal error takes precedence over response sources", func(t *testing.T) {
		cdt := handlertest.NewHandlerMiddlewareTest(t,
			handlertest.WithMiddlewares(backend.NewErrorSourceMiddleware()),
		)
		cdt.TestHandler.CallResourceFunc = func(ctx context.Context, _ *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
			cdt.CallResourceCtx = ctx
			require.NoError(t, sender.Send(&backend.CallResourceResponse{Status: 500, ErrorSource: backend.ErrorSourceDownstream}))
			return someErr
		}

		err := cdt.MiddlewareHandler.CallResource(context.Background(), &backend.CallResourceRequest{}, backend.CallResourceResponseSenderFunc(func(_ *backend.CallResourceResponse) error {
			return nil
		}))
		require.ErrorIs(t, err, someErr)
		require.Equal(t, backend.ErrorSourcePlugin, backend.ErrorSourceFromContext(cdt.CallResourceCtx))
	})

	t.Run("CallResource response source precedence", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			responses []*backend.CallResourceResponse
			err       error
			expected  backend.ErrorSource
		}{
			{
				name:      "downstream only",
				responses: []*backend.CallResourceResponse{{Status: 502, ErrorSource: backend.ErrorSourceDownstream}},
				expected:  backend.ErrorSourceDownstream,
			},
			{
				name: "plugin before downstream",
				responses: []*backend.CallResourceResponse{
					{Status: 500, ErrorSource: backend.ErrorSourcePlugin},
					{Status: 502, ErrorSource: backend.ErrorSourceDownstream},
				},
				expected: backend.ErrorSourcePlugin,
			},
			{
				name: "untagged error after downstream",
				responses: []*backend.CallResourceResponse{
					{Status: 502, ErrorSource: backend.ErrorSourceDownstream},
					{Status: 500},
				},
				expected: backend.ErrorSourcePlugin,
			},
			{
				name:      "source on a success response is ignored",
				responses: []*backend.CallResourceResponse{{Status: 200, ErrorSource: backend.ErrorSourceDownstream}},
				expected:  backend.ErrorSourcePlugin,
			},
			{
				name:      "returned downstream error",
				responses: []*backend.CallResourceResponse{{Status: 500, ErrorSource: backend.ErrorSourcePlugin}},
				err:       backend.DownstreamErrorf("upstream unavailable"),
				expected:  backend.ErrorSourceDownstream,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				cdt := handlertest.NewHandlerMiddlewareTest(t,
					handlertest.WithMiddlewares(backend.NewErrorSourceMiddleware()),
				)
				cdt.TestHandler.CallResourceFunc = func(ctx context.Context, _ *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
					cdt.CallResourceCtx = ctx
					for _, r := range tc.responses {
						require.NoError(t, sender.Send(r))
					}
					return tc.err
				}

				err := cdt.MiddlewareHandler.CallResource(context.Background(), &backend.CallResourceRequest{}, backend.CallResourceResponseSenderFunc(func(_ *backend.CallResourceResponse) error {
					return nil
				}))
				if tc.err != nil {
					require.ErrorIs(t, err, tc.err)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, tc.expected, backend.ErrorSourceFromContext(cdt.CallResourceCtx))
			})
		}
	})
}

func setupHandlersWithError(cdt *handlertest.HandlerMiddlewareTest, err error) {
	cdt.TestHandler.QueryDataFunc = func(ctx context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		cdt.QueryDataCtx = ctx
		return nil, err
	}
	cdt.TestHandler.CheckHealthFunc = func(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
		cdt.CheckHealthCtx = ctx
		return nil, err
	}
	cdt.TestHandler.CallResourceFunc = func(ctx context.Context, _ *backend.CallResourceRequest, _ backend.CallResourceResponseSender) error {
		cdt.CallResourceCtx = ctx
		return err
	}
	cdt.TestHandler.CollectMetricsFunc = func(ctx context.Context, _ *backend.CollectMetricsRequest) (*backend.CollectMetricsResult, error) {
		cdt.CollectMetricsCtx = ctx
		return nil, err
	}
	cdt.TestHandler.SubscribeStreamFunc = func(ctx context.Context, _ *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
		cdt.SubscribeStreamCtx = ctx
		return nil, err
	}
	cdt.TestHandler.PublishStreamFunc = func(ctx context.Context, _ *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
		cdt.PublishStreamCtx = ctx
		return nil, err
	}
	cdt.TestHandler.RunStreamFunc = func(ctx context.Context, _ *backend.RunStreamRequest, _ *backend.StreamSender) error {
		cdt.RunStreamCtx = ctx
		return err
	}
	cdt.TestHandler.ValidateAdmissionFunc = func(ctx context.Context, _ *backend.AdmissionRequest) (*backend.ValidationResponse, error) {
		cdt.ValidateAdmissionCtx = ctx
		return nil, err
	}
	cdt.TestHandler.MutateAdmissionFunc = func(ctx context.Context, _ *backend.AdmissionRequest) (*backend.MutationResponse, error) {
		cdt.MutateAdmissionCtx = ctx
		return nil, err
	}
	cdt.TestHandler.ConvertObjectsFunc = func(ctx context.Context, _ *backend.ConversionRequest) (*backend.ConversionResponse, error) {
		cdt.ConvertObjectCtx = ctx
		return nil, err
	}
}
