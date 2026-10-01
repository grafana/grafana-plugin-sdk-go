package marketplace

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

var (
	errNoLicenseFound = errors.New("no valid license found")
	errInvalidLicense = errors.New("invalid license for the marketplace plugin")
)

var _ backend.QueryDataHandler = (*LicenseValidator)(nil)
var _ backend.CallResourceHandler = (*LicenseValidator)(nil)
var _ backend.CheckHealthHandler = (*LicenseValidator)(nil)
var _ backend.CollectMetricsHandler = (*LicenseValidator)(nil)
var _ backend.StreamHandler = (*LicenseValidator)(nil)
var _ instancemgmt.InstanceDisposer = (*LicenseValidator)(nil)

func marketplaceInstanceFactory(factory datasource.InstanceFactoryFunc) datasource.InstanceFactoryFunc {
	return func(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
		i, err := factory(ctx, settings)
		if err != nil {
			return nil, err
		}
		return newLicenseValidator(i), nil
	}
}

// LicenseValidator validates the request license before calling the wrapped data source.
type LicenseValidator struct {
	instance instancemgmt.Instance
}

func newLicenseValidator(instance instancemgmt.Instance) *LicenseValidator {
	return &LicenseValidator{
		instance: instance,
	}
}

func (lv *LicenseValidator) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	if err := validateMarketplaceLicense(ctx, req.PluginContext.PluginID); err != nil {
		if errors.Is(err, errInvalidLicense) || errors.Is(err, errNoLicenseFound) {
			return getErrorResponse(req, err), nil
		}

		return nil, err
	}
	if queryHandler, ok := lv.instance.(backend.QueryDataHandler); ok {
		return queryHandler.QueryData(ctx, req)
	}

	return nil, status.Error(codes.Unimplemented, "QueryData not implemented")
}

func (lv *LicenseValidator) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	if err := validateMarketplaceLicense(ctx, req.PluginContext.PluginID); err != nil {
		return err
	}
	if resourceHandler, ok := lv.instance.(backend.CallResourceHandler); ok {
		return resourceHandler.CallResource(ctx, req, sender)
	}

	return status.Error(codes.Unimplemented, "CallResource not implemented")
}

func (lv *LicenseValidator) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if err := validateMarketplaceLicense(ctx, req.PluginContext.PluginID); err != nil {
		return nil, err
	}
	if healthHandler, ok := lv.instance.(backend.CheckHealthHandler); ok {
		return healthHandler.CheckHealth(ctx, req)
	}

	return nil, status.Error(codes.Unimplemented, "CheckHealth not implemented")
}

func (lv *LicenseValidator) CollectMetrics(ctx context.Context, req *backend.CollectMetricsRequest) (*backend.CollectMetricsResult, error) {
	if err := validateMarketplaceLicense(ctx, req.PluginContext.PluginID); err != nil {
		return nil, err
	}
	if metricsHandler, ok := lv.instance.(backend.CollectMetricsHandler); ok {
		return metricsHandler.CollectMetrics(ctx, req)
	}

	return nil, status.Error(codes.Unimplemented, "CollectMetrics not implemented")
}

func (lv *LicenseValidator) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	if err := validateMarketplaceLicense(ctx, req.PluginContext.PluginID); err != nil {
		return nil, err
	}
	if streamHandler, ok := lv.instance.(backend.StreamHandler); ok {
		return streamHandler.SubscribeStream(ctx, req)
	}

	return nil, status.Error(codes.Unimplemented, "SubscribeStream not implemented")
}

func (lv *LicenseValidator) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	if err := validateMarketplaceLicense(ctx, req.PluginContext.PluginID); err != nil {
		return err
	}
	if streamHandler, ok := lv.instance.(backend.StreamHandler); ok {
		return streamHandler.RunStream(ctx, req, sender)
	}

	return status.Error(codes.Unimplemented, "RunStream not implemented")
}

func (lv *LicenseValidator) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	if err := validateMarketplaceLicense(ctx, req.PluginContext.PluginID); err != nil {
		return nil, err
	}
	if streamHandler, ok := lv.instance.(backend.StreamHandler); ok {
		return streamHandler.PublishStream(ctx, req)
	}

	return nil, status.Error(codes.Unimplemented, "PublishStream not implemented")
}

func (lv *LicenseValidator) Dispose() {
	if disposer, ok := lv.instance.(instancemgmt.InstanceDisposer); ok {
		disposer.Dispose()
	}
}

func validateMarketplaceLicense(ctx context.Context, pluginID string) error {
	l, exists := licensing.LicenseFromIncomingGRPCContext(ctx)
	if !exists {
		backend.Logger.Error("No license information found in request")
		return errNoLicenseFound
	}

	token := &licensing.LicenseToken{}
	token.Parse(l.LicenseToken, l.AppURL, l.ValidationKeys, pluginID)
	return validateMarketplaceLicenseToken(token, pluginID)
}

func validateMarketplaceLicenseToken(token *licensing.LicenseToken, pluginID string) error {
	if token.Error == nil {
		return nil
	}
	backend.Logger.Error("Marketplace License Error", "error", token.Error)
	if token.Status == licensing.Expired {
		if grace, ok := marketplaceLicenseGracePeriod(token, time.Now()); ok {
			backend.Logger.Warn("The plugin will work until", "date", grace)
			return nil
		}
	}
	backend.Logger.Error("Invalid license for the marketplace plugin", "pluginId", pluginID)
	return errors.Join(errInvalidLicense, token.Error)
}

func getErrorResponse(req *backend.QueryDataRequest, err error) *backend.QueryDataResponse {
	dataResponse := backend.DataResponse{
		Frames:      data.Frames{},
		Error:       err,
		ErrorSource: backend.ErrorSourceDownstream,
		Status:      backend.StatusUnauthorized,
	}

	response := backend.NewQueryDataResponse()
	for _, q := range req.Queries {
		response.Responses[q.RefID] = dataResponse
	}

	return response
}
