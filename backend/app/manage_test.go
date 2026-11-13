package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
)

func TestManageAppliesLicensing(t *testing.T) {
	licensingErr := errors.New("licensing failed")
	conversionHandler := backend.ConvertObjectsFunc(func(context.Context, *backend.ConversionRequest) (*backend.ConversionResponse, error) {
		return nil, nil
	})
	called := false

	err := Manage("my-app", func(context.Context, backend.AppInstanceSettings) (instancemgmt.Instance, error) {
		return struct{}{}, nil
	}, ManageOpts{
		GRPCSettings:      backend.GRPCSettings{MaxReceiveMsgSize: 1},
		ConversionHandler: conversionHandler,
		Licensing: func(pluginID string, opts backend.ServeOpts) (backend.ServeOpts, error) {
			called = true
			require.Equal(t, "my-app", pluginID)
			require.NotNil(t, opts.CheckHealthHandler)
			require.NotNil(t, opts.CallResourceHandler)
			require.NotNil(t, opts.QueryDataHandler)
			require.NotNil(t, opts.StreamHandler)
			require.NotNil(t, opts.ConversionHandler)
			require.Equal(t, 1, opts.GRPCSettings.MaxReceiveMsgSize)
			return backend.ServeOpts{}, licensingErr
		},
	})

	require.True(t, called)
	require.ErrorIs(t, err, licensingErr)
}
