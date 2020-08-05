# Grafana Plugin SDK for Go

This SDK enables building [Grafana](https://github.com/grafana/grafana) backend plugins using Go.

[![License](https://img.shields.io/github/license/grafana/grafana-plugin-sdk-go)](LICENSE)
[![Go.dev](https://pkg.go.dev/badge/github.com/grafana/grafana-plugin-sdk-go)](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go?tab=doc)
[![Go Report Card](https://goreportcard.com/badge/github.com/grafana/grafana-plugin-sdk-go)](https://goreportcard.com/report/github.com/grafana/grafana-plugin-sdk-go)
[![Circle CI](https://img.shields.io/circleci/build/gh/grafana/grafana-plugin-sdk-go/master)](https://circleci.com/gh/grafana/grafana-plugin-sdk-go?branch=master)

## Current state

This SDK is still in development. The protocol between the Grafana server and the plugin SDK is considered stable, but we might introduce breaking changes in the SDK. This means that plugins using the older SDK should work with Grafana, but might lose out on new features and capabilities that we introduce in the SDK.

## Contributing

If you're interested in contributing to this project:

- Start by reading the [Contributing guide](/CONTRIBUTING.md).
- Learn how to set up your local environment, in our [Developer guide](/contribute/developer-guide.md).

## License

[Apache 2.0 License](https://github.com/grafana/grafana-plugin-sdk-go/blob/master/LICENSE)
