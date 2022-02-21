# E2E HTTP Fixture Proxy

The goal of the proxy is to provide a way to record and replay HTTP interactions between a data source backend and the target API. The use of recorded fixtures makes testing infrastructure simpler, and the stability of response data makes it easier to achieve deterministic tests.

The default storage for recorded interactions are [HAR](https://en.wikipedia.org/wiki/HAR_(file_format)) files. Using the HAR format allows recorded interactions to be easily reviewed in tools like Postman or in browser dev tools. It's also possible to use browser generated HARs as the source of the fixture data. In this scenario the proxy would only be used for playback.

## Setup & Usage

1. Create a `proxy.json` config file in the root of your plugin repo and replace `example.com` with the `host` or `host:port` of the API you wish to capture:

```json
{
	"storage": {
		"type": "har",
		"path": "fixtures/e2e.har"
	},
	"address": "127.0.0.1:9999",
	"hosts": ["example.com"]
}
```

2. Start proxy using one of the commands listed below. For example:

```
mage e2e:append
```

3. Point Grafana at the proxy by exporting the `HTTP_PROXY` and `HTTPS_PROXY` environment variables:

```
export HTTP_PROXY=127.0.0.1:9999
export HTTPS_PROXY=127.0.0.1:9999
```

4. Start Grafana

**Note:** Only queries with **absolute time ranges** should be used with the proxy. Relative time ranges are not supported in the default matcher.

## Config

### address

The hostname or IP address and port for the proxy server.

Default: `127.0.0.1:9999`

### hosts

An allow list can be used to restrict captured traffic to a specific set of hosts.

Default: `[]` (traffic for all hosts will be captured)

### storage

An object used to define a type and configuration options for the fixture's storage.

Default: 
```json
{
	"type": "har",
	"path": "fixtures/e2e.har"
}
```

## Mage Commands

### Append mode

Append mode should be used to record interactions for any new tests. It will record requests and responses for any requests that haven't been seen before, and return recorded responses for any requests that match previously recorded interactions.

```
mage e2e:append
```

### Overwrite mode

Overwrite mode should be used if previously recorded interactions need to be replaced with new data.

```
mage e2e:overwrite
```

### Replay mode

Replay mode should be used in CI or locally if only playback of recorded data is needed. Replay mode will return recorded responses for any matching requests, and pass any requests that don't match recorded interactions to the target API.

```
mage e2e:replay
```

### Certificate

This command prints the CA certificate to stdout so that it can be added to the local test environment.

```
mage e2e:certificate
```

## Modifying default behavior

You can modify the default request processor, response processor, and matching behavior in your plugin project by modifying the `Magefile.go` in the root of your project:

```go
//+build mage

package main

import (
	// mage:import
	build "github.com/grafana/grafana-plugin-sdk-go/build"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/config"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/fixture"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/storage"
)

// Default configures the default target.
var Default = build.BuildAll

func CustomE2E() error {
	cfg, err := config.LoadConfig("proxy.json")
	if err != nil {
		return err
	}
	var store storage.Storage
	if cfg.Storage == nil || cfg.Storage.Type == config.StorageTypeHAR {
		store = storage.NewHARStorage(cfg.Storage.Path)
	}

	fixture := fixture.NewFixture(store)

	// modify incoming requests
	fixture.WithRequestProcessor(func(req *http.Request) *http.Request {
		req.URL.Path = "/example"
		return req
	})

	// modify incoming responses
	fixture.WithResponseProcessor(func(res *http.Response) *http.Response {
		res.StatusCode = 201
		return res
	})

	// modify matching behavior
	fixture.WithMatcher(func(a, b *http.Request) bool {
			return true
	})

	proxy := e2e.NewProxy(e2e.ProxyModeAppend, fixture, cfg)
	return proxy.Start()
}
```

