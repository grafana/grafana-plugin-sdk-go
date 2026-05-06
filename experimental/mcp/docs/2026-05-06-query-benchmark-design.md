# MCP Datasource Query Benchmark — Design Spec

## Background

The MCP layer in `grafana-plugin-sdk-go/experimental/mcp` translates datasource plugin schema files (query types, examples, routes) into MCP tools and resources that an LLM agent can call. The information exposed via MCP — tool descriptions, schema annotations, query examples, resources, and custom tools — varies in richness depending on what the plugin author provides.

It is not yet known which combinations of that information most improve the quality of queries an agent generates. This benchmark provides a repeatable, scored harness to measure that experimentally.

## Goal

Determine which MCP context dimensions have the most impact on query quality, by running a Claude agent against a live plugin MCP server under named configuration variants and scoring the resulting traces on three independent dimensions.

## Quality Dimensions

Three dimensions, scored separately:

| Dimension | Meaning | Method |
|---|---|---|
| **Correctness** | Right tool called, required args present, query executes without error, result matches expected shape | Deterministic — no LLM |
| **Semantic accuracy** | Result reflects what the user actually asked for | LLM-as-judge (separate Claude call) |
| **Efficiency** | Agent reached a correct answer in the fewest tool calls | `OptimalCalls / ActualCalls`, capped at 1.0 |

Composite score for summary reporting: `0.4×correctness + 0.4×semantic + 0.2×efficiency`. Weights are constants in `eval/scoring.go`.

## Datasources in Scope

- `github-datasource` — API-backed, rich query type variety (PRs, issues, commits, labels, milestones, deployments, contributors)
- `redshift-datasource` — SQL-backed, requires schema discovery (tables, columns) to construct valid queries

Both datasources run against live backends. The benchmark is not designed for replay against fixtures.

The Redshift test cases reference specific table and column names (e.g. `orders`, `analytics`). These must be calibrated against the actual test database before the benchmark is run — the `ExpectedArgs` and `ResultShape` fields in `redshift.json` are placeholders until the schema is known.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Benchmark Runner                                            │
│                                                             │
│  1. Start plugin process with variant's MCP configuration   │
│  2. Read dist/mcp.addr → connect MCP client                 │
│  3. Inject test case prompt into Claude via Anthropic API   │
│  4. Agent loop: Claude calls MCP tools until done/timeout   │
│  5. Capture structured trace (tool calls, args, results)    │
│  6. Score trace on 3 dimensions                             │
│  7. Write result JSON                                        │
└─────────────────────────────────────────────────────────────┘
         │                         │
         ▼                         ▼
  Plugin process              Anthropic API
  (github or redshift)        (claude-sonnet-4-6 as agent)
  MCP server on :auto              │
         │                         │ tool calls
         └─────────────────────────┘
```

The benchmark runner drives one cell (variant × test case × datasource) at a time. Cells are independent and can be parallelised with `--parallel N`.

## Directory Layout

```
benchmark/
├── cmd/bench/main.go       # entry point, flag parsing, run orchestration
├── variants/variants.go    # Variant type + named variant registry
├── cases/
│   ├── cases.go            # TestCase type + JSON loader
│   ├── github.json         # test cases for github-datasource
│   └── redshift.json       # test cases for redshift-datasource
├── agent/loop.go           # Claude MCP client agent loop
├── eval/
│   ├── correctness.go      # deterministic correctness scorer
│   ├── semantic.go         # LLM-as-judge scorer
│   ├── efficiency.go       # tool call count scorer
│   ├── scoring.go          # composite weights + Scores type
│   └── judge_prompt.txt    # versioned judge prompt template
├── trace/trace.go          # Trace + ToolCall types, JSON serialisation
├── results/
│   ├── store.go            # write/read result files
│   └── compare.go          # diff two run directories, render summary table
└── go.mod
```

## Configuration Variants

```go
type Variant struct {
    Name                  string
    RichToolDescriptions  bool  // hand-written descriptions vs auto-generated
    FullSchemaAnnotations bool  // field descriptions, enums, required present
    QueryExamples         bool  // examples://query resource + per-tool examples
    SchemaResources       bool  // table/column/schema metadata as MCP resources
    DescribeDatasource    bool  // explain-this-datasource tool
}
```

Named variants — each adds exactly one dimension to the previous, so score deltas are attributable:

| Variant | Rich descriptions | Full schema | Examples | Schema resources | Describe tool |
|---|---|---|---|---|---|
| `baseline` | | | | | |
| `rich-descriptions` | ✓ | | | | |
| `with-schema` | ✓ | ✓ | | | |
| `with-examples` | ✓ | ✓ | ✓ | | |
| `with-resources` | ✓ | ✓ | ✓ | ✓ | |
| `full` | ✓ | ✓ | ✓ | ✓ | ✓ |

Arbitrary subsets can be selected via `--variants baseline,full` for targeted comparisons.

## Test Cases

### Structure

```go
type TestCase struct {
    ID               string
    Datasource       string         // "github" | "redshift"
    Category         string         // see categories below
    Prompt           string
    ExpectedTool     string         // tool that must be called for correctness
    ExpectedArgs     map[string]any // required arg values that must be present
    ExpectedResult   ResultShape
    OptimalToolCalls int
}

type ResultShape struct {
    MinRows     int
    FieldsExist []string
    FilterCheck string  // JSONPath expression that must hold on every row
}
```

### Categories

| Category | Description | Correctness baseline | `OptimalToolCalls` |
|---|---|---|---|
| **Straightforward** | Right tool and args obvious from the prompt | All variants should pass | 1 |
| **Schema-dependent** | Requires knowing exact field/column names | `baseline` likely fails | 1–2 |
| **Multi-step discovery** | Must call one tool to get input for the next | Needs route tools or resources | 2–3 |
| **Error-recovery** | Naive first attempt returns wrong/empty result; agent must self-correct | Needs describe tool or schema resources | 2 |

### GitHub Datasource Cases

| ID | Category | Prompt | Expected tool | What a weak variant gets wrong |
|---|---|---|---|---|
| `github-open-prs` | Straightforward | "Show me open pull requests in grafana/grafana" | `query_Pull_Requests` | — |
| `github-recent-commits` | Straightforward | "List commits from the last 7 days in grafana/grafana" | `query_Commits` | — |
| `github-pr-author-filter` | Schema-dependent | "Show me merged PRs authored by renovate-bot in grafana/grafana" | `query_Pull_Requests` | Guesses wrong field name (`user`, `creator` instead of `author`) |
| `github-issue-label-filter` | Schema-dependent | "Find open issues with the 'type/bug' label in grafana/grafana" | `query_Issues` | Guesses wrong field name for label filtering |
| `github-milestones-then-issues` | Multi-step discovery | "Show me issues belonging to the first open milestone in grafana/grafana" | `get_milestones` → `query_Issues` | Skips milestone discovery, passes wrong milestone ID |
| `github-labels-then-query` | Multi-step discovery | "Pick a label that exists in grafana/grafana and show me its open issues" | `get_labels` → `query_Issues` | Invents a label name instead of fetching available ones |
| `github-wrong-querytype` | Error-recovery | "Show me the deployment history for grafana/grafana" | `query_Deployments` | Tries `query_Commits` first; needs schema or describe tool to find correct type |
| `github-nonexistent-field` | Error-recovery | "Show me pull requests with more than 5 review comments" | `query_Pull_Requests` | Attempts a field that may not exist; needs schema to recover or rephrase |

### Redshift Datasource Cases

| ID | Category | Prompt | Expected tool | What a weak variant gets wrong |
|---|---|---|---|---|
| `redshift-simple-select` | Straightforward | "Show me the first 10 rows from the orders table" | `query_sql` | — |
| `redshift-count-by-status` | Straightforward | "Count orders grouped by status" | `query_sql` | — |
| `redshift-exact-column` | Schema-dependent | "Show me orders where the total amount exceeds 1000" | `query_sql` | Guesses column name (`total_amount`? `amount`? `order_total`?) |
| `redshift-date-column` | Schema-dependent | "Find orders created in the last 30 days" | `query_sql` | Guesses timestamp column name |
| `redshift-discover-then-query` | Multi-step discovery | "Show me 5 rows from whichever table has the most columns" | `get_tables` → `get_columns` → `query_sql` | Skips schema discovery, invents a table name |
| `redshift-schema-then-table` | Multi-step discovery | "List the tables in the analytics schema and query one of them" | `get_schemas` → `get_tables` → `query_sql` | Invents schema name |
| `redshift-wrong-table` | Error-recovery | "Show me data from the transactions table" | `query_sql` | Queries non-existent table; needs `get_tables` to recover |
| `redshift-ambiguous-column` | Error-recovery | "Find the most recent records" | `query_sql` | No obvious timestamp column; needs schema context to identify the right column |

## Trace Structure

One trace file per (variant × case). Written to `results/<run-id>/traces/<variant>/<datasource>/<case-id>.json`.

```go
type Trace struct {
    RunID       string
    Variant     string
    Datasource  string
    CaseID      string
    Category    string
    StartedAt   time.Time
    DurationMs  int64
    Prompt      string
    ToolCalls   []ToolCall
    FinalAnswer string
    Scores      Scores
}

type ToolCall struct {
    Seq        int
    Tool       string
    Args       map[string]any
    ResultJSON string
    DurationMs int64
    IsError    bool
}

type Scores struct {
    Correctness float64
    Semantic    float64
    Efficiency  float64
    Composite   float64
}
```

## Agent Loop

`agent/loop.go` drives one trace to completion:

1. Read `dist/mcp.addr` → connect to the running plugin process using `modelcontextprotocol/go-sdk`'s HTTP client (`mcpsdk.NewClient` + HTTP transport). The `mcptest` package is in-memory only and not suitable here.
2. Call `tools/list` → build Anthropic API tool definitions
3. Send prompt + tools to `claude-sonnet-4-6`
4. Loop until done or `MaxTurns` (default 10) exceeded:
   - `tool_use` blocks → call each tool via MCP, append `tool_result` messages
   - Text-only response → extract final answer, stop
   - Timeout → record as incomplete trace, stop

`MaxTurns` is configurable per run. Error-recovery cases may warrant a higher limit than straightforward ones.

## Scoring

### Correctness (`eval/correctness.go`)

Binary — 0 or 1. All four checks must pass:

1. Expected tool was called at some point in the trace
2. All `ExpectedArgs` key/value pairs are present in the matching tool call
3. That tool call did not return an MCP error
4. Result satisfies `ResultShape` (min rows, required fields, filter check)

### Semantic Accuracy (`eval/semantic.go`)

LLM-as-judge. A separate Claude API call (not part of the agent loop) receives:
- The original prompt
- The tool calls made (tool name + args only, not results)
- The agent's final answer text

The judge scores 0–3 on each of three axes, normalised to 0.0–1.0:
- Did the agent query for the right thing?
- Are the key filters and constraints from the prompt reflected in the args?
- Does the final answer address what was asked?

The judge prompt is stored in `eval/judge_prompt.txt` and version-controlled.

### Efficiency (`eval/efficiency.go`)

```
score = min(1.0, OptimalToolCalls / len(ToolCalls))
```

Error-recovery cases have `OptimalToolCalls=2` by design — one discovery/recovery call plus one correct call — so a 2-call solution scores 1.0 even though the first attempt failed.

### Composite

```
composite = 0.4×correctness + 0.4×semantic + 0.2×efficiency
```

Used only for summary reporting. Per-dimension scores are the primary analysis unit.

## Results & CLI

### Results Layout

```
results/
└── <run-id>/
    ├── meta.json          # variant list, datasources, start time, flags used
    └── traces/
        └── <variant>/
            └── <datasource>/
                └── <case-id>.json
```

### Summary Table (bench show / bench compare)

```
                          baseline   rich-desc   with-schema   with-examples   with-resources   full
github-open-prs           C:1 S:.72  C:1 S:.81   ...
github-pr-author-filter   C:0 S:.41  C:1 S:.79   ...
redshift-wrong-table      C:0 S:.38  C:0 S:.44   ...
...
composite (avg)           0.51       0.63        ...
```

Efficiency shown with `--verbose`. Category subtotals shown with `--by-category`.

### CLI

```bash
# run all variants against all cases for both datasources
bench run --datasource both --all-variants

# run specific variants
bench run --datasource github --variants baseline,full

# run a single case for debugging
bench run --datasource github --variants with-examples --case github-open-prs

# compare two runs
bench compare <run-id-a> <run-id-b>

# show results for a single run
bench show <run-id> [--verbose] [--by-category]
```

Key flags for `bench run`:
- `--max-turns N` — agent loop turn limit (default 10)
- `--parallel N` — concurrent cells (default 1)
- `--output dir` — results directory (default `./results`)

## Success Criteria

The benchmark is useful when:

1. `baseline` vs `full` shows a measurable score difference on schema-dependent and error-recovery cases
2. The additive variant structure isolates which single dimension drives each improvement
3. Results are reproducible across runs (same variant + same case → scores within ±0.1)
4. A new test case can be added by editing a JSON file with no Go changes required
