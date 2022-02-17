# E2E HTTP Fixture Proxy

The goal of the proxy is to provide a way to record and replay HTTP interactions between a data source backend and the target API. The use of recorded fixtures makes testing infrastructure simpler, and the stability of response data makes it easier to achieve deterministic tests.

The default storage for recorded interactions are [HAR](https://en.wikipedia.org/wiki/HAR_(file_format)) files. Using the HAR format allows recorded interactions to be easily reviewed in tools like Postman or in browser dev tools. It's also possible to use browser generated HARs as the source of the fixture data. In this scenario the proxy would only be used for playback.

## Setup & Usage

1. Start proxy using one of the commands listed below. For example:

```
mage e2e:append 127.0.0.1:9999 fixtures/e2e.har
```

2. Point Grafana at the proxy by exporting the `HTTP_PROXY` and `HTTPS_PROXY` environment variables:

```
export HTTP_PROXY=127.0.0.1:9999
export HTTPS_PROXY=127.0.0.1:9999
```

3. Start Grafana

### Limitations

* Enable the `Skip TLS Verify` option in the data source config if the target API protocol is HTTPS.
* Only queries with **absolute time ranges** should be used with the proxy. Relative time ranges are not supported in the default matcher.

### Commands

#### Append mode

Append mode should be used to record interactions for any new tests. It will record requests and responses for any requests that haven't been seen before, and return recorded responses for any requests that match previously recorded interactions.

```
mage e2e:append <host:port> <path>
```

#### Overwrite mode

Overwrite mode should be used if previously recorded interactions need to be replaced with new data.

```
mage e2e:overwrite <host:port> <path>
```

#### Replay mode

Replay mode should be used in CI or locally if only playback of recorded data is needed. Replay mode will return recorded responses for any matching requests, and pass any requests that don't match recorded interactions to the target API.

```
mage e2e:replay <host:port> <path>
```