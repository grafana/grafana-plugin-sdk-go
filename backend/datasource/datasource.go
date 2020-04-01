package datasource

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// CheckDataSourceHealthRequest contains the healthcheck request
type CheckDataSourceHealthRequest struct {
	pluginConfig     backend.PluginConfig
	OrgID            int64
	DataSourceConfig backend.DataSourceConfig
}

// CheckDataSourceHealthHandler enables users to send health check
// requests to a data source plugin.
type CheckDataSourceHealthHandler interface {
	CheckDataSourceHealth(ctx context.Context, req *CheckDataSourceHealthRequest) (*backend.CheckHealthResult, error)
}

type CheckDataSourceHealthHandlerFunc func(ctx context.Context, req *CheckDataSourceHealthRequest) (*backend.CheckHealthResult, error)

func (fn CheckDataSourceHealthHandlerFunc) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return fn(ctx, &CheckDataSourceHealthRequest{
		pluginConfig:     req.PluginConfig,
		OrgID:            req.PluginConfig.OrgID,
		DataSourceConfig: *(req.PluginConfig.DataSourceConfig),
	})
}

type CallDataSourceResourceRequest struct {
	pluginConfig     backend.PluginConfig
	OrgID            int64
	DataSourceConfig backend.DataSourceConfig
	Path             string
	Method           string
	URL              string
	Headers          map[string][]string
	Body             []byte
	User             *backend.User
}

// CallDataSourceResourceHandler handles resource calls.
type CallDataSourceResourceHandler interface {
	CallDataSourceResource(ctx context.Context, req *CallDataSourceResourceRequest, sender backend.CallResourceResponseSender) error
}

type CallDataSourceResourceHandlerFunc func(ctx context.Context, req *CallDataSourceResourceRequest, sender backend.CallResourceResponseSender) error

func (fn CallDataSourceResourceHandlerFunc) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	return fn(ctx, &CallDataSourceResourceRequest{
		pluginConfig:     req.PluginConfig,
		OrgID:            req.PluginConfig.OrgID,
		DataSourceConfig: *(req.PluginConfig.DataSourceConfig),
		User:             req.User,
		Path:             req.Path,
		URL:              req.URL,
		Method:           req.Method,
		Headers:          req.Headers,
		Body:             req.Body,
	}, sender)
}

func NewCheckDataSourceHealthHandlerFunc(h backend.CheckHealthHandler) CheckDataSourceHealthHandlerFunc {
	return func(ctx context.Context, req *CheckDataSourceHealthRequest) (*backend.CheckHealthResult, error) {
		return h.CheckHealth(ctx, &backend.CheckHealthRequest{
			PluginConfig: req.pluginConfig,
		})
	}
}

func NewCallDataSourceResourceHandlerFunc(h backend.CallResourceHandler) CallDataSourceResourceHandlerFunc {
	return func(ctx context.Context, req *CallDataSourceResourceRequest, sender backend.CallResourceResponseSender) error {
		return h.CallResource(ctx, &backend.CallResourceRequest{
			PluginConfig: req.pluginConfig,
			User:         req.User,
			Path:         req.Path,
			URL:          req.URL,
			Method:       req.Method,
			Headers:      req.Headers,
			Body:         req.Body,
		}, sender)
	}
}
