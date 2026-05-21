# POC Findings

A running log of issues, surprises, and decisions encountered while embedding
an MCP server inside Grafana datasource plugins (github-datasource,
redshift-datasource) via the SDK package at
`grafana-plugin-sdk-go/experimental/mcp`.

Each entry: what was observed, the root cause, and the workaround/fix.

---

## 1. Anthropic API rejects tool schemas as not draft 2020-12

**Observed.** Calling tools through Claude as the MCP client returned:

```
API Error: 400 tools.8.custom.input_schema: JSON schema is invalid.
It must match JSON Schema draft 2020-12 (https://json-schema.org/draft/2020-12).
```

Only one tool index was reported, even though the issue affected every tool.

**Root cause.** The plugin schema files (e.g.
`github-datasource/pkg/schema/v0alpha1/query.types.json`) declare each query
type's schema with:

```json
"$schema": "https://json-schema.org/draft-04/schema"
```

Anthropic's tool-use API strictly validates input schemas against
[JSON Schema draft 2020-12](https://json-schema.org/draft/2020-12). The
draft-04 `$schema` URI alone is enough to fail validation, regardless of
whether the rest of the schema body is structurally compatible. The API
stops at the first failing tool, which is why only `tools.8` was reported —
all 20 query tools had the same problem.

Beyond `$schema`, draft-04 also differs from 2020-12 in a few other ways
that can appear in legacy plugin schemas:

- `id` was renamed to `$id`.
- `exclusiveMinimum` / `exclusiveMaximum` were booleans modifying
  `minimum`/`maximum`; in 2020-12 they are numbers that replace them.
- `definitions` was renamed to `$defs` (not yet seen in our schemas, but
  worth noting for future plugins).

**Fix.** Added `normalizeJSONSchema` to `experimental/mcp/server.go`. Every
tool's `InputSchema` is normalized before being marshaled and registered
with the underlying MCP SDK:

- Recursively strip `$schema` and `id`.
- Convert boolean `exclusiveMinimum` / `exclusiveMaximum` into the
  2020-12 numeric form (or drop them when the boolean is `false`).
- Walk nested `properties`, arrays, and sub-schemas without mutating the
  caller's map.

This keeps the normalization concern in the SDK so every plugin embedding
the MCP server gets the fix without touching its schema files.

---

## 2. Authentication is a quick-and-dirty hack — needs a proper solution

**Observed.** The MCP HTTP endpoint on `127.0.0.1:7401` (and `:7402` for
redshift) has no authentication. Any process on the host that can reach
loopback can call any registered tool. Tool calls reuse cached
credentials that were captured from an earlier Grafana → plugin gRPC
call, with no link between the calling MCP client's identity and the
credentials being used.

**Root cause / mechanism.** Plugins receive credentials via
`backend.PluginContext.DataSourceInstanceSettings` on every gRPC request
from Grafana. Grafana decrypts secrets per-request and passes them in.
MCP tool calls don't originate from Grafana, so there's no
PluginContext on the request — yet the bound handlers
(`QueryData`/`CallResource`/`CheckHealth`) need one to talk to the
upstream API.

To bridge that gap, the SDK uses a "context capture" hack in
`backend/datasource/manage.go`:

- `contextCapture` wraps the plugin handler and intercepts every gRPC
  call from Grafana, stashing `req.PluginContext` in a map keyed by
  datasource UID (`mcp.Server.RegisterPluginContext`).
- When an MCP tool call arrives on the HTTP endpoint, the server looks
  up the cached PluginContext by `datasource_uid` (or auto-picks the
  only one, if a single datasource is registered) and uses it to
  build the gRPC-style request the bound handler expects.
- The MCP HTTP server itself accepts every request — no bearer token,
  no Grafana session cookie, no mTLS. The `server.go:194` warning
  ("MCP listener bound to non-loopback address; auth must be handled
  by a gateway") is the explicit acknowledgement of this.

**Risks of the current approach.**

- Credentials are reused across MCP callers. A PluginContext stashed
  from a request made by user A will be reused for an MCP call from
  any local client, regardless of who they claim to be.
- Cached secrets can go stale (rotation, revocation, datasource
  reconfiguration). The cache is only refreshed when Grafana next
  calls the plugin via gRPC.
- "Loopback-only" is the only access control. Anything with local
  access — sidecar containers, other plugins, dev tools — can call
  every tool with the cached credentials of every registered
  datasource. Not safe for shared/multi-tenant hosts.
- Tools are unconditionally callable before Grafana has ever made a
  gRPC request: in that state the lookup fails with "no datasource
  instance registered yet", but the endpoint itself is still open.

**Long-term fix (TBD).** Options worth exploring, roughly ordered by
how invasive they are:

1. **Per-request bearer token.** Grafana mints a short-lived token tied
   to a (user, datasource) pair, hands it to the MCP client, the MCP
   server validates it against Grafana on every call. Closes the
   "any local process" hole and binds credentials to the actual
   caller.
2. **Grafana-mediated MCP.** Grafana exposes the MCP endpoint, handles
   auth at its existing boundary, and forwards into the plugin via
   the existing gRPC channel — so PluginContext is built fresh per
   request, the way Grafana already does for regular plugin calls.
   Eliminates the cache entirely.
3. **mTLS between Grafana and plugin's MCP endpoint.** Lower-level
   transport auth; doesn't solve per-user credential scoping but does
   close the "any local process" hole.

For the POC, option 0 (loopback + cached PluginContext) is fine —
flagging here so we don't ship it.

---

## 3. MCP runs on its own HTTP port — chosen for speed, not as the end state

**Context.** When planning the POC we considered two ways to expose
MCP from a backend datasource plugin:

1. **Extend the existing plugin gRPC contract.** Add MCP-shaped RPCs
   (list tools, call tool, read resource, get prompt) alongside the
   existing `QueryData` / `CallResource` / `CheckHealth` methods.
   Grafana would terminate MCP at its own boundary and forward into
   the plugin via gRPC, the same way it handles every other plugin
   request today.
2. **Run a standalone HTTP server inside the plugin process.** The
   plugin binary opens its own listener on a port (currently
   `127.0.0.1:7401` for github, `:7402` for redshift) and speaks the
   MCP HTTP transport directly to clients.

**Decision.** We went with option 2 for the POC, because it lets us
iterate entirely inside the plugin SDK and the plugin binaries
without changes on the Grafana side. We can ship a working,
testable MCP server today instead of waiting on a contract change.

**Trade-offs we're accepting for now.**

- **Port management is awkward.** Plugins can't read environment
  variables, so the port has to be hardcoded in each plugin's
  `main.go`. We hand out ports manually (`:7401`, `:7402`, …) and
  have no allocation story for the general case.
- **Auth has nowhere natural to live.** See finding #2 — because
  the MCP endpoint isn't behind Grafana, there's no Grafana session
  / user identity on the request, and we resort to caching
  PluginContext from the gRPC side.
- **Discovery is ad-hoc.** External clients have to know the port.
  The SDK writes `dist/mcp.addr` next to the plugin so tools can
  find it, but that's a workaround, not a real discovery
  mechanism.
- **Two transports to maintain.** The plugin now has both gRPC
  (Grafana → plugin) and HTTP (MCP client → plugin) running side
  by side, with their own lifecycle, error paths, and logging.

**Long-term direction.** Option 1 (gRPC contract) is the more likely
end state. It puts MCP behind Grafana's existing auth and routing,
removes the port/discovery problem, and lets PluginContext be built
fresh per request rather than cached. The cost is a contract change
on the Grafana side and a longer path to a first working version —
which is exactly the cost we wanted to defer for the POC.

---

## 4. Bundled datasource plugins don't fit the current SDK entry point

**Context.** The next plugins we want MCP support in are Loki,
Prometheus, and MySQL. Unlike github-datasource and
redshift-datasource, these ship bundled with Grafana — they live
under `grafana/pkg/tsdb/{loki,prometheus,mysql}` and are linked
directly into the Grafana binary rather than running as separate
plugin processes.

**Why this breaks the current wiring.** The SDK exposes MCP through
`MCPServer` on `datasource.ManageOpts` in
`backend/datasource/manage.go`. `Manage()` is what an external
plugin's `main.go` calls to stand up its gRPC server, and it's also
where the embedded MCP listener is started and where the
`contextCapture` PluginContext-stashing hack lives. Bundled plugins
never go through `Manage()` — they're registered into Grafana's
in-process plugin registry as Go services. There's no `ManageOpts`
to hang an `MCPServer` off, no separate process to host the
listener, and no gRPC boundary to capture PluginContext from in the
first place.

**Why the standalone-HTTP shape is also wrong for bundled plugins.**
Even if we plumbed `mcp.Server` directly into each bundled plugin's
service constructor, three loopback ports inside one Grafana
process gives us none of the (already weak) isolation benefit of
the per-plugin-process model, makes the auth story worse rather
than better (everything is inside Grafana's auth boundary already,
but the MCP listener sits outside it), and multiplies the
port/discovery problems from finding #3.

**Options, ordered by how invasive they are.**

1. **One shared MCP server owned by Grafana; bundled plugins
   register into it.** Grafana stands up a single `mcp.Server` as a
   service alongside its HTTP server. Each bundled plugin's
   `ProvideService` calls the existing
   `BindQueryData`/`BindCallResource`/`BindCheckHealth` against
   that shared server, namespacing its tools by datasource type.
   Requires exposing a public construct/start/stop surface on
   `mcp.Server` that doesn't go through `datasource.Manage`, and
   moving the auth boundary onto Grafana's existing middleware
   (which also lets us drop the cached-PluginContext hack — Grafana
   has the real one in-process). This is the natural convergence
   point with finding #2 option 2 and finding #3 option 1.
2. **gRPC contract first, bundled second.** If we land the
   gRPC-contract direction from finding #3 before tackling bundled
   plugins, bundled plugins ride along for free: Grafana terminates
   MCP at its boundary and dispatches to whichever in-process or
   out-of-process plugin owns the tool. Bundled plugins just need
   to implement the new contract methods. Strictly better than
   option 1 long-term, but blocked on the same contract change.
3. **Unbundle Loki/Prometheus/MySQL.** Run them as external plugin
   processes again so `datasource.Manage()` works as-is. Clean from
   an SDK perspective, but a multi-quarter project and orthogonal
   to MCP — not realistic as a way to extend the POC.

**Recommendation for the next POC step.** Option 1: extract a
Grafana-ownable surface from `mcp.Server` (constructor + lifecycle
that doesn't assume `Manage()` is the caller), then wire it into
Grafana's service graph and have one bundled plugin (Loki is
probably the most useful) bind its handlers to it. That validates
both the shared-server model and the Grafana-side auth path before
we commit to a contract change.

---

## 5. grafana-assistant only consumes MCP tools, not resources

**Observed.** When connecting our embedded MCP servers to
grafana-assistant, only the registered tools were picked up and
made available to the model. Resources registered via
`mcp.Server.RegisterResource` (see `experimental/mcp/resources.go`)
are exposed correctly over the MCP transport — `resources/list`
and `resources/read` work when probed with a generic MCP client —
but grafana-assistant never lists or reads them.

**Implication.** Any datasource context we'd want to surface as a
read-only, addressable blob (schema dumps, dashboards, saved
queries, docs snippets) has to be modeled as a tool call rather
than as an MCP resource if we want grafana-assistant to actually
use it. That changes the design trade-off: resources are the more
natural MCP shape for "here is some context, read it if you need
it", but right now they're effectively dead weight for the
assistant integration.

**Open questions.**

- Is this a deliberate choice on the grafana-assistant side
  (tools-only by design), or a not-yet-implemented capability?
- If it's the former, should the SDK keep encouraging resource
  registration at all for the assistant use case, or steer
  plugins toward tool-shaped equivalents?
- Prompts (`experimental/mcp/prompts.go`) are in the same boat —
  worth confirming whether they're consumed either.

**Action.** Flagging with the grafana-assistant team to confirm
intent and timeline. Until then, plugin authors targeting the
assistant should prefer tools over resources for any context they
want the model to reach.

---

## 6. Anthropic API rejects property keys containing `[` or `]`

**Observed.** With the `mcp-from-openapi` server (port 7501) wired
into grafana-assistant as the `datasource-information` built-in
provider, every chat turn failed. The UI showed the generic:

> There was an issue with your request. Please try again or contact
> support if the problem persists.

The assistant API logs had the real error:

```
POST "https://api.anthropic.com/v1/messages": 400 Bad Request
tools.27.custom.input_schema.properties: Property keys should match
pattern '^[a-zA-Z0-9_.-]{1,64}$'
```

The generic UI message comes from the assistant frontend's
`createUserFriendlyErrorMessage` mapping any `bad_request`/`400`
to that string
(`apps/plugin/src/utils/errorMessages.ts`), so the underlying cause
is only visible in the API container logs.

**Root cause.** Anthropic's tool-use API enforces a stricter
property-key pattern than JSON Schema itself does: keys in
`input_schema.properties` must match `^[a-zA-Z0-9_.-]{1,64}$`. The
OpenAPI→MCP generator preserves the wire-level parameter name
verbatim, so Prometheus/Loki endpoints that take a repeated
`match[]` query parameter end up with a property literally named
`match[]`. The `[` and `]` characters are rejected, and Anthropic
fails the entire `messages` request at the first offending tool
(tool 27 in our case — only one is reported even though four tools
share the same problem).

Tools affected in the current `mcp-from-openapi` output:

- `loki_proxy_get`
- `prometheus_series`
- `prometheus_labels`
- `prometheus_label_values`

**Fix.** Rename the JSON property to `match` (no brackets) in the
generated `inputSchema.properties`, keep `type: array`, and mention
the wire-level `match[]` name in the property description if it
matters for the proxy step. The `[]` suffix is a Prometheus
query-string serialization convention for repeated parameters, not
a JSON property-naming convention — JSON Schema already expresses
the multi-value semantics via `type: array`.

If other OpenAPI specs we ingest carry parameter names with
characters outside `[a-zA-Z0-9_.-]` (spaces, colons, slashes,
parentheses, names longer than 64 chars, …), the generator should
sanitize them at the same point: rewrite the key, remember the
original on the side, and translate back when constructing the
outgoing HTTP request. Doing this in the generator keeps the
assistant API free of provider-specific tool-schema patching.

**Related.** Same general shape as finding #1 (Anthropic enforces
constraints beyond plain JSON Schema). The mitigation pattern is
the same: normalize tool schemas at registration time so that
upstream tool authors don't have to know Anthropic's exact rules.
