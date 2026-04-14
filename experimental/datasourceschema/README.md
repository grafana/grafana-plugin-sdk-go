# Datasource Schema Generation

This experimental package exposes Mage targets for generating datasource schema
artifacts from a backend datasource plugin.

Run these commands from your plugin root.

## Generate OpenAPI

`mage datasource:generateOpenAPI`

This command writes `spec.v0alpha1.openapi.json` to the plugin root. Extraction
warnings are printed to `stderr`.

## Generate Query Types

`mage datasource:generateQueryTypes`

This command writes `spec.v0alpha1.query.types.json` to the plugin root.
Extraction warnings are printed to `stderr`.
