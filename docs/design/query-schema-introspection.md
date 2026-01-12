# Query Schema Introspection for AI Agents

## Background

The Grafana Plugin SDK enables backend plugins to handle data queries via the [`QueryDataHandler`](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go/backend@v0.284.0#QueryDataHandler) interface. Each query contains a [`DataQuery`](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go/backend@v0.284.0#DataQuery) struct with a `JSON json.RawMessage` field that holds datasource-specific query parameters. This JSON structure varies entirely between datasources—Prometheus queries look different from MySQL queries, which look different from Elasticsearch queries, etc.

The query flow works as follows:
1. Grafana sends a `QueryDataRequest` containing one or more `DataQuery` objects to the plugin via gRPC
2. The plugin unmarshals the `JSON` field into its own query model struct
3. The plugin executes the query and returns `DataResponse` frames

There is currently no mechanism for external systems to discover the expected structure of the `JSON` field for a given datasource.

## Problem

Grafana is adding AI capabilities that allow agents to execute data queries on behalf of users. For an AI agent to construct a valid query, it needs to know:
- What fields are available in the query JSON
- What types those fields expect
- Which fields are required vs optional
- What values are valid (enums, ranges, etc.)

Today, AI agents must guess at query structure based on training data, documentation, or trial-and-error. This leads to:
- **High error rates**: Agents frequently produce malformed queries
- **Limited datasource coverage**: Only well-documented datasources work reliably
- **Poor user experience**: Users must manually fix agent-generated queries

Because of these challenges, we cannot currently utilize the generic `QueryData` endpoint for AI-driven queries. Instead, we must implement dedicated AI tooling for each datasource individually—a "Prometheus query" tool, a "MySQL query" tool, etc.—each with hardcoded knowledge of that datasource's query model. This approach does not scale to the hundreds of datasources in the Grafana ecosystem and excludes community plugins from AI features entirely.

As AI-assisted querying becomes more central to Grafana's value proposition, this lack of introspection becomes a significant blocker.

## Goals

1. **Enable schema discovery**: Provide a mechanism for AI agents (and other tools) to retrieve a machine-readable schema describing a datasource's query model
2. **Maintain backwards compatibility**: Existing plugins must continue to work without modification
3. **Opt-in adoption**: Plugin authors can choose whether to implement schema introspection
4. **Support query type variations**: Some datasources have multiple query types (e.g., metrics vs logs) with different schemas
5. **Use standard formats**: Prefer JSON Schema or similar well-known formats that AI models understand

**Non-goals for initial implementation:**
- Automatic schema generation from Go structs (could be added later as a convenience)
- Frontend integration (focus on backend API first)
- Validation enforcement (schema is informational, not a runtime validator)

## Proposals

### Proposal 0: Do Nothing

If we do nothing, AI agents will continue to rely on:
- Hardcoded knowledge from training data (quickly outdated)
- Documentation scraping (fragile, incomplete)
- Trial-and-error with query execution (slow, poor UX)
- **Per-datasource tooling**: We currently have to build specific AI tools for each datasource (e.g., a "Prometheus query" tool, a "MySQL query" tool), each with hardcoded schema knowledge

**Impact over time:**
- **Short term (0-6 months)**: AI query features work for popular, well-documented datasources (Prometheus, MySQL) but fail for others. Each supported datasource requires dedicated engineering effort to build and maintain its AI tooling.
- **Medium term (6-18 months)**: As more users expect AI assistance, support burden increases; community plugins are effectively excluded from AI features. The per-datasource approach does not scale to the hundreds of datasources in the Grafana ecosystem.
- **Long term (18+ months)**: Grafana's AI capabilities are perceived as unreliable compared to competitors who solve this problem. The "big tent" philosophy—where Grafana embraces a wide ecosystem of community and partner datasources—is undermined because only first-party datasources with dedicated tooling get AI support.

**Verdict**: Unacceptable. The per-datasource approach contradicts Grafana's core value of being an open, extensible platform. We need a scalable solution that allows any datasource to participate in AI features.

---

### Proposal 1: New Optional Interface with gRPC Service

Introduce a new `QuerySchemaHandler` interface and corresponding gRPC service, following the established pattern used by `AdmissionHandler`, `ConversionHandler`, and `QueryConversionHandler`.

#### Design

**1. Define the interface in `backend/`:**

```go
// QuerySchemaHandler provides JSON Schema introspection for query models.
// This is an optional interface that plugins can implement to enable
// AI-assisted query building and other tooling.
type QuerySchemaHandler interface {
    // GetQuerySchema returns a JSON Schema describing the expected structure
    // of the DataQuery.JSON field for this datasource.
    GetQuerySchema(ctx context.Context, req *GetQuerySchemaRequest) (*GetQuerySchemaResponse, error)
}

type GetQuerySchemaRequest struct {
    PluginContext PluginContext
    // QueryType allows requesting schema for a specific query type.
    // If empty, returns the default/primary schema.
    QueryType string
}

type GetQuerySchemaResponse struct {
    // Schema is a JSON Schema document (draft-07 or later recommended)
    Schema json.RawMessage
    // QueryTypes lists available query types if the datasource supports multiple.
    // Each type may have a different schema.
    QueryTypes []QueryTypeInfo
}

type QueryTypeInfo struct {
    // Type identifier (matches DataQuery.QueryType)
    Type string
    // Human-readable name
    Name string
    // Description of what this query type does
    Description string
}
```

**2. Add to `ServeOpts`:**

```go
type ServeOpts struct {
    // ... existing fields ...
    
    // QuerySchemaHandler provides schema introspection for AI tooling.
    // Optional to implement.
    QuerySchemaHandler QuerySchemaHandler
}
```

**3. Extend `proto/backend.proto`:**

```protobuf
service Schema {
    rpc GetQuerySchema(GetQuerySchemaRequest) returns (GetQuerySchemaResponse);
}

message GetQuerySchemaRequest {
    PluginContext pluginContext = 1;
    string queryType = 2;
}

message GetQuerySchemaResponse {
    bytes schema = 1;
    repeated QueryTypeInfo queryTypes = 2;
}

message QueryTypeInfo {
    string type = 1;
    string name = 2;
    string description = 3;
}
```

**4. Wire up in SDK** (following existing patterns):
- Create `schemaSDKAdapter` to bridge interface to gRPC
- Conditionally register in `GRPCServeOpts` when handler is provided
- Add `SchemaGRPCPlugin` to `grpcplugin/`

#### Benefits

- **Type-safe**: Schema delivery is a proper gRPC service with defined types
- **Discoverable**: Grafana can detect which plugins support schema introspection by checking for the service
- **Follows existing patterns**: Consistent with how other optional capabilities work in the SDK
- **Extensible**: Easy to add new gRPC methods to the service, and new optional interfaces (e.g., `QueryValidationHandler`) following the same pattern
- **Versioned**: Can evolve the protobuf schema with compatibility guarantees

#### Trade-offs

- **Requires proto changes**: Must update `backend.proto` and regenerate code
- **More implementation work**: New service, adapter, plugin type
- **Plugin authors must implement**: No automatic schema generation (though helpers could be added)
- **Grafana core changes needed**: Must add client-side code to call the new service

#### Migration Path

1. Add interface and gRPC service to SDK (this repo)
2. Update Grafana core to detect and call schema service
3. Document how plugin authors can implement the interface
4. Gradually add implementations to official datasource plugins

---

### Proposal 2: Resource-Based Convention

Use the existing `CallResourceHandler` to expose schema at a well-known HTTP path, requiring no protocol changes.

#### Design

**1. Define a convention** (documentation only, no code changes to SDK):

Plugins that want to expose query schema should handle:
```
GET /schema/query
GET /schema/query?queryType=<type>
```

Response format:
```json
{
  "schema": { /* JSON Schema document */ },
  "queryTypes": [
    {
      "type": "metrics",
      "name": "Metrics Query",
      "description": "Query time-series metrics"
    }
  ]
}
```

**2. Plugins implement in their `CallResource` handler:**

```go
func (d *Datasource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
    if req.Path == "schema/query" {
        schema := d.getQuerySchema(req.URL.Query().Get("queryType"))
        return sender.Send(&backend.CallResourceResponse{
            Status: http.StatusOK,
            Body:   schema,
        })
    }
    // ... handle other resources ...
}
```

**3. Grafana queries the resource endpoint** to discover schema when needed.

#### Benefits

- **No SDK changes required**: Works with current SDK version
- **No proto changes**: Uses existing `CallResource` infrastructure
- **Immediate availability**: Plugins can start implementing today
- **Simple**: Just HTTP endpoint conventions
- **Flexible**: Plugins control exact response format

#### Trade-offs

- **Convention-based**: No type safety; relies on documentation compliance
- **Not discoverable**: Grafana can't know if a plugin supports this without trying the endpoint
- **404 ambiguity**: A 404 could mean "not implemented" or "bug in path handling"
- **Inconsistent implementations**: Different plugins might return slightly different formats
- **No SDK helpers**: Each plugin must implement from scratch
- **Versioning challenges**: Harder to evolve the convention without breaking plugins

#### Migration Path

1. Document the convention in plugin developer docs
2. Update Grafana core to query `schema/query` endpoint
3. Handle 404/errors gracefully (assume no schema available)
4. Gradually add implementations to official datasource plugins

---

### Comparison Summary

| Aspect | Proposal 1 (gRPC Service) | Proposal 2 (Resource Convention) |
|--------|---------------------------|----------------------------------|
| SDK changes required | Yes (new interface, proto, adapter) | No |
| Type safety | Strong (protobuf types) | Weak (JSON convention) |
| Discoverability | Service presence indicates support | Must try endpoint |
| Implementation effort | Higher initially | Lower initially |
| Consistency | Enforced by types | Relies on documentation |
| Extensibility | Add new gRPC methods + optional interfaces | Add more path conventions |
| Time to first implementation | ~2-4 weeks | ~1 week |

**Recommendation**: Proposal 1 (gRPC Service) is preferred for long-term maintainability and consistency with the SDK's architecture. However, Proposal 2 could serve as a short-term solution while Proposal 1 is developed, or as a fallback for plugins that can't update their SDK version.

## Other Notes

### JSON Schema Considerations

- Recommend JSON Schema draft-07 or draft-2020-12 for broad tooling support
- Consider providing SDK helpers to generate schema from Go structs (using libraries like `github.com/invopop/jsonschema`)
- AI models are trained on JSON Schema, making it ideal for this use case

### Related Work

- The [`QueryConversionHandler`](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go/backend@v0.284.0#QueryConversionHandler) is the closest existing pattern for optional query-related capabilities
- OpenAPI/Swagger specifications solve similar problems for REST APIs
- GraphQL introspection is analogous prior art

### Future Extensions

Once schema introspection exists, we could add:
- `ValidateQuery`: Check a query against schema before execution
- `GetQueryExamples`: Return example queries for AI few-shot prompting
- `GetQueryDocumentation`: Rich documentation for query fields
- Automatic schema generation from Go struct tags

### Open Questions

1. Should schema be cached? At what granularity (per-datasource-instance, per-datasource-type)?
2. How do we handle datasources where schema depends on connected database (e.g., available tables/columns)?
3. Should we support schema composition (base schema + query-type-specific extensions)?
