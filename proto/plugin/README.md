# grafana.plugin.v3 (experimental)

A modern, self-contained protobuf module that lives alongside the legacy
`pluginv2` (`backend.proto`) contract. It exists to explore a plugin API that
exposes the generated gRPC interfaces directly, rather than wrapping every
protobuf message in a hand-written SDK-native Go type as the legacy contract
does. The intent is for this `grafana.plugin.v3` package to eventually replace
most of that older contract.

## Services

- `RouteService` — an HTTP-over-gRPC service ported from the legacy `Resource`
  service (`CallResource` → `CallRoute`), without a `PluginContext`.
- `AdmissionControlService` and `ResourceConversionService` — ported from the
  legacy services of the same purpose, but **without a `PluginContext`** in
  their requests, and renamed to satisfy the buf `STANDARD` lint rules
  (`Service` suffix, per-RPC `<Method>Request`/`<Method>Response` messages).

## Conventions

- **Protobuf Editions 2024**, the newest edition supported by the pinned
  toolchain. Note that edition 2024 makes protoc-gen-go emit the **Opaque API**
  (hidden fields with generated getters/setters/builders) rather than plain
  struct literals.
- **Versioned, dotted package** `grafana.plugin.v3`, with the files laid out in
  a directory that matches the package (`grafana/plugin/v3/`).
- **buf v2** tooling with `STANDARD` lint and `FILE` breaking-change checks.
- **buf managed mode** derives the Go import path, so `go_package` is not
  hardcoded in the `.proto` files.

## Generating

Requires `buf`, `protoc-gen-go`, and `protoc-gen-go-grpc` on your `PATH`:

```sh
go install github.com/bufbuild/buf/cmd/buf@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Then, from this directory:

```sh
buf lint
buf generate
```

Output is written to `genproto/grafana/plugin/v3/` at the repository root
(Go package `pluginv3`).
