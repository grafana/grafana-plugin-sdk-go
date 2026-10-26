# Finding reference

Each row gives a small input that produces the finding, and how to fix it. Check the finding's severity and `FixedIn` against your minimum Grafana version. Change `Meta.Type` only when the frame really has the shape of that type.

| Code | Example input | Fix |
| --- | --- | --- |
| `query-error` | The response for `A` has a `DataResponse.Error`. | Fix the query or the upstream error. For a test that expects the error, set `ExpectedFailure` and add an allowance for this code. |
| `missing-refid` | The request asks for `A`, but the response has only `B`. | Key every response by its query's RefID. SQL reads a missing `A` as a table with no columns. |
| `unexpected-refid` | The request asks for `A`, and the response also has `B`. | Return results only for the queries in the request. SQL never reads `B`. |
| `nil-response` | The handler returns `(nil, nil)`. | Return a `QueryDataResponse` or an error. |
| `panic` | The handler panics while it decodes an upstream response. | Return an error in the `DataResponse`. How far the panic spreads depends on how the plugin recovers from panics. |
| `no-frames` | `data.Frames{}`. | If the query should give SQL a schema, return a frame with typed fields and zero rows. |
| `no-fields` | One `data.NewFrame("")`. | Keep the fields of the empty result, with zero rows. |
| `missing-type` | One frame with no labels and no `Meta.Type`. | SQL reads it as a table. Set `Meta.Type` to say what shape the frame has. This finding is info. |
| `multiple-table-frames` | Two table frames for one RefID. | Return one table, or use a multi-series type that matches the frames. |
| `labels-on-table` | Table field `v` has `Labels{"host":"a"}`. | Add a string `host` column. Or, if the frame is a series, set `Meta.Type` to timeseries-wide, timeseries-multi, numeric-wide or numeric-multi. |
| `mixed-types` | The first frame is numeric-multi, and a later frame has a different type. | Return one shape per RefID. The first frame decides how Grafana converts all of them. |
| `wide-frame-count` | Two timeseries-wide frames. | Return one wide frame, or timeseries-multi frames. |
| `missing-time-field` | A timeseries-wide frame has only a numeric field. | Add a non-nullable time field, or use a numeric or table type. |
| `nullable-time` | Time series input whose only time fields are `[]*time.Time`. | Use a non-nullable time field. If times can be NULL, return a table or long frame. |
| `numeric-row-count` | The first numeric field has `[1,2]`. | Numeric-wide and numeric-multi hold one value per field. For more values, return a table or a time series. |
| `numeric-values-truncated` | The first numeric field has `[1]`, and a later one has `[2,3]`. | Return a shape that keeps every value. Grafana drops the extra values of later fields. |
| `empty-numeric-field` | A numeric-wide frame has an empty numeric field. | For an empty result, return a table with typed fields and zero rows. |
| `nil-frame` | The first frame is a nil pointer. | Return a frame. Grafana can stop at an earlier problem before it reaches a later nil frame. |
| `nil-field` | A frame has a nil field pointer. | Create fields with `data.NewField`. |
| `invalid-field` | `&data.Field{Name:"v"}`, with no values array. | Create the field with `data.NewField`. Grafana can accept the table and still fail when it reads the columns. |
| `unequal-field-lengths` | The time and value fields have different lengths, or table fields do. | Give every field the same length. Depending on the type, Grafana drops values, fails the conversion or fails when SQL reads the rows. |
| `dropped-field` | A string `status` field next to numeric fields in timeseries-wide. | Return a table or long frame to keep the string column. |
| `dropped-label` | The time field has a `host` label. | Put series labels on the numeric fields, or add string columns. |
| `integer-as-float` | A wide or multi value of `int64(9007199254740993)`. | Return a table or long frame with int64 fields when exact integers matter. Full-long conversion turns them into float64. |
| `large-integer-in-float` | A large float32 or float64 that may be a rounded integer. | Decode integer values into integer fields. A table or long frame keeps them as integers. The kit can't recover digits that were already lost. |
| `nonfinite-as-null` | A numeric value that SQL reads is NaN or infinity. | Check that NULL is what you want, especially for `COUNT(column)` and filters. This finding is info. |
| `invalid-json` | `json.RawMessage("{")`, or an empty byte slice that isn't nil. | Return valid JSON. SQL reads invalid JSON as NULL. A nil `RawMessage` and the JSON text `null` are both valid, and both mean JSON null. |
| `enum-field` | A table with enum indexes and text in the field config. | Return a string field if SQL should see the text. SQL reads enum indexes as JSON numbers. |
| `duplicate-column-name` | Table columns `host` and `HOST`. | Give columns names that differ when case is ignored. |
| `generated-name-clash` | A label key `time` clashes with the generated `time` column. | Rename the label. If a Grafana fix renames the column for your minimum version, use the new name. |
| `missing-label-column` | Only an empty timeseries-multi series has the label `host`. | Return a shape that keeps the empty series' columns, such as wide, table or long. |

More examples:

- Small malformed inputs: `core_regression_test.go`.
- Renamed columns and Grafana versions: `version_ranges_test.go`.
- Fixed table and wide-metric shapes: the [author examples](examples/author_test.go).

Suite failures, such as a missing column, too many cells or an unused allowance, appear as problems in the suite report. They don't have finding codes.
