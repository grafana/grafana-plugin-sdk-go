# Developer guide

This guide helps you get started developing Grafana Plugin SDK for Go.

## Tooling

Make sure you have the following tools installed before setting up your developer environment:

- [Git](https://git-scm.com/)
- [Go](https://golang.org/dl/) (see [go.mod](../go.mod#L3) for minimum required version)
- [Mage](https://magefile.org/)

## Building

We use [Mage](https://magefile.org/) as our primary tool for development related tasks like building and testing etc. It should be run from the root of this repository.

List available Mage targets that are available:

```bash
mage -l
```

You can use the `build` target to verify all code compiles. It doesn't output any binary though.

```bash
mage -v build
```

The `-v` flag can be used to show verbose output when running Mage targets.

### Testing

```bash
mage test
```

### Linting

```bash
mage lint
```

### Generate Go code for Protobuf definitions

A prerequisite is to have [Buf CLI](https://buf.build/docs/installation) installed and available in your path.

To install protoc-gen-go (version should automatically be taken from the go.mod):

```shell
go install google.golang.org/protobuf/cmd/protoc-gen-go
```

To install protoc-gen-go-grpc (latest version as of this writing is v1.3.0):
```shell
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

To compile the protobuf:

```shell
mage protobuf
# or
mage protobuf:generate
```

To verify no breaking changes Protobuf definitions compared with latest commit in main:
```shell
mage protobuf:validate
```

### Changing `generic_*.go` files in the `data` package

Currently [genny](https://github.com/cheekybits/genny) is used for generating some go code. If you make changes to generic template files then `genny` needs to be installed, and then `mage dataGenerate`. Changed generated files should be committed with the change in the template files.

### Dependency management

We use Go modules for managing Go dependencies. After you've updated/modified modules dependencies, please run `go mod tidy` to cleanup dependencies.

## Releasing

If you want to create a new version of the SDK for release, follow these steps:

- Make sure that you have `gorelease` installed
   - If not, run `go install golang.org/x/exp/cmd/gorelease@latest`
- Checkout the commit you want to tag (`git checkout <COMMIT_SHA>`)
- Run [`gorelease`](https://pkg.go.dev/golang.org/x/exp/cmd/gorelease) to compare with the previous release. For example, when preparing to release v0.123.0:

```
gorelease -base v0.122.0 -version v0.123.0
github.com/grafana/grafana-plugin-sdk-go/backend/gtime
------------------------------------------------------
Compatible changes:
- package added

v0.123.0 is a valid semantic version for this release.
```

- Run `git tag <VERSION>` (For example **v0.123.0**)
  - NOTE: We're using Lightweight Tags, so no other options are required
- Run `git push origin <VERSION>`
- Verify that the tag was create successfully [here](https://github.com/grafana/grafana-plugin-sdk-go/tags)
- Create a release from the tag on GitHub.
  - Use the tag name as title.
  - Click on the _Auto-generate release notes_ button.
  - Add a compatibility section and add the output of the command above.

**Release notes example:**

- Title: v0.123.0
- Content:

````md
<!-- Auto generated release notes -->

## Compatibility
```
gorelease -base v0.122.0 -version v0.123.0
github.com/grafana/grafana-plugin-sdk-go/backend/gtime
------------------------------------------------------
Compatible changes:
- package added

v0.123.0 is a valid semantic version for this release.
```

````
