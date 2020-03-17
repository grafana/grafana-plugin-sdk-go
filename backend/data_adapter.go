package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// dataSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type dataSDKAdapter struct {
	queryDataHandler QueryDataHandler
	configCache      map[int64]*PluginConfig // thread safety?!
}

func newDataSDKAdapter(handler QueryDataHandler) *dataSDKAdapter {
	return &dataSDKAdapter{
		queryDataHandler: handler,
	}
}

func (a *dataSDKAdapter) QueryData(ctx context.Context, preq *pluginv2.QueryDataRequest) (*pluginv2.QueryDataResponse, error) {
	req := fromProto().QueryDataRequest(preq)

	// Parse the query objects
	for i, q := range req.Queries {
		m, err := a.queryDataHandler.ParseQueryModel(q.JSON)
		if err != nil {
			return nil, err
		}
		req.Queries[i].Model = m
	}

	// Check for cached version here
	dscfg := req.PluginConfig.DataSourceConfig
	cfg := a.configCache[dscfg.ID]
	if cfg != nil {
		if cfg.Updated != req.PluginConfig.Updated || cfg.DataSourceConfig.Updated != dscfg.Updated {
			// CONFIG CHANGED! maybe make a callback?
			cfg = nil
		}
	}
	if cfg == nil { // first time or if it changed
		m, err := a.queryDataHandler.ParseDataSourceConfigModel(dscfg.JSONData, dscfg.DecryptedSecureJSONData)
		if err != nil {
			return nil, err
		}
		cfg = &req.PluginConfig
		cfg.DataSourceConfig.Model = m
		a.configCache[dscfg.ID] = cfg
	}
	req.PluginConfig = *cfg

	resp, err := a.queryDataHandler.QueryData(ctx, req)
	if err != nil {
		return nil, err
	}

	return toProto().QueryDataResponse(resp)
}
