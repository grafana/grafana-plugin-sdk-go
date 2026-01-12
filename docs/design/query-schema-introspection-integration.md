# Query Schema Introspection - Integration Plans

This document provides integration plans for adopting the Query Schema Introspection feature in:
1. Grafana core (server-side)
2. Datasource plugins

## Part 1: Grafana Core Integration

### Overview

Grafana core needs to:
1. Detect which plugins support the `QuerySchema` gRPC service
2. Call `GetQuerySchema` to retrieve schemas
3. Expose schemas to AI agents and other tooling
4. Cache schemas appropriately

### Implementation Steps

#### Step 1: Add QuerySchema Client Support

The plugin client in Grafana needs to be updated to call the new gRPC service.

**Location**: `pkg/plugins/backendplugin/grpcplugin/`

- Add a `QuerySchemaClient` that wraps `pluginv2.QuerySchemaClient`
- Update the plugin client interface to include `GetQuerySchema` method
- Handle the case where older plugins don't support the service (graceful degradation)

```go
// Example interface addition
type Plugin interface {
    // ... existing methods ...
    
    // GetQuerySchema returns the JSON Schema for query models, or nil if not supported
    GetQuerySchema(ctx context.Context, req *backend.GetQuerySchemaRequest) (*backend.GetQuerySchemaResponse, error)
    
    // SupportsQuerySchema returns true if the plugin implements QuerySchemaHandler
    SupportsQuerySchema() bool
}
```

#### Step 2: Plugin Capability Detection

When a plugin is loaded, detect whether it supports the `QuerySchema` service.

**Options**:
- Try calling `GetQuerySchema` and check for "unimplemented" error
- Add a capability field to plugin metadata (plugin.json)
- Use gRPC reflection if available

**Recommendation**: Try calling and cache the result. This is consistent with how other optional capabilities work.

#### Step 3: Expose Schema via API

Create a new HTTP API endpoint to retrieve query schemas.

**Endpoint**: `GET /api/plugins/:pluginId/query-schema`

**Query Parameters**:
- `queryType` (optional): Specific query type to get schema for
- `datasourceUid` (optional): For datasource-specific schemas

**Response**:
```json
{
  "schema": { /* JSON Schema */ },
  "queryTypes": [
    {
      "type": "metrics",
      "name": "Metrics Query", 
      "description": "Query time-series metrics"
    }
  ]
}
```

**Location**: `pkg/api/` - add new route handler

#### Step 4: Caching Strategy

Schemas are unlikely to change at runtime, so aggressive caching is appropriate.

**Recommended approach**:
- Cache per plugin ID + query type
- Invalidate on plugin reload/update
- TTL of 1 hour as fallback
- Store in memory (not distributed cache - schemas are small and local)

**Location**: Add to existing plugin manager or create dedicated schema cache

#### Step 5: AI Agent Integration

Expose schema information to the AI/LLM tooling layer.

**Considerations**:
- AI agents need schemas at tool-definition time
- Consider pre-fetching schemas for all installed datasources
- Include schema in tool descriptions for query execution tools

**Example tool definition**:
```json
{
  "name": "execute_prometheus_query",
  "description": "Execute a Prometheus query",
  "parameters": {
    "$ref": "#/schemas/prometheus-query"
  }
}
```

#### Step 6: Fallback for Non-Supporting Plugins

For plugins that don't implement `QuerySchemaHandler`:
- Return a 404 or empty response from the API
- AI agents should fall back to their existing approach (training data, documentation)
- Consider maintaining a registry of "known schemas" for popular datasources

### Testing

- Integration tests calling the API endpoint
- Tests for graceful handling of non-supporting plugins
- Tests for schema caching behavior
- E2E tests with AI agent mock

---

## Part 2: Datasource Plugin Integration

### Overview

Datasource plugin authors need to:
1. Define their query model as a JSON Schema
2. Implement `QuerySchemaHandler` interface
3. Register the handler in `ServeOpts`

### Implementation Steps

#### Step 1: Define Query Model Schema

Create a JSON Schema document describing your query model.

**Option A: Static schema file**

Create `src/query-schema.json`:
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "queryText": {
      "type": "string",
      "description": "The query expression to execute"
    },
    "format": {
      "type": "string",
      "enum": ["time_series", "table"],
      "default": "time_series",
      "description": "Output format"
    },
    "legendFormat": {
      "type": "string",
      "description": "Legend template for time series"
    }
  },
  "required": ["queryText"]
}
```

**Option B: Generate from Go struct**

Use a library like `github.com/invopop/jsonschema`:
```go
import "github.com/invopop/jsonschema"

type MyQuery struct {
    QueryText    string `json:"queryText" jsonschema:"required,description=The query expression"`
    Format       string `json:"format" jsonschema:"enum=time_series,enum=table,default=time_series"`
    LegendFormat string `json:"legendFormat,omitempty"`
}

func generateSchema() json.RawMessage {
    r := jsonschema.Reflector{}
    schema := r.Reflect(&MyQuery{})
    bytes, _ := json.Marshal(schema)
    return bytes
}
```

#### Step 2: Implement QuerySchemaHandler

Add the handler to your datasource struct:

```go
package plugin

import (
    "context"
    _ "embed"
    "encoding/json"
    
    "github.com/grafana/grafana-plugin-sdk-go/backend"
)

//go:embed query-schema.json
var querySchema []byte

type Datasource struct {
    // ... existing fields ...
}

// Implement QuerySchemaHandler
func (d *Datasource) GetQuerySchema(ctx context.Context, req *backend.GetQuerySchemaRequest) (*backend.GetQuerySchemaResponse, error) {
    // If you have multiple query types, check req.QueryType
    // and return the appropriate schema
    
    return &backend.GetQuerySchemaResponse{
        Schema: querySchema,
        QueryTypes: []backend.QueryTypeInfo{
            {
                Type:        "",  // empty string = default
                Name:        "Query",
                Description: "Execute a query against the datasource",
            },
        },
    }, nil
}
```

#### Step 3: Register in ServeOpts

Update your main.go to register the handler:

```go
func main() {
    ds := &plugin.Datasource{}
    
    err := backend.Manage("my-datasource", backend.ServeOpts{
        CheckHealthHandler:  ds,
        QueryDataHandler:    ds,
        CallResourceHandler: ds,
        QuerySchemaHandler:  ds,  // Add this line
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

#### Step 4: Handle Multiple Query Types (Optional)

If your datasource supports multiple query types:

```go
func (d *Datasource) GetQuerySchema(ctx context.Context, req *backend.GetQuerySchemaRequest) (*backend.GetQuerySchemaResponse, error) {
    schemas := map[string]json.RawMessage{
        "metrics": metricsSchema,
        "logs":    logsSchema,
        "traces":  tracesSchema,
    }
    
    queryTypes := []backend.QueryTypeInfo{
        {Type: "metrics", Name: "Metrics Query", Description: "Query time-series metrics"},
        {Type: "logs", Name: "Logs Query", Description: "Query log entries"},
        {Type: "traces", Name: "Traces Query", Description: "Query distributed traces"},
    }
    
    // If specific type requested, return just that schema
    if req.QueryType != "" {
        schema, ok := schemas[req.QueryType]
        if !ok {
            return nil, fmt.Errorf("unknown query type: %s", req.QueryType)
        }
        return &backend.GetQuerySchemaResponse{
            Schema:     schema,
            QueryTypes: queryTypes,
        }, nil
    }
    
    // Default: return first/primary schema
    return &backend.GetQuerySchemaResponse{
        Schema:     schemas["metrics"],
        QueryTypes: queryTypes,
    }, nil
}
```

#### Step 5: Schema Best Practices

**Include helpful metadata**:
```json
{
  "properties": {
    "expr": {
      "type": "string",
      "description": "PromQL expression",
      "examples": ["rate(http_requests_total[5m])", "up{job=\"prometheus\"}"]
    }
  }
}
```

**Use `$ref` for reusable definitions**:
```json
{
  "$defs": {
    "duration": {
      "type": "string",
      "pattern": "^[0-9]+(ms|s|m|h|d|w|y)$",
      "description": "Duration in Prometheus format (e.g., 5m, 1h)"
    }
  },
  "properties": {
    "range": { "$ref": "#/$defs/duration" },
    "step": { "$ref": "#/$defs/duration" }
  }
}
```

**Document enum values**:
```json
{
  "properties": {
    "format": {
      "type": "string",
      "enum": ["time_series", "table", "heatmap"],
      "enumDescriptions": [
        "Returns data as time series for graph panels",
        "Returns data as a table",
        "Returns data formatted for heatmap visualization"
      ]
    }
  }
}
```

### Testing Your Implementation

```go
func TestGetQuerySchema(t *testing.T) {
    ds := &Datasource{}
    
    resp, err := ds.GetQuerySchema(context.Background(), &backend.GetQuerySchemaRequest{
        PluginContext: backend.PluginContext{
            PluginID: "my-datasource",
        },
    })
    
    require.NoError(t, err)
    require.NotNil(t, resp)
    require.NotEmpty(t, resp.Schema)
    
    // Validate it's valid JSON Schema
    var schema map[string]interface{}
    err = json.Unmarshal(resp.Schema, &schema)
    require.NoError(t, err)
    require.Equal(t, "object", schema["type"])
}
```

### Updating Plugin SDK Version

Ensure your `go.mod` uses a version of the SDK that includes `QuerySchemaHandler`:

```bash
go get github.com/grafana/grafana-plugin-sdk-go@latest
```

---

## Rollout Strategy

### Phase 1: SDK Release
- Release grafana-plugin-sdk-go with QuerySchemaHandler
- Document the feature for plugin developers

### Phase 2: Grafana Core Support
- Add client support and API endpoint
- Initially behind a feature flag

### Phase 3: First-Party Plugins
- Implement in official datasources (Prometheus, Loki, MySQL, etc.)
- Use these as reference implementations

### Phase 4: Community Adoption
- Announce to plugin developers
- Add to plugin development documentation
- Consider adding schema validation to plugin review process

### Phase 5: AI Integration
- Enable AI agents to use schemas
- Deprecate per-datasource hardcoded tools
- Monitor adoption and error rates