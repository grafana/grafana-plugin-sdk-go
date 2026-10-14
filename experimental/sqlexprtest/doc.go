// Package sqlexprtest checks whether a data source's responses work as input
// to Grafana SQL Expressions.
//
// SQL Expressions read each query's frames as one SQL table. Grafana decides
// how to build that table from the frames' Meta.Type, field types, and labels.
// A response that works in a panel can still fail in SQL, or lose columns,
// rows, or digits without an error. This package runs the same decisions on a
// plugin's frames, so a plugin's own tests can catch those problems.
//
// Use [Check] on the frames of one query, [CheckResponse] on a whole
// QueryDataResponse, or [CheckQueryData] to call a plugin's QueryData handler
// the way SQL Expressions call it. In tests, [AssertFrames] and
// [AssertResponse] fail the test on findings at or above a severity.
//
//	func TestQueryWorksInSQLExpressions(t *testing.T) {
//		resp, err := ds.QueryData(ctx, req)
//		require.NoError(t, err)
//		sqlexprtest.AssertResponse(t, resp, []string{"A"})
//	}
//
// Each [Finding] has a [Severity] and an [Owner]:
//
//   - SeverityError: the SQL Expression fails on this input.
//   - SeverityWarning: SQL runs, but data is lost or changed without an error.
//   - SeverityInfo: advice; nothing is lost.
//
// OwnerPlugin findings are fixed in the plugin. OwnerGrafana findings describe
// how Grafana handles the input today; the plugin can often work around them,
// for example by returning a table or long frame instead of a wide one.
//
// The checks follow Grafana's SQL input handling at the time of writing. They
// do not run SQL and do not import Grafana. A [Report] also lists the columns
// SQL will see, with their SQL types.
//
// This package is experimental. Its API and checks may change as Grafana's SQL
// input handling changes.
package sqlexprtest
