# SQL Expressions compatibility tests

`sqlexprtest` checks, in an ordinary Go test, whether your datasource's responses work in Grafana SQL Expressions. It doesn't import Grafana. Start by adding one assertion to an existing datasource test. Then add the empty, nullable, multi-series and error cases that matter to your plugin.

| Function | What it does |
| --- | --- |
| `AssertFrames`, `AssertResponse`, `AssertQueryData` | Fail a Go test when the kit predicts a warning or an error. |
| `Check`, `CheckResponse`, `CheckQueryData` | Predict the conversion mode, columns, rows and problems for your frames or handler. |
| `MinGrafana`, `MinGrafanaFromPluginJSON` | Predict the behavior of the oldest Grafana version your plugin supports. |
| `Codes` | List every finding code. |

All of these APIs are experimental.

## Start in an existing test

If your test already calls the datasource's `QueryData`, add one assertion after you check the returned error:

```go
sqlexprtest.AssertResponse(t, response, []string{"A"})
```

Pass the RefIDs from the request, so a missing response fails the test too. If the test already has frames, use `sqlexprtest.AssertFrames(t, frames)`. To let the kit call your handler and check every RefID in the request, use:

```go
sqlexprtest.AssertQueryData(t, t.Context(), datasource, request)
```

Import `github.com/grafana/grafana-plugin-sdk-go/experimental/sqlexprtest`. By default, the helpers fail on warnings and errors, including known Grafana limits. They return the reports, so you can add your own assertions. They predict what SQL gets as input. They don't run SQL or check cell values.

Save this test as `sql_input_test.go` and run it with `go test`:

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

Test the frames your real handler returns when you can. Mock the upstream service below the handler's decoder, so the test covers query parsing, response decoding and frame building. A hand-built frame is useful to pin one shape. It doesn't show that your datasource returns that shape.

## Choose what fails the test

Pass `FailOn` and `Ignore` to the Assert helpers:

```go
sqlexprtest.AssertQueryData(t, ctx, ds, req,
    sqlexprtest.MinGrafana("13.2.4"),
    sqlexprtest.FailOn(sqlexprtest.SeverityError),
    sqlexprtest.Ignore(sqlexprtest.CodeMissingType),
)
```

- `FailOn` sets the lowest severity that fails the test. The default is `SeverityWarning`.
- `Ignore` stops a code from failing the test, for every field and query. The report still lists it.

## Test with and without the expression header

Grafana marks the data source requests it makes for expressions with `HeaderFromExpression` (`X-Grafana-From-Expr`). Not every path sets it yet: requests through the query service don't have it. `CheckQueryData` passes your request's headers unchanged. If your plugin returns different data when the header is set, test the request both ways:

```go
sqlexprtest.AssertQueryData(t, ctx, ds, req)

withHeader := *req
withHeader.Headers = maps.Clone(req.Headers)
withHeader.SetHTTPHeader(sqlexprtest.HeaderFromExpression, "true")
sqlexprtest.AssertQueryData(t, ctx, ds, &withHeader)
```

## Read a failure

A failing test prints one report per query:

```
SQL Expressions input check failed:
  query "A": mode full-long, input type timeseries-wide, rows 1
    columns: time TIMESTAMP, __value__ DOUBLE, __metric_name__ TEXT, host TEXT
    warning [dropped-field, grafana] frame 0 field "status": ...
```

The [finding reference](FINDINGS.md) gives an example input and a fix for every code. `Owner` says where the fix belongs: `plugin`, or `grafana` for how Grafana handles the input today. Warnings and errors fail by default, whatever the owner.

`ModeTable` with an error finding means Grafana builds the table, but SQL fails when it reads it. `ModeNone` means Grafana builds no table. Findings about malformed frames describe what Grafana does with the frames as given. Your plugin's transport may reject such frames before Grafana sees them.

## Versions

Pass the oldest Grafana version your plugin supports, with `MinGrafana` or `MinGrafanaFromPluginJSON`. Without one, the kit assumes that none of the Grafana fixes it knows about are present. It doesn't detect the version of a running Grafana. If the kit can't read a `grafanaDependency` range, it returns an error instead of guessing.

A pre-release sorts before its release, so `13.3.0-pre` doesn't have a fix released in `13.3.0`.

`Columns` shows the columns at your minimum version. `FixedIn` names the release with a known fix. It doesn't mean your minimum version has that fix. The kit only knows fixes that stay in every later release. A fix backported to an older line, but missing from a newer one, needs a new version rule before the kit can record it.

## What the kit doesn't check

- Query logic that runs only in the browser. The backend never sees it.
- Cell values, NULLs and precision after SQL runs. The kit predicts the input table only.
- The `sseGroupByDatasource` feature flag. With the flag on, Grafana converts grouped results with a different code path, so the kit's predictions don't apply.
- Time handling in Grafana's SQL engine, such as sub-second rounding and time zones.

## Keeping the kit in sync with Grafana

The kit copies Grafana's conversion rules, so it can drift from them. A parity test for Grafana's `pkg/expr` will run the same frames through Grafana's real code. It will compare the conversion stage, rows, columns in order and real SQL results with the kit's predictions. It opens as a Grafana PR once an SDK release includes the kit. Until then, the kit's own tests check it only against itself.
