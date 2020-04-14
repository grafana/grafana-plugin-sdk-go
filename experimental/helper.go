package experimental

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// InstanceManager is a singleton that holds all datasource instances
type InstanceManager struct {
	sync.RWMutex

	instances map[string]instanceInfo
	host      PluginHost
}

// NewInstanceManager creates a new instance manager.
func NewInstanceManager(host PluginHost) *InstanceManager {
	return &InstanceManager{
		host:      host,
		instances: make(map[string]instanceInfo),
	}
}

// RunGRPCServer starts the GRPC server
func (p *InstanceManager) RunGRPCServer() error {
	return backend.Serve(backend.ServeOpts{
		CallResourceHandler: p,
		QueryDataHandler:    p,
		CheckHealthHandler:  p,
	})
}

type instanceInfo struct {
	// the exe updated time + the datasource updated time
	// used as a flag to check if anything changed
	updated int64

	// The raw GRPC values that create the instance
	config backend.PluginConfig

	// The specific instance
	instance DataSourceInstance

	// The last time it was used (so we can expire old things)
	last time.Time
}

func (p *InstanceManager) getDataSourceInstance(config backend.PluginConfig) (DataSourceInstance, error) {
	if config.DataSourceConfig == nil {
		return nil, fmt.Errorf("no datasource in PluginConfig")
	}
	updated := config.Updated.UnixNano() + config.DataSourceConfig.Updated.UnixNano()
	key := fmt.Sprintf("%d/%d", config.OrgID, config.DataSourceConfig.ID)

	p.RLock()
	defer p.RUnlock()

	info, ok := p.instances[key]

	// Check if we need to create a new instance
	if !ok || updated != info.updated {
		if ok {
			info.instance.Destroy()
		}

		// Create a new one
		instance, err := p.host.NewDataSourceInstance(config)
		if err != nil {
			return nil, err
		}

		info = instanceInfo{
			updated:  updated,
			config:   config,
			instance: instance,
		}

		// Set the instance for the key (will replace the old value if exists)
		p.instances[key] = info
	}
	info.last = time.Now()
	return info.instance, nil
}

// CheckHealth checks if the plugin is running properly
func (p *InstanceManager) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// 1. Check the datasource config
	if req.PluginConfig.DataSourceConfig != nil {
		ds, err := p.getDataSourceInstance(req.PluginConfig)
		if err != nil {
			// Error reading datasource config
			return &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: err.Error(),
			}, nil
		}
		return ds.CheckHealth(), nil
	}

	// Otherwise the host application
	return p.host.CheckHostHealth(req.PluginConfig), nil
}

// QueryData queries for data.
func (p *InstanceManager) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	if req.PluginConfig.DataSourceConfig != nil {
		ds, err := p.getDataSourceInstance(req.PluginConfig)
		if err != nil {
			return nil, err
		}
		return ds.QueryData(req)
	}
	return nil, fmt.Errorf("only datasource supports QueryData (for now)")
}

// CallResource calls a resource.
func (p *InstanceManager) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	if req.PluginConfig.DataSourceConfig != nil {
		ds, err := p.getDataSourceInstance(req.PluginConfig)
		if err != nil {
			return err
		}
		return ds.CallResource(req, sender)
	}

	return fmt.Errorf("only datasource supports Resources (for now)")
}
