# Datasource MCP POC - Design

## Background

Grafana datasource plugins expose runtime capabilities (query handling, `CallResourceHandler`, health checks) via gRPC, but those capabilities are not discoverable in a structured way for agents. Today, agent integrations either hardcode datasource-specific knowledge or rely on bespoke assistant tooling - that does not scale across the plugin ecosystem.

A separate effort recently added static schema files for datasource plugins. Plugins ship them in their archive at:

```
dist/schema/{apiVersion}/query.types.{json|yaml}
dist/schema/{apiVersion}/query.examples.{json|yaml}
dist/schema/{apiVersion}/settings.{json|yaml}
dist/schema/{apiVersion}/settings.examples.{json|yaml}
dist/schema/{apiVersion}/routes.{json|yaml}
```

These are loaded via `experimental/pluginschema.NewCompositeFileSchemaProvider` into a `PluginSchema` struct (see `grafana-plugin-sdk-go` PR #1533). The github-datasource PR #291 wires the schema-builder for query types and examples.

This design extends that work: the plugin SDK gains an embedded MCP server that translates the existing schema and runtime handlers into MCP `tools` and `resources`. Each plugin process exposes its own MCP endpoint on a local port. Plugins keep one source of truth (their schema files plus their existing handlers).

## Problem

Agents cannot reliably use datasource plugins without bespoke integration code because they do not know which capabilities a plugin exposes, what parameters those capabilities need, what query language is expected or where to find datasource-specific examples. This duplicates implementation work and limits agent support to a small set of hand-integrated datasources.

## POC scope

The POC proves the embedded MCP layer end-to-end at the protocol surface. It does **not** include live agent integration, MetaMCP, Grafana-side proxying or auth.

In scope:

- New `experimental/mcp` package in `grafana-plugin-sdk-go`
- One MCP HTTP listener per plugin process, lifecycle-managed by the SDK
- Schema-driven registration helpers covering: query types, query examples, `/resources/*` routes, health checks
- Code-level registration for custom tools, custom resources and prompts (no schema source)
- POC implementation in `github-datasource` and `redshift-datasource`
- Verification via a CLI MCP client (e.g. `mcp-inspector`) against the running plugin processes

Out of scope (deferred):

- Settings and settings-examples schema mapping (settings as a resource is not part of this POC)
- Grafana-side aggregation or proxy of plugin MCP endpoints
- MetaMCP integration, auth, multi-tenancy
- Live agent (Claude/Cursor) end-to-end demos
- Overlay files for MCP-specific descriptions or hide-lists (`mcp.json`)
- Schema-driven prompts (`prompts.json`)
- Tool-level call-order enforcement
- App plugins (datasource only)
- Streaming query results over MCP (single-shot only)
- `/proxy/*` routes (skipped - not meaningful as agent tools)

## Goals

1. Each datasource plugin can expose tools, resources and prompts over MCP without duplicating its existing handlers
2. The existing `pluginschema` files are the only authored source of truth for query types, query examples and resource routes
3. Plugin authors can layer custom tools, custom resources and prompts via code-level registration
4. The MCP server's lifecycle is owned by the SDK and follows the existing plugin process lifecycle
5. The MCP listener exposes the standard MCP protocol so any MCP client can connect

## Non-goals

- Replacing or refactoring the existing plugin gRPC runtime
- Requiring every plugin to adopt MCP
- Designing a new packaging format
- Normalizing query languages across datasources
- Solving auth, policy or multi-tenancy in the plugin (deferred to a gateway)

## Architecture

```
+--------------------------------------------------------------------+
|  Plugin process (single OS process)                                |
|                                                                    |
|  +----------------------+        +-----------------------------+   |
|  |  gRPC server         |        |  MCP server (HTTP)          |   |
|  |  (existing)          |        |  (new, this POC)            |   |
|  |                      |        |                             |   |
|  |  - QueryData         |        |  - tools/list, tools/call   |   |
|  |  - CallResource      |        |  - resources/list, read     |   |
|  |  - CheckHealth       |        |  - prompts/list, get        |   |
|  +-----------+----------+        +-------------+---------------+   |
|              |                                 |                   |
|              v                                 v                   |
|  +---------------------------------------------------------------+ |
|  |  Plugin handlers (QueryDataHandler, CallResourceHandler, ...) | |
|  +---------------------------------------------------------------+ |
|                                                                    |
+-----------+-----------------------------------+--------------------+
            |                                   |
            v unix socket (Grafana <-> plugin)  v TCP host:port
       +---------+                        +-----------------+
       | Grafana |                        | MCP client      |
       +---------+                        | (inspector,     |
                                          |  MetaMCP, ...)  |
                                          +-----------------+
```

Key properties:

- One process, two listeners. The MCP server runs in-process alongside the existing gRPC server.
- MCP tool calls re-use the same handler implementations as gRPC (`QueryDataHandler`, `CallResourceHandler`, `CheckHealthHandler`). No duplicate logic.
- The plugin's `main.go` is the explicit registration point - it constructs the MCP server, calls schema-driven helpers, and registers any custom additions.
- The MCP listener binds to `127.0.0.1` by default. Auth and policy belong to a gateway that sits in front of the listener; the plugin trusts whatever reaches it.

## SDK package layout

A new `experimental/mcp` package in `grafana-plugin-sdk-go`:

```
experimental/mcp/
├── server.go          # Server type, Start/Stop, transport wiring
├── server_test.go
├── tools.go           # Tool type + registration helpers
├── resources.go       # Resource type + registration helpers
├── prompts.go         # Prompt type + registration helpers
├── fromschema/
│   ├── querytypes.go  # PluginSchema.QueryTypes -> []Tool
│   ├── routes.go      # PluginSchema.Routes -> []Tool
│   ├── examples.go    # PluginSchema.QueryExamples -> Resource + tool examples
│   └── healthcheck.go # CheckHealthHandler -> Tool
└── mcptest/
    └── client.go      # In-memory MCP client for unit tests
```

The package depends on `github.com/modelcontextprotocol/go-sdk` for protocol primitives. The Grafana-specific glue lives here: handler binding, schema walking, lifecycle integration.

`mcp.Server` has a small surface (`Register*`, `Bind*`, `Start`, `Shutdown`). The schema-walking helpers in `fromschema` are opt-in and don't pollute the core API. A plugin that wants pure code-level registration ignores `fromschema` entirely.

### Lifecycle integration

`backend/datasource.ManageOpts` gains:

```go
type ManageOpts struct {
    // ... existing fields ...
    MCPServer *mcp.Server // optional
}
```

When set, the SDK starts the MCP listener after the gRPC server is up, before `Manage` blocks. On `SIGINT`/`SIGTERM`, the SDK calls `mcpServer.Shutdown(ctx)` with a 5-second drain before the gRPC server stops.

If MCP startup fails (e.g. port in use), the plugin logs the error but does **not** exit - gRPC continues serving. MCP is opt-in and must not take a plugin down.

## Plugin author API

The shape of `main.go` for a plugin that opts in:

```go
package main

import (
    "embed"
    "os"

    "github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
    "github.com/grafana/grafana-plugin-sdk-go/backend/log"
    "github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
    "github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/fromschema"
    "github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"

    ghds "github.com/grafana/github-datasource/pkg/github"
)

//go:embed schema/v0alpha1/*.json
var schemaFS embed.FS

func main() {
    ds := ghds.NewDatasource(...)

    schema, err := pluginschema.NewCompositeFileSchemaProvider(schemaFS).Get("v0alpha1")
    if err != nil {
        log.DefaultLogger.Error("schema load failed", "err", err)
        os.Exit(1)
    }

    mcpServer := mcp.NewServer(mcp.ServerOpts{
        Name:    "grafana-github-datasource",
        Version: "1.0.0",
    })

    // bind handlers - these are how MCP tool calls reach the plugin
    mcpServer.BindQueryDataHandler(ds)
    mcpServer.BindCallResourceHandler(ds)
    mcpServer.BindCheckHealthHandler(ds)

    // schema-driven registration
    fromschema.RegisterQueryTools(mcpServer, schema)
    fromschema.RegisterRouteTools(mcpServer, schema)
    fromschema.RegisterQueryExamples(mcpServer, schema)
    fromschema.RegisterHealthCheckTool(mcpServer)

    // custom code-level additions
    mcpServer.RegisterPrompt(mcp.Prompt{
        Name:        "investigate-pull-requests",
        Description: "Walk through a recent PR investigation",
        Template:    investigatePRTemplate,
    })

    if err := datasource.Manage(
        "grafana-github-datasource",
        ds.NewInstance,
        datasource.ManageOpts{MCPServer: mcpServer},
    ); err != nil {
        log.DefaultLogger.Error("plugin exited", "err", err)
        os.Exit(1)
    }
}
```

### Design decisions

**Bind handlers, then register tools.** `BindQueryDataHandler` etc. don't register anything by themselves. They tell the MCP server *how to execute* tools that come from `QueryTypeDefinition`s. The `fromschema` helpers reuse the bound handler when generating tools. This separates "what to expose" (registration) from "how to execute" (handler binding).

**`fromschema` is a separate sub-package, not on the server.** Keeps the core API small and lets plugins skip it when they want full code-level control.

**Tool naming is deterministic.** `query_<discriminator-value>` for query types, `<method>_<sanitized-path>` for route tools, `check_health` for health. MCP clients can rely on these names across plugin versions.

**Tool execution path.** When MCP calls `query_Pull_Requests`, the SDK constructs a `backend.QueryDataRequest` from the validated tool args, invokes the bound `QueryDataHandler`, then encodes the resulting `data.Frames` to JSON for the tool output.

**Routes-as-tools execution path.** When MCP calls a route tool, the SDK constructs a synthetic `backend.CallResourceRequest` (path with params substituted, method, query string from remaining args, JSON body from `requestBody` arg) and invokes the bound `CallResourceHandler`. The first response chunk's body is decoded (JSON if `application/json`, else raw text) and returned.

## Schema -> MCP mapping

| Source | MCP primitive | Name pattern | Notes |
|---|---|---|---|
| `query.types.json` items | Tool | `query_<discriminator-value>` | InputSchema = the spec's JSON Schema, ungrafted |
| `routes.json` paths (`/resources/*` only) | Tool | `<method>_<sanitized-path>` | InputSchema derived from OpenAPI parameters + requestBody; `/proxy/*` skipped |
| `CheckHealthHandler` | Tool | `check_health` | Empty input schema; output is the health JSON |
| `query.examples.json` | Resource | `examples://query` | Single resource listing all examples; also attached as `Tool.examples` per matching tool |
| code-only | Tool / Resource / Prompt | author-supplied | Plugin calls `Register*` directly |

### Query types -> tools

```
QueryTypeDefinition.metadata.name = "Pull_Requests"
QueryTypeDefinition.spec.discriminators[].field = "queryType"
QueryTypeDefinition.spec.discriminators[].value = "Pull_Requests"
QueryTypeDefinition.spec.schema = JSON Schema
                                                ↓
mcp.Tool{
    Name:        "query_Pull_Requests",
    Description: schema.description (fallback: "Query " + name),
    InputSchema: schema,
    Annotations: {"queryType": "Pull_Requests"},
    Handler:     unifiedQueryToolHandler,  // shared across all query tools
}
```

The shared handler:

1. Validates input against the tool's JSON Schema
2. Wraps it in a `backend.DataQuery` with `RefID="A"`, `QueryType` from the discriminator, JSON body = the input
3. Calls the bound `QueryDataHandler.QueryData(ctx, ...)`
4. Encodes the resulting `data.Frames` to JSON and returns as MCP tool output

### Routes -> tools

```
Path "/labels", method GET, parameters [owner, repository, query]
                ↓
mcp.Tool{
    Name:        "get_labels",
    Description: operation.summary or operation.description,
    InputSchema: derived from OpenAPI parameters + requestBody,
    Handler:     callResourceHandler("/labels", "GET"),
}
```

The handler builds a `backend.CallResourceRequest`:

- `Path` = OpenAPI path with path-params substituted
- `Method` = OpenAPI method
- `URL.Query` = remaining args
- `Body` = JSON of `requestBody` arg if any

Path sanitization: `/labels` -> `labels`, `/repos/{owner}/{repo}/files` -> `repos_files` (path params dropped from the name; their values come from tool args).

### Query examples -> resource + tool examples

For each `QueryExample`:

- Attach as an entry in the `examples` field of the tool whose `queryType` matches
- Also publish a single resource `examples://query` whose body is the full examples list - useful for one-shot agent context

### Things intentionally not auto-derived

- Tool descriptions richer than what's in the JSON Schema. Plugins can override after registration via `mcpServer.UpdateTool(name, ...)` (escape hatch).
- Cross-tool dependencies / call-order. Plugins can hint via Annotations; no enforcement in v1.
- Tool aliases / hidden tools. Plugins that want a curated subset use code-level registration directly instead of `fromschema`.

## Transport and lifecycle

**Transport**: Streamable HTTP (the current MCP HTTP transport spec). SSE used for server-to-client streaming where the official Go SDK requires it. Stdio not used - the plugin process is long-running and already owns stdin/stdout for hashicorp go-plugin handshake.

**Address resolution** (priority order):

1. `GF_PLUGIN_MCP_ADDR` env var (`":7401"`, `"127.0.0.1:7401"`, or `"0.0.0.0:7401"`)
2. `mcp.ServerOpts.Addr` if set in code
3. Auto-pick a free port on `127.0.0.1`

**Port advertisement**: when the port is auto-picked, the SDK writes `dist/mcp.addr` (sibling of the existing `dist/standalone.txt` debug file), containing `host:port\n`. Tooling and tests read this to find the live MCP endpoint.

**Auth**: none in the POC. Listener binds to `127.0.0.1` by default. Per the upstream design, auth and policy belong to a gateway (e.g. MetaMCP) and are out of scope.

## POC implementation

### `grafana-plugin-sdk-go`

- New `experimental/mcp` package (server, tools, resources, prompts)
- New `experimental/mcp/fromschema` sub-package (query, route, examples, healthcheck walkers)
- New `experimental/mcp/mcptest` (in-memory client for tests)
- Extend `backend/datasource.ManageOpts` with `MCPServer *mcp.Server`
- Extend the underlying `backend.ServeOpts` plumbing in `Manage` to wire MCP startup/shutdown
- Unit tests: each `fromschema` walker, the bind/handler paths, lifecycle (start, shutdown, error-on-startup)

### `github-datasource`

- Pull in (or rebase) PR #291 to get `src/schema/v0alpha1/query.types.json` and `query.examples.json` up to date with the current SDK
- Author `routes.json` from the existing `resource_handlers.go` routes (`/labels`, `/milestones`)
- Update `pkg/main.go` to construct an `mcp.Server`, call the `fromschema` helpers, register one example custom prompt and pass it to `datasource.Manage`
- Embed schema files via `embed.FS`

### `redshift-datasource`

- Author schema files in `src/schema/v0alpha1/` from scratch:
  - `query.types.json` (the SQL query type, generated via `schemabuilder` like github did)
  - `query.examples.json` (a small set of sample SQL queries)
  - `routes.json` (`/secrets`, `/secret`, `/clusters`, `/workgroups` plus the SQL default routes - need to enumerate which of those make sense as agent tools)
- Same `main.go` updates as github-datasource

### `grafana/grafana`

No changes for the POC. Per the narrowed scope, agent integration is deferred. The MCP listener is reachable directly by any MCP client. Grafana-side discovery and proxying is a follow-up effort.

## Verification

1. Build and run `github-datasource` and `redshift-datasource` standalone (existing magefile target)
2. Read the address printed in logs or stored in `dist/mcp.addr`
3. Connect with `mcp-inspector` (or curl) and verify:
   - `tools/list` returns the expected set: one tool per query type, one tool per `/resources/*` route, plus `check_health`
   - `tools/call` for a query-type tool returns real data with valid credentials configured
   - `tools/call` for a route tool returns the same response as the equivalent HTTP `CallResource` request
   - `resources/list` returns `examples://query`
   - `prompts/list` returns the custom prompt registered in code
4. SDK unit tests using the in-memory `mcptest` client cover each path without needing a real plugin process

## Risks and mitigations

**Risk: schema files drift from runtime handlers.** A query type in `query.types.json` with no matching `QueryType` discriminator at runtime would surface as a tool that always errors.

Mitigation: at server startup, log a warning for any registered query tool whose discriminator does not appear in any `DataQuery.QueryType` the bound handler will accept. We don't have a way to enumerate handler-supported types automatically, so this stays as a soft warning rather than a hard fail.

**Risk: route tool input schemas are weak.** OpenAPI parameter definitions can be loose, leading to tool input schemas that are imprecise.

Mitigation: accept this for the POC. Plugins can override the generated tool with `mcpServer.UpdateTool(name, ...)` if they want a tighter schema.

**Risk: port collisions in dev when running multiple plugins.** Two plugins both auto-pick - usually fine, but if both honor the same env var the second crashes.

Mitigation: env var is opt-in and per-plugin. Auto-pick is the default. The `dist/mcp.addr` file makes the picked port discoverable.

**Risk: MCP listener exposed beyond `127.0.0.1`.** A plugin operator setting `0.0.0.0` for testing leaves the unauthenticated MCP endpoint reachable on the network.

Mitigation: SDK logs a warning when the bound address is non-loopback and there is no gateway in front. Hard enforcement is a gateway concern, not a plugin concern.

## Open questions

1. Do we want a way to mark a query type or route as MCP-hidden in the schema files? (Out of scope for the POC; a hide-list is mentioned as future work.)
2. Should query examples become MCP `Tool.examples` exclusively, or also remain a standalone resource? (POC does both - cheap to revisit.)
3. Should the SDK provide a `mcpServer.RegisterDefaults(schema)` convenience that calls all `fromschema` helpers in one shot? (Not for v1 - keeping the call sites explicit makes it obvious what's exposed.)

## Success criteria

This POC is successful if:

- A plugin built on the SDK can stand up an MCP endpoint by calling the new helpers, with no duplicate authoring beyond the existing `pluginschema` files
- Both `github-datasource` and `redshift-datasource` expose tools for every query type and every `/resources/*` route, plus `check_health`
- An off-the-shelf MCP client can list and call those tools and read the query examples resource
- Plugin authors can register custom tools, custom resources and prompts in code without touching schema files
- Nothing on the existing query/resource/health gRPC path changes behavior
