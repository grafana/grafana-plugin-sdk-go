# Author and integration examples

To change an existing datasource test as little as possible, start with the [simple assertions](../README.md#start-in-an-existing-test). These examples show how to check expected columns and rows, and how to run real SQL if you want to.

From the SDK root, run the example datasource tests. They need no containers:

```sh
go test ./experimental/sqlexprtest/examples/ -v
```

`author_test.go` has one example handler and three groups of scenarios:

- table results with exact integers and an empty result that keeps its columns;
- labeled wide metrics;
- prepared queries, tested with and without the expression header.

Replace the example handler with your datasource, and mock its upstream service. Write the expected columns and values by hand, not from the handler's output. Change the scenarios to cover the query modes your plugin supports.

The prepared-query example marks one browser-only mode as unsupported. Its Go test passes, but `Passed()` on its report is false because of that mode. Remove the unsupported mode if it doesn't describe your plugin. Otherwise, add a test for the mode before you claim to support it.

## Run real SQL

The compose file starts a local test stack. Don't use its provisioning in a deployed Grafana:

- Grafana listens only on `127.0.0.1`, and anonymous users are administrators.
- The database password is a fixed test value.

From the SDK root:

```sh
export GRAFANA_VERSION=13.2.4
docker compose -p sqlexprtest -f experimental/sqlexprtest/examples/compose.yaml up -d --wait
SQLEXPRTEST_GRAFANA_URL=http://127.0.0.1:13031 \
SQLEXPRTEST_GRAFANA_VERSION="$GRAFANA_VERSION" \
SQLEXPRTEST_PILOTS=1 \
go test ./experimental/sqlexprtest/grafanatest/ ./experimental/sqlexprtest/examples/ -v -count=1
docker compose -p sqlexprtest -f experimental/sqlexprtest/examples/compose.yaml down --volumes
```

Run `down --volumes` every time, even after a test fails. To use another host port, set `GRAFANA_PORT` and change the URL to match. The live tests need a pinned Grafana release. Pick each version you claim to support. The tests fail if `/api/health` reports a different version.

The live tests run only when you turn them on. They cover:

| Datasource | How the query runs | What the tests check |
| --- | --- | --- |
| TestData | Built-in handler, through Grafana's HTTP API and SQL | An exact large integer, NULL, selected columns, counts, an empty result's columns, a join, and an expected SQL error. |
| Prometheus | Real plugin, reading from a fixed fake server, then SQL | Range values, timestamps, labels, order and a sum. |
| PostgreSQL | Real plugin, reading from a PostgreSQL 17.11 database, then SQL | Exact int64 values, NULLs, counts and an empty result's columns. |

`prometheus.py` always returns the same data. It isn't a real Prometheus. PostgreSQL loads the `samples` table from `postgres.sql`.

`TestLiveDatasourcePilots` runs only when `SQLEXPRTEST_PILOTS=1` is set, so it never runs against a server that lacks these fixtures. The TestData tests run when `SQLEXPRTEST_GRAFANA_URL` is set. For login and datasource UID options, see the [`grafanatest` README](../grafanatest/README.md).

These tests send queries straight to Grafana's HTTP API. They don't test:

- the browser's query editor;
- alerts;
- grouped queries;
- a real provider's credentials;
- every mode of these datasources.

Test frontend query preparation in your existing browser end-to-end setup. Add your plugin's own fixtures to cover more modes.

## CI

`ci.yaml` runs this SDK's examples. It isn't an active workflow. For a plugin repo, start from `ci.plugin.yaml`. It builds your plugin, loads it into the test stack with `compose.plugin.yaml`, and runs your tests against each Grafana version you list. To use it:

1. Copy `ci.plugin.yaml` into `.github/workflows/`.
2. Copy `compose.yaml`, `compose.plugin.yaml` and `provisioning/` into your repo.
3. Set `PLUGIN_ID` to the id in your `plugin.json`.
4. Provision a datasource for your plugin in `provisioning/datasources/`. Remove the example datasources you don't need.
5. Change the test packages and the `-run` pattern to your SQL Expressions tests.
6. Save the plugin and SDK revisions with the test results.

The template assumes the create-plugin layout: `npm run build` and `mage build:linux` write the plugin to `dist/`. The overlay mounts `dist/` into Grafana and allows that one unsigned plugin.

To run your plugin in the stack locally:

```sh
PLUGIN_ID=myorg-my-datasource PLUGIN_DIST=./dist \
  docker compose -p sqlexprtest -f compose.yaml -f compose.plugin.yaml up -d --wait
```

`compose.yaml` pins PostgreSQL and the fake Prometheus server by digest. Grafana's image uses the tag from `GRAFANA_VERSION`, so the matrix can change it.

To test this unreleased checkout from a plugin's module, use a local `replace`. Record the `replace` with the test results. It doesn't show whether a released SDK version has this package.

`grafanatest` compares the exact SQL output field types, including whether they are nullable. For example, `COUNT(*)` returns a non-nullable `[]int64`, and a nullable int64 column selected unchanged returns `[]*int64`. Write the expected types from what the query should return, not from the response under test.

## Measure check speed

Run the benchmarks where your plugin tests run:

```sh
go test ./experimental/sqlexprtest/ -run '^$' -bench BenchmarkSQLInputCheck -benchmem
```
