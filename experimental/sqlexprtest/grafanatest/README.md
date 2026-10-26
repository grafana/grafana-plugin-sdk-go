# Check SQL results in Grafana

`grafanatest` sends datasource queries and SQL Expressions to a running Grafana, and checks the values that come back. It is optional. The `sqlexprtest` checks need no server.

Use fixed upstream data or a test database. Run the plugin's real backend, so the test covers query decoding, paging, frame building and serialization. An HTTP 200 alone doesn't show that the values are right.

## Write a runtime contract

Provision the datasource first. Then send its real query model, with `refId` and `datasource`, followed by a SQL Expression. Put this code in a Go test, and replace the source query with your plugin's query model:

```go
client := grafanatest.Client{
    URL:             os.Getenv("SQLEXPRTEST_GRAFANA_URL"),
    ExpectedVersion: os.Getenv("SQLEXPRTEST_GRAFANA_VERSION"),
    Headers: http.Header{
        "Authorization": {"Bearer " + os.Getenv("SQLEXPRTEST_GRAFANA_TOKEN")},
    },
}
result, err := client.Query(t.Context(), grafanatest.Request{
    From: "1760000000000",
    To:   "1760000060000",
    Queries: []json.RawMessage{
        json.RawMessage(`{"refId":"A","datasource":{"uid":"your-datasource"},"query":"your query"}`),
        grafanatest.SQL("B", "SELECT id FROM A ORDER BY id"),
    },
})
require.NoError(t, err)
grafanatest.AssertResult(t, result, map[string]grafanatest.Expectation{
    "B": {Frames: data.Frames{data.NewFrame("",
        data.NewField("id", nil, []*int64{new(int64(9007199254740993))}),
    )}},
})
```

The test imports `encoding/json`, `net/http`, `os`, the SDK's `data` and `experimental/sqlexprtest/grafanatest` packages, and your usual assertion library. If your test server doesn't need a login, leave out the `Authorization` header. The client sends your headers unchanged. It doesn't add the expression header.

Pin `ExpectedVersion` in CI. For an unreleased build, also set `ExpectedCommit`. Each result records the version and commit from `/api/health`, and `direct-api` as the way the query ran. A result covers only the queries you sent, on that version. Test browser query preparation and alerts separately if your plugin needs them.

## What a contract checks

- The order of frames and columns, field names, exact SDK field types including nullable ones, labels, row counts and values.
- Integers above `2^53` stay exact when the client decodes and compares them, including inside JSON values.
- JSON comparison ignores key order and whitespace, but compares numbers as written: `1` and `1.0` differ.
- SQL NULL is checked separately from zero.
- A null cell in a non-nullable field fails the check. The exception is a NaN or infinity float placeholder that the frame metadata declares.
- Notices must match exactly. A warning that Grafana cut the output short fails the check, even on a source query whose values you didn't check.
- Every returned query is checked for errors, even when you only set values for the SQL Expression.

HTTP serialization can turn SQL NULL and JSON `null` into the same thing in a JSON column. To tell them apart, check `COUNT(column)` or `IS NULL` in the SQL itself.

Value checks ignore frame names, RefIDs inside frames, and display metadata. The key in the response map is the query's RefID. Use `ORDER BY` when the row order matters.

Expected `[]int64` and `[]*int64` are different on purpose: `[]*int64` expects a nullable column. `data.Frames{}` expects no frames. A frame with typed, empty fields expects those columns and zero rows.

To expect a failure, set `Expectation{ErrorContains: "missing_column"}` instead of `Frames`. This checks only the error. To test that SQL accepts your input, add queries that succeed. Set `Notices` only in a test that expects a limit or another warning. Never set both `Frames` and `ErrorContains`.

`Check` returns the mismatches without failing a test. `AssertResult` reports them through `testing.TB`. You can save the result and the mismatches as JSON in CI. They hold query data and errors, so use test data that is safe to save.

## Run the included live tests

The SDK's [examples](../examples/README.md) start a local Grafana with a TestData datasource whose UID is `sqlexpr-testdata`. The live tests cover large integers, NULL counts, an empty result's columns, a join of two RefIDs and an expected SQL error.

```sh
SQLEXPRTEST_GRAFANA_URL=http://127.0.0.1:13031 \
SQLEXPRTEST_GRAFANA_VERSION=13.2.4 \
go test ./experimental/sqlexprtest/grafanatest/ -run TestQueryLive -v
```

You can also set `SQLEXPRTEST_GRAFANA_COMMIT`, `SQLEXPRTEST_GRAFANA_TOKEN` and `SQLEXPRTEST_DATASOURCE_UID`. Without the URL, the live tests skip. The client and comparison tests still run against `httptest`, so normal package tests need no running Grafana.

The client:

- works with Grafana on a subpath;
- rejects redirects;
- limits each response to 16 MiB by default;
- gives the health check and the query 30 seconds together.

Raise those limits for larger fixtures. Login failures and malformed responses return an error. Grafana's errors for single queries stay in the result, so you can check them.
