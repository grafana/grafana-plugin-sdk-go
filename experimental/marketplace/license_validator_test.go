package marketplace

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

func TestValidateMarketplaceLicenseToken(t *testing.T) {
	now := time.Now()
	cause := errors.New("license validation failed")
	tests := []struct {
		name    string
		token   licensing.LicenseToken
		allowed bool
	}{
		{name: "valid", token: licensing.LicenseToken{Status: licensing.Valid}, allowed: true},
		{name: "invalid", token: licensing.LicenseToken{Status: licensing.Invalid, Error: cause}},
		{name: "invalid subject", token: licensing.LicenseToken{Status: licensing.InvalidSubject, Error: cause}},
		{name: "expired token with active entitlement", token: licensing.LicenseToken{Status: licensing.Expired, Error: cause, Expires: now.Add(-30 * 24 * time.Hour).Unix(), LicenseExpires: now.Add(24 * time.Hour).Unix()}, allowed: true},
		{name: "expired entitlement within grace", token: licensing.LicenseToken{Status: licensing.Expired, Error: cause, LicenseExpires: now.Add(-13 * 24 * time.Hour).Unix()}, allowed: true},
		{name: "expired entitlement outside grace", token: licensing.LicenseToken{Status: licensing.Expired, Error: cause, Expires: now.Add(-time.Hour).Unix(), LicenseExpires: now.Add(-15 * 24 * time.Hour).Unix()}},
		{name: "invalid token cannot use grace", token: licensing.LicenseToken{Status: licensing.Invalid, Error: cause, LicenseExpires: now.Add(24 * time.Hour).Unix()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMarketplaceLicenseToken(&tt.token, "test-plugin")
			if tt.allowed {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, errInvalidLicense)
				require.ErrorIs(t, err, cause)
			}
		})
	}
}

func TestLicenseValidatorRejectsUnlicensedRequests(t *testing.T) {
	// An unsupported instance would return Unimplemented if any handler skipped validation.
	validator := newLicenseValidator(struct{}{})
	pluginContext := backend.PluginContext{PluginID: "test-plugin"}
	tests := []struct {
		name string
		ctx  context.Context
		want error
	}{
		{name: "missing license", ctx: context.Background(), want: errNoLicenseFound},
		{name: "malformed metadata", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", "{")), want: errNoLicenseFound},
		{name: "invalid license", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", `{"licenseToken":"invalid","appURL":"https://grafana.example.com"}`)), want: errInvalidLicense},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := validator.QueryData(tt.ctx, &backend.QueryDataRequest{
				PluginContext: pluginContext,
				Queries:       []backend.DataQuery{{RefID: "A"}, {RefID: "B"}},
			})
			require.NoError(t, err)
			require.Len(t, response.Responses, 2)
			for _, refID := range []string{"A", "B"} {
				result := response.Responses[refID]
				require.ErrorIs(t, result.Error, tt.want)
				require.Equal(t, backend.StatusUnauthorized, result.Status)
				require.Equal(t, backend.ErrorSourceDownstream, result.ErrorSource)
				require.Empty(t, result.Frames)
			}

			health, err := validator.CheckHealth(tt.ctx, &backend.CheckHealthRequest{PluginContext: pluginContext})
			require.Nil(t, health)
			require.ErrorIs(t, err, tt.want)
			metrics, err := validator.CollectMetrics(tt.ctx, &backend.CollectMetricsRequest{PluginContext: pluginContext})
			require.Nil(t, metrics)
			require.ErrorIs(t, err, tt.want)
			require.ErrorIs(t, validator.CallResource(tt.ctx, &backend.CallResourceRequest{PluginContext: pluginContext}, nil), tt.want)
			subscription, err := validator.SubscribeStream(tt.ctx, &backend.SubscribeStreamRequest{PluginContext: pluginContext})
			require.Nil(t, subscription)
			require.ErrorIs(t, err, tt.want)
			publication, err := validator.PublishStream(tt.ctx, &backend.PublishStreamRequest{PluginContext: pluginContext})
			require.Nil(t, publication)
			require.ErrorIs(t, err, tt.want)
			require.ErrorIs(t, validator.RunStream(tt.ctx, &backend.RunStreamRequest{PluginContext: pluginContext}, nil), tt.want)
		})
	}
}

func TestValidateMarketplaceLicenseDoesNotUseEnvironment(t *testing.T) {
	t.Setenv(marketplaceLicenseTextEnv, "environment token")
	t.Setenv(marketplaceLicensePathEnv, "environment.jwt")
	t.Setenv(marketplaceAppURLEnv, "https://grafana.example.com")
	t.Setenv(marketplaceLicenseValidationKeyEnv, "environment keys")
	require.ErrorIs(t, validateMarketplaceLicense(context.Background(), "test-plugin"), errNoLicenseFound)
}

func TestLicenseValidatorRevalidatesEachRequest(t *testing.T) {
	validator := newLicenseValidator(struct{}{})
	request := &backend.QueryDataRequest{PluginContext: backend.PluginContext{PluginID: "test-plugin"}, Queries: []backend.DataQuery{{RefID: "A"}}}
	invalidContext := metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", `{"licenseToken":"invalid"}`))
	response, err := validator.QueryData(invalidContext, request)
	require.NoError(t, err)
	require.ErrorIs(t, response.Responses["A"].Error, errInvalidLicense)
	response, err = validator.QueryData(context.Background(), request)
	require.NoError(t, err)
	require.ErrorIs(t, response.Responses["A"].Error, errNoLicenseFound)
}

type disposableMarketplaceInstance struct {
	disposed bool
}

func (i *disposableMarketplaceInstance) Dispose() { i.disposed = true }

func TestMarketplaceInstanceFactory(t *testing.T) {
	instance := &disposableMarketplaceInstance{}
	settings := backend.DataSourceInstanceSettings{UID: "datasource"}
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "value")
	factory := marketplaceInstanceFactory(func(gotCtx context.Context, gotSettings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
		require.Same(t, ctx, gotCtx)
		require.Equal(t, settings, gotSettings)
		return instance, nil
	})
	wrapped, err := factory(ctx, settings)
	require.NoError(t, err)
	validator, ok := wrapped.(*LicenseValidator)
	require.True(t, ok)
	require.Same(t, instance, validator.instance)
	validator.Dispose()
	require.True(t, instance.disposed)
	// Wrapping an instance without Dispose must remain safe.
	newLicenseValidator(struct{}{}).Dispose()
}

func TestMarketplaceInstanceFactoryError(t *testing.T) {
	cause := errors.New("factory failed")
	factory := marketplaceInstanceFactory(func(context.Context, backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
		return nil, cause
	})
	instance, err := factory(context.Background(), backend.DataSourceInstanceSettings{})
	require.Nil(t, instance)
	require.ErrorIs(t, err, cause)
}
