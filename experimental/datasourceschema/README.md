# Datasource Schema Extractor

This experimental package contains the Go datasource schema extractor that was
previously maintained in `plugin-tools`.

## Mage entrypoint

The supported CLI entrypoint is the reusable Mage target implemented in
`build/datasource.go`.

That Mage target:

- defaults the plugin directory to `.`
- sets `GOTOOLCHAIN=auto` when it is not already configured
- calls `datasourceschema.GenerateOpenAPI(...)`
- writes warnings to `stderr`
- writes `spec.v0alpha1.openapi.json` in the plugin root

The target is exposed by the reusable `build` Mage namespace as:

`mage datasource:generateOpenAPI [path-to-plugin]`

The paired query type output is exposed as:

`mage datasource:generateQueryTypes [path-to-plugin]`

Those targets write the provider-compatible filenames in the plugin root:

- `spec.v0alpha1.openapi.json`
- `spec.v0alpha1.query.types.json`

## Programmatic API

Packages that want to invoke the extractor directly can call:

`datasourceschema.GenerateOpenAPI(datasourceschema.OpenAPIOptions{...})`

and

`datasourceschema.GenerateQueryTypes(datasourceschema.OpenAPIOptions{...})`

The public API is defined in `openapi.go`. `OpenAPIOptions` accepts the plugin
directory plus optional package patterns and build flags. Each result includes
the generated JSON payload and any extraction warnings.

## Behavior

The extractor analyzes a datasource backend implementation and generates:

- plugin OpenAPI settings output compatible with `experimental/pluginspec`
- datasource query type definitions as a separate JSON document

The default configuration analyzes `./...` under the provided directory.

The package is kept under `experimental` because the extractor behavior and API
surface are still evolving.
