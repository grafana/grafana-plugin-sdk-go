# SQL Expressions compatibility tests

`sqlexprtest` checks, in an ordinary Go test, whether your datasource's responses work in Grafana SQL Expressions. It doesn't import Grafana. Start by adding one assertion to an existing datasource test. Then add the empty, nullable, multi-series and error cases that matter to your plugin.

| Tool | What it checks |
| --- | --- |
| `AssertFrames`, `AssertResponse`, `AssertQueryData` | Fail a Go test when the kit predicts a warning or an error. |
| `Check`, `CheckResponse`, `CheckQueryData` | Predict the conversion mode, columns, rows and problems for your frames or handler. |
| `AssertSuite` | Check named scenarios against the columns and rows you expect, and report which query modes have tests. |
| [`grafanatest`](grafanatest/README.md) | Run real SQL Expressions in a running Grafana, and check the values that come back. |
| Grafana's parity tests | Check that the kit's predictions still match Grafana's real code. |

Each tool checks a different part of compatibility. All of these APIs are experimental.

## Start in an existing test

If your test already calls the datasource's `QueryData`, add one assertion after you check the returned error:

```go
sqlexprtest.AssertResponse(t, response, []string{"A"})
```

Pass the RefIDs from the request, so a missing response fails the test too. If the test already has frames, use `sqlexprtest.AssertFrames(t, frames)`. To let the kit call your handler and check every RefID in the request, use:

```go
sqlexprtest.AssertQueryData(t, t.Context(), datasource, request)
```

Import `github.com/grafana/grafana-plugin-sdk-go/experimental/sqlexprtest`. By default, the helpers fail on warnings and errors, including known Grafana limits. They return the reports, so you can add your own assertions. They predict what SQL gets as input. They don't run SQL or check cell values. You don't need a suite or a running Grafana to use them.

Save this test as `sql_input_test.go` and run it with `go test`, in a module that uses this SDK checkout:

```go
package plugin_test

import (
    "testing"

    "github.com/grafana/grafana-plugin-sdk-go/data"
    "github.com/grafana/grafana-plugin-sdk-go/experimental/sqlexprtest"
)

func TestSQLInput(t *testing.T) {
    frame := data.NewFrame("",
        data.NewField("host", nil, []string{"a", "b"}),
        data.NewField("bytes", nil, []int64{10, 20}),
    ).SetMeta(&data.FrameMeta{Type: data.FrameTypeTable})
    sqlexprtest.AssertFrames(t, data.Frames{frame}, sqlexprtest.MinGrafana("13.2.4"))
}
```

Pass the oldest Grafana version your plugin supports. Without `MinGrafana`, the kit assumes that none of the Grafana fixes it knows about are present. It doesn't detect the version of a running Grafana.

Test the frames your real handler returns when you can. Mock the upstream service below the handler's decoder, so the test covers query parsing, response decoding and frame building. A hand-built frame is useful to pin one shape. It doesn't show that your datasource returns that shape.

## Check expected columns and rows

Use `AssertSuite` to name scenarios, track tests for each query mode, and check exact columns and row counts. You can use it next to the simple assertions.

This test checks the column names, their order, field types, SQL types, the row count and the conversion mode:

```go
package plugin_test

import (
    "testing"
    "github.com/grafana/grafana-plugin-sdk-go/data"
    "github.com/grafana/grafana-plugin-sdk-go/experimental/sqlexprtest"
)

func TestSQLInputContract(t *testing.T) {
    frame := data.NewFrame("",
        data.NewField("host", nil, []string{"a", "b"}),
        data.NewField("bytes", nil, []int64{10, 20}),
    ).SetMeta(&data.FrameMeta{Type: data.FrameTypeTable})
    rows := 2
    sqlexprtest.AssertSuite(t, t.Context(), sqlexprtest.Suite{
        Name: "my datasource",
        Modes: []sqlexprtest.QueryMode{{Name: "table"}},
        Scenarios: []sqlexprtest.Scenario{{
            Name: "two hosts", Mode: "table",
            Input: sqlexprtest.FramesInput("A", data.Frames{frame}),
            Want: map[string]sqlexprtest.Contract{"A": {
                Mode: sqlexprtest.ModeTable, Rows: &rows,
                Columns: []sqlexprtest.Column{
                    {Name: "host", Type: data.FieldTypeString, SQLType: "TEXT"},
                    {Name: "bytes", Type: data.FieldTypeInt64, SQLType: "BIGINT"},
                },
            }},
        }},
    }, sqlexprtest.MinGrafana("13.2.4"))
}
```

To test your handler, replace `FramesInput` with `QueryDataInput(yourHandler, request)`. If a test already has a `QueryDataResponse`, use `ResponseInput`, and pass the RefIDs from the request so a missing response shows up. Write `Want` by hand, from what your plugin should return. If you copy `Want` from the report, the test agrees with any regression.

The [author examples](examples/author_test.go) cover table data with exact integers, empty schemas, labeled metrics and prepared queries. Run them with:

```sh
go test ./experimental/sqlexprtest/examples/ -v
```

## List the query modes you test

List your query modes in `Suite.Modes`. Give each supported mode at least one scenario with valid input. Use separate modes when queries send different backend requests or return different frame shapes, for example table, range and instant, or builder and raw queries.

| Status | Meaning |
| --- | --- |
| `passed` | The scenarios matched what you expected, and every finding was allowed. |
| `failed` | A scenario's columns, rows, findings, allowances or input didn't match. |
| `untested` | A listed mode has no passing scenario with valid input. Scenarios that expect errors don't count. |
| `skipped` | The scenario didn't run. The report shows its reason. |
| `unsupported` | You marked the mode unsupported, with a reason. |

`AssertSuite` fails when a scenario or mode is `failed` or `untested`. If you skip every valid-input scenario of a mode, the mode is untested and the test fails. A mode with one passing scenario and one skipped scenario is `skipped`, and the test doesn't fail. An unsupported mode doesn't fail the test either.

To require every listed mode to pass, also check `report.Passed()`. It is false when any mode or scenario is skipped, unsupported or untested. The kit doesn't find your modes for you. You list them.

Consider these scenarios for each mode:

- Results that are full, empty with typed fields, all NULL, or a mix of empty and full.
- Sparse labels, display names, several series and extra frames.
- Large integers that must stay exact, fractional times, invalid JSON, and NULL compared with zero.
- Upstream errors, missing RefIDs, paging and results that aren't ready yet.
- The backend request with and without `HeaderFromExpression`. The kit passes your headers unchanged.

A contract checks only what you set. An empty `Mode`, or a nil `Rows` or `Columns`, is not checked. A non-nil empty `Columns` slice checks that there are no columns. Each scenario must list a contract for every RefID it reports. A missing or extra report fails. A response for a RefID that no query asked for is the exception. It needs no contract, only an allowance for `unexpected-refid`, and it doesn't count as SQL input.

`ExpectedFailure` marks a scenario that should fail, such as an upstream error. It still needs an allowance for the error's code. It doesn't count as a test with valid input.

## Allow one known finding

Use an allowance in the scenario, not `Ignore`:

```go
field := "status"
allowance := sqlexprtest.Allowance{
    Code: sqlexprtest.CodeEnumField, RefID: "A", Field: &field,
    Reason: "This legacy mode documents status as an index; tracked in our migration issue.",
}
```

- `RefID`, `Frame` and `Field` are optional. When set, they must match exactly.
- The first allowance that matches a finding is the one used.
- Every allowance needs a reason. An allowance that matches nothing fails the test.
- Allowed findings stay in the report.
- Only findings at or above the failure severity use an allowance. If a Grafana fix turns a warning into info, the allowance goes unused and fails the test, so you know to remove it.
- An error finding needs an `ExpectedFailure` scenario.

`Field` matches `Finding.Field`, which can be the name of a generated column. It can't tell apart two fields with the same name in one frame. Rename those fields, or test them in separate scenarios.

The simple Assert helpers still accept `Ignore`. `AssertSuite` rejects it, so a suite can't hide a code everywhere by accident.

## Read a failure

A suite report shows the query mode, the scenario, the RefID, the conversion mode, the SQL columns in order, the rows, the input cells, the findings and the allowances used. The [finding reference](FINDINGS.md) gives an example input and a fix for every code. `Owner` says where the fix belongs. Warnings and errors fail by default, whatever the owner.

`ModeTable` with an error finding means Grafana builds the table, but SQL fails when it reads it. `ModeNone` means Grafana builds no table. Findings about malformed frames describe what Grafana does with the frames as given. Your plugin's transport may reject such frames before Grafana sees them.

The report is a prediction. It doesn't hold cell values, and the SDK has no SQL engine. To check values, SQL NULLs, precision, joins and SQL output, use `grafanatest`.

## Versions and sizes

Pass the oldest Grafana version your plugin supports, with `MinGrafana` or `MinGrafanaFromPluginJSON`. If the kit can't read a `grafanaDependency` range, it returns an error instead of guessing. A pre-release sorts before its release, so `13.3.0-pre` doesn't have a fix released in `13.3.0`.

`Columns` shows the columns at your minimum version. `FixedIn` names the release with a known fix. It doesn't mean your minimum version has that fix. The kit only knows fixes that stay in every later release. A fix backported to an older line, but missing from a newer one, needs a new version rule before the kit can record it.

`Report.Cells()` is the number of rows times the number of columns after conversion. For example, 50 wide series with 1,000 timestamps and three generated columns give 50,000 rows and 150,000 cells. `Contract.MaxCells` sets a limit for one input. Zero or a negative value means no limit. It isn't Grafana's limit, which counts all inputs together, and it doesn't predict the size of the SQL output. `grafanatest` fails on notices you don't expect, including the warning Grafana adds when it cuts the output short.

## CI and saved reports

Run the fast tests in your normal `go test` job. The [integration setup](examples/README.md) is optional. It runs a pinned Grafana with TestData, a real PostgreSQL database, and the real Prometheus plugin reading from a fixed fake server. The [CI template](examples/ci.yaml) runs those tests on the Grafana versions you list. Choose the tests and versions your plugin supports.

`Evaluate` returns the same `SuiteReport` as `AssertSuite`, without a test. Save it with `encoding/json`. `schemaVersion: 1` marks the experimental report format. Reject versions you don't know. Without a minimum version, the JSON has no `minGrafana`, and the checks assume none of the known fixes. Save the plugin revision and the output of `go list -m -json github.com/grafana/grafana-plugin-sdk-go` next to the report. `grafanatest` results also record the Grafana version, commit and how the query ran.

The included Grafana tests send queries straight to the HTTP API. They don't test the browser's query preparation, alerting, grouped queries or every mode of a plugin. If frontend preparation matters, add browser tests in your plugin's existing end-to-end setup.

## Keeping the kit in sync with Grafana

Grafana's parity tests run the same frames through Grafana's real code. They compare the conversion stage, rows, columns in order and real SQL values with the kit's predictions. A generator adds 64 random cases that repeat exactly. Each case name includes the generator version and seed, and a failing case prints its frames as JSON. A fuzz target explores more valid shapes. Malformed shapes have their own fixed test cases. When the generator or fuzzer finds a failure, add it as a small fixed case before you change the generator.
