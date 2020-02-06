# Grafana Plugin SDK for Go

Develop Grafana backend plugins with this Go SDK.

**Warning**: This SDK is currently in alpha and will likely have major breaking changes during early development. Please do not consider this SDK published until this warning has been removed.

## Usage

```go
package main

import (
	"context"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)


//
// This is just an example - a real project would encapsulate this into its own file
//
type MyCallHandler struct {
}

func (h *MyCallHandler) CheckHealth(ctx context.Context) (*backend.CheckHealthResult, error) {
	return nil, nil
}

func (h *MyCallHandler) CallResource(ctx context.Context, req *backend.CallResourceRequest) (*backend.CallResourceResponse, error) {
	return nil, nil
}

func (h *MyCallHandler) DataQuery(ctx context.Context, req *backend.DataQueryRequest) (*backend.DataQueryResponse, error) {
	return nil, nil
}

func (h *MyCallHandler) TransformData(ctx context.Context, req *backend.DataQueryRequest, callBack backend.TransformCallBackHandler) (*backend.DataQueryResponse, error) {
	return nil, nil
}

func main() {
	var handler *MyCallHandler
	handler = new(MyCallHandler)

	backend.Serve(backend.ServeOpts{
		CallResourceHandler:  handler,
		CheckHealthHandler:   handler,
		DataQueryHandler:     handler,
		TransformDataHandler: handler,
	})
}
```

## Developing

### Generate Go code for Protobuf definitions

```
make build-proto
```

### Changing `generic_*.go` files in the `dataframe` package

Currently [genny](https://github.com/cheekybits/genny) is used for generating some go code. If you make changes to generic template files then `genny` needs to be installed, and then `go generate` needs to be run from with the `dataframe` directory. Changed generated files should be committed with the change in the template files.
