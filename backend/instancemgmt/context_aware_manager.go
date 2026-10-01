package instancemgmt

import (
	"context"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
)

// NewInstanceManagerWrapper creates a new instance manager that dynamically selects
// between standard and TTL instance managers based on feature toggles from the Grafana config.
func NewInstanceManagerWrapper(provider InstanceProvider) InstanceManager {
	return &instanceManagerWrapper{
		provider:        provider,
		standardManager: New(provider),
	}
}

// instanceManagerWrapper is a wrapper that dynamically selects the appropriate
// instance manager implementation based on feature toggles in the context.
type instanceManagerWrapper struct {
	provider        InstanceProvider
	standardManager InstanceManager

	// ttlManager is constructed lazily: whether it's ever needed depends on
	// per-request feature toggle state, which isn't known at construction
	// time. Building it eagerly would spawn its cache's cleanup goroutine in
	// every plugin process regardless of whether the toggle is ever enabled.
	ttlManagerOnce sync.Once
	ttlManager     InstanceManager
}

// getTTLManager returns the TTL instance manager, constructing it on first use.
func (c *instanceManagerWrapper) getTTLManager() InstanceManager {
	c.ttlManagerOnce.Do(func() {
		c.ttlManager = NewTTLInstanceManager(c.provider)
	})
	return c.ttlManager
}

// selectManager returns the appropriate instance manager based on the feature toggle
// from the Grafana config in the plugin context.
func (c *instanceManagerWrapper) selectManager(_ context.Context, pluginContext backend.PluginContext) InstanceManager {
	// Check if TTL instance manager feature toggle is enabled
	if pluginContext.GrafanaConfig != nil {
		featureToggles := pluginContext.GrafanaConfig.FeatureToggles()
		if featureToggles.IsEnabled(featuretoggles.TTLInstanceManager) {
			return c.getTTLManager()
		}
	}

	// Default to standard instance manager
	return c.standardManager
}

// Get returns an Instance using the appropriate manager based on feature toggles.
func (c *instanceManagerWrapper) Get(ctx context.Context, pluginContext backend.PluginContext) (Instance, error) {
	manager := c.selectManager(ctx, pluginContext)
	return manager.Get(ctx, pluginContext)
}

// Do provides an Instance as argument to fn using the appropriate manager based on feature toggles.
func (c *instanceManagerWrapper) Do(ctx context.Context, pluginContext backend.PluginContext, fn InstanceCallbackFunc) error {
	manager := c.selectManager(ctx, pluginContext)
	return manager.Do(ctx, pluginContext, fn)
}
