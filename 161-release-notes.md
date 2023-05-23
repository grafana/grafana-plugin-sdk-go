## What's Changed
* Use tenant ID from incoming gRPC meta for instance caching by @wbrowne in https://github.com/grafana/grafana-plugin-sdk-go/pull/676

**Full Changelog**: https://github.com/grafana/grafana-plugin-sdk-go/compare/v0.161.0...v0.162.0

## Breaking change
Both the [Instance Manager](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go@v0.161.0/backend/instancemgmt#InstanceManager) and [Instance Provider](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go@v0.161.0/backend/instancemgmt#InstanceProvider) interfaces have been updated to require a [context.Context](https://pkg.go.dev/context#Context) as part of their APIs.This affects all plugins which perform manual instance management via the Instance Manager API.

For example:

```go
package main

func main() {
	err := datasource.Serve(plugin.New())
	if err != nil {
		os.Exit(1)
	}
}

```

```go
package plugin

type Plugin struct {
	im instancemgmt.InstanceManager
}

type instance struct {
	token string
}

func New() datasource.ServeOpts {
	p := newPlugin()
	return datasource.ServeOpts{
		QueryDataHandler: p,
	}
}

func newPlugin() *Plugin {
	return &Service{
		im: datasource.NewInstanceManager(newDataSourceInstance)
	}
}

func newDataSourceInstance(s backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	cfg := models.LoadCfg(s)
	return &instance{token: cfg.Token}, nil
}

func (p *Plugin) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	i, err := p.im.Get(ctx, req.PluginContext) // ctx is now required
	if err != nil {
		return nil, err
	}
	// ..
}
```

### Recommended fix

#### Automatic instance management
Automatic instance management for data sources was added to the SDK in version [0.97.0](https://github.com/grafana/grafana-plugin-sdk-go/releases/tag/v0.97.0), which
removes the need for the plugin developer to use the [Instance Manager](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go@v0.161.0/backend/instancemgmt#InstanceManager) directly. Support for app plugins was added in [v0.140.0](https://github.com/grafana/grafana-plugin-sdk-go/releases/tag/v0.140.0).

To use auto instance management, please refer to the relevant SDK documentation:

- [Datasources](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go@v0.161.0/backend/datasource#Manage)
- [Apps](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go@v0.161.0/backend/app#Manage)


The following demonstrates a migration from the example above to use automatic instance management:

```go
package main

func main() {
	err := datasource.Manage("grafana-test-datasource", plugin.New(), datasource.ManageOpts{})
	if err != nil {
		os.Exit(1)
	}
}

```

```go
package plugin

func New(s backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	cfg := models.LoadCfg(s)
	return &plugin{token: cfg.Token}, nil
}

type plugin struct {
	token string
}

func (p *plugin) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return backend.NewQueryDataResponse(), nil
}
```

#### Alternative
If you would prefer not to use automatic instance management, you can instead just pass [context.Context](https://pkg.go.dev/context#Context) from each handler to the instance manager.

For example:

```go
func (p * Plugin) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	i, err := p.im.Get(ctx, req.PluginContext) // ctx is now required
	if err != nil {
		return nil, err
	}
	return makeQuery(ctx, i.token)
}

func (p * Plugin) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	i, err := p.im.Get(ctx, req.PluginContext) // ctx is now required
	if err != nil {
		return nil, err
	}
	return makeQuery(ctx, i.token)
}
```

## Compatibility
```
gorelease -base v0.161.0 -version v0.162.0
# github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt
## incompatible changes
InstanceManager.Do: changed from func(github.com/grafana/grafana-plugin-sdk-go/backend.PluginContext, InstanceCallbackFunc) error to func(context.Context, github.com/grafana/grafana-plugin-sdk-go/backend.PluginContext, InstanceCallbackFunc) error
InstanceManager.Get: changed from func(github.com/grafana/grafana-plugin-sdk-go/backend.PluginContext) (Instance, error) to func(context.Context, github.com/grafana/grafana-plugin-sdk-go/backend.PluginContext) (Instance, error)
InstanceProvider.GetKey: changed from func(github.com/grafana/grafana-plugin-sdk-go/backend.PluginContext) (interface{}, error) to func(context.Context, github.com/grafana/grafana-plugin-sdk-go/backend.PluginContext) (interface{}, error)
InstanceProvider.NeedsUpdate: changed from func(github.com/grafana/grafana-plugin-sdk-go/backend.PluginContext, CachedInstance) bool to func(context.Context, github.com/grafana/grafana-plugin-sdk-go/backend.PluginContext, CachedInstance) bool
InstanceProvider.NewInstance: changed from func(github.com/grafana/grafana-plugin-sdk-go/backend.PluginContext) (Instance, error) to func(context.Context, github.com/grafana/grafana-plugin-sdk-go/backend.PluginContext) (Instance, error)

# github.com/grafana/grafana-plugin-sdk-go/backend/tenant
## compatible changes
package added
```
