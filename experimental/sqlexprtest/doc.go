// Package sqlexprtest checks whether a data source's responses work as input
// to Grafana SQL Expressions.
//
// SQL Expressions read each query's frames as one SQL table. Grafana decides
// how to build that table from the frames' Meta.Type, field types, and labels.
// A response that works in a panel can still fail in SQL, or lose columns,
// rows, or digits without an error. This package predicts what Grafana does
// with a plugin's frames, so a plugin's own tests can catch those problems.
// It does not import Grafana or run SQL. A parity test in Grafana runs the
// same frames through Grafana's real code and checks that the two agree.
//
// Use [Check] on the frames of one query, [CheckResponse] on a whole
// QueryDataResponse, or [CheckQueryData] to call a plugin's QueryData handler.
// Pass CheckResponse the RefIDs the request asked for, so it can report
// missing ones. CheckQueryData calls the handler once per query, as Grafana
// does for SQL Expression inputs, and passes the request's headers unchanged.
//
// In tests, [AssertFrames], [AssertResponse], and [AssertQueryData] fail the
// test on warnings and errors, whichever the owner:
//
//	func TestQueryWorksInSQLExpressions(t *testing.T) {
//		sqlexprtest.AssertQueryData(t, ctx, ds, req)
//	}
//
// [FailOn] changes the severity that fails the test. [Ignore] stops a code
// from failing the test, for every field and query, but the report still
// lists it. [Codes] lists every finding code.
//
// Grafana sets the [HeaderFromExpression] header on some SQL Expression
// requests but not all of them. If the plugin changes what it returns for that
// header, test the request both with and without it.
//
// Each [Finding] has a code, a [Severity], and an [Owner]:
//
//   - SeverityError: the SQL Expression fails on this input.
//   - SeverityWarning: data is lost or changed without an error, or some SQL
//     queries fail, such as a query on an ambiguous column name.
//   - SeverityInfo: advice, or a change SQL makes on purpose, such as reading
//     NaN as NULL. These findings don't fail tests by default.
//
// OwnerPlugin findings are fixed in the plugin. OwnerGrafana findings describe
// how Grafana handles the input today; the plugin can often work around them,
// for example by returning a table or long frame instead of a wide one.
//
// A [Report] lists the conversion mode, the columns SQL will see with their
// SQL types, and the number of rows. ModeTable and ModeFullLong mean Grafana
// builds a table, but an error finding can still mean SQL fails when it reads
// that table. ModeNone means Grafana builds no table, and Rows is 0. A missing
// RefID in a response gives an empty table. A nil response is an error.
//
// When a Grafana release fixes a behavior that an OwnerGrafana finding
// reports, the finding's FixedIn names that release. The check keeps
// reporting it, because users of older Grafana versions still have the
// behavior. Pass the oldest Grafana version the plugin supports, with
// [MinGrafana] or [MinGrafanaFromPluginJSON]. Once every supported version has
// the fix, the report predicts the fixed behavior, including renamed columns.
// Until then, Columns shows the columns of the oldest version, and the
// finding names the new ones. A prerelease comes before its release, so
// 13.3.0-pre does not have a fix released in 13.3.0.
//
//	opt, err := sqlexprtest.MinGrafanaFromPluginJSON("../src/plugin.json")
//	require.NoError(t, err)
//	sqlexprtest.AssertQueryData(t, ctx, ds, req, opt)
//
// [AssertSuite] runs each scenario you list and checks its conversion mode,
// columns, rows, and cells. Each [Allowance] needs a reason and fails when unused.
// [SuiteReport] lists skipped, unsupported, and untested cases. Use encoding/json
// to save it. The optional grafanatest subpackage runs queries against Grafana
// and checks their results.
//
// This package is experimental. Its API and checks may change as Grafana's SQL
// input handling changes.
package sqlexprtest
