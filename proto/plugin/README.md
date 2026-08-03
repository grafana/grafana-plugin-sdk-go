# grafana.plugin.v3 (experimental)

A modern, self-contained protobuf module that lives alongside the legacy
`pluginv2` (`backend.proto`) contract. It exists to explore a plugin API that
exposes the generated gRPC interfaces directly, rather than wrapping every
protobuf message in a hand-written SDK-native Go type as the legacy contract
does. The intent is for this `grafana.plugin.v3` package to eventually replace
most of that older contract.

## Why v3?

This module is an answer to three questions:

- **What does a service without `PluginContext` look like?** The legacy
  contract threads a `PluginContext` (org/user/data-source settings, instance
  identity) through every request. The v3 services deliberately drop it: the
  RPCs here operate on self-describing objects (Kubernetes-style admission and
  conversion payloads, or plain HTTP-shaped route requests) and do not require
  the host to resolve and inject per-instance settings. Dropping
  `PluginContext` also removes the need for **instance management** — the SDK
  machinery that caches and reuses one plugin instance per unique context.

- **Can it run next to an existing plugin?** Yes. v3 is additive, not a
  replacement wired in behind the scenes. A single plugin binary can serve the
  legacy services *and* v3 at the same time by setting both the legacy
  `*Server` fields and the new `V3Server` field on
  [`grpcplugin.ServeOpts`](../../backend/grpcplugin/serve.go). Each is dispensed
  under its own go-plugin name, so hosts adopt v3 incrementally.

- **Can we skip the hand-crafted Go structs?** Yes. The V3 service contracts are
  the generated gRPC interfaces themselves — plugin authors implement
  `pluginv3.RouteServiceServer` (etc.) and work with the generated protobuf
  messages directly, with no SDK-native wrapper type in between. See the
  runnable [`pluginv3example`](../../experimental/pluginv3example) package for
  the intended plugin shape.

## Services

- `RouteService` — an HTTP-over-gRPC service ported from the legacy `Resource`
  service (`CallResource` → `CallRoute`)
- `AdmissionControlService` — admission hooks invoked when an object is created,
  updated, or deleted:
  - `ValidateObject` — accept or reject the object
  - `MutateObject` — return a modified copy of the object to store
- `ResourceConversionService` — convert objects from one API version to another

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
- **`require_unimplemented_servers=false`** in `buf.gen.yaml` keeps the
  generated `Unimplemented*Server` stubs optional to embed. Plugin authors embed
  them anyway (via [`grpcplugin.UnimplementedV3Server`](../../backend/grpcplugin/grpc_pluginv3.go))
  so unimplemented RPCs return `codes.Unimplemented` instead of failing to
  compile — letting a plugin serve only the RPCs it cares about.

## Generating

Requires `buf`, `protoc-gen-go`, and `protoc-gen-go-grpc` on your `PATH`:

```sh
go install github.com/bufbuild/buf/cmd/buf@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

From the repository root, `mage protobuf:generate` regenerates both this module
and the legacy `pluginv2` one. To run `buf` directly, do it from this directory,
since the `out` paths in `buf.gen.yaml` are relative to it:

```sh
buf lint
buf generate
```

Output is written to `genproto/grafana/plugin/v3/` at the repository root
(Go package `pluginv3`).
