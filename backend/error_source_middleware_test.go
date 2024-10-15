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
