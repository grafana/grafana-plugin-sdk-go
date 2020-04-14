package experimental

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// PluginHelper singelton host service
type PluginHelper struct {
	sync.RWMutex

	instances map[string]instanceInfo
	host      PluginHost
}

// Init creates the datasource and sets up all the routes
func NewPluginHelper(host PluginHost) *PluginHelper {
	return &PluginHelper{
		host:      host,
		instances: make(map[string]instanceInfo),
	}
}

// RunGRPCServer starts the GRPC server
func (p *PluginHelper) RunGRPCServer() error {
	return backend.Serve(backend.ServeOpts{
		CallResourceHandler: p,
		QueryDataHandler:    p,
		CheckHealthHandler:  p,
	})
}

type instanceInfo struct {
	updated int64

	// The raw GRPC values that create the instance
	config backend.PluginConfig

	// the Specific instance
	instance DataSourceInstance

	// The last time it was used
	last time.Time
}

func (p *PluginHelper) getDataSourceInstance(config backend.PluginConfig) (DataSourceInstance, error) {
	if config.DataSourceConfig == nil {
		return nil, nil
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

		// Set the instance for the key
		p.instances[key] = info
	}
	info.last = time.Now()
	return info.instance, nil
}

// CheckHealth checks if the plugin is running properly
func (p *PluginHelper) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// 1.  Check datasource requests
	ds, err := p.getDataSourceInstance(req.PluginConfig)
	if err != nil {
		// Error reading datasource config
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: err.Error(),
		}, nil
	}
	if ds != nil {
		return ds.CheckHealth(), nil
	}

	// finally, try the plugin host itself
	return p.host.CheckHostHealth(req.PluginConfig), nil
}

// QueryData queries for data.
func (p *PluginHelper) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// 1.  Check datasource requests
	ds, err := p.getDataSourceInstance(req.PluginConfig)
	if err != nil {
		return nil, err
	}
	if ds != nil {
		return ds.QueryData(req)
	}

	return nil, fmt.Errorf("host does not support query data")
}

// CallResource returns HTTP style results
func (p *PluginHelper) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	// 1.  Check datasource requests
	ds, err := p.getDataSourceInstance(req.PluginConfig)
	if err != nil {
		return err
	}
	if ds != nil {
		return ds.CallResource(req, sender)
	}

	return fmt.Errorf("host does not (yet!) support query data")
}
