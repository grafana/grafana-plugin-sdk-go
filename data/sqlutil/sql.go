package sqlutil

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// FrameFromRows returns a new Frame populated with the data from rows. The field types
// will be nullable ([]*T) if the SQL column is nullable or if the nullable property is unknown.
// Otherwise, the field types will be non-nullable ([]T) types.
//
// The number of rows scanned is limited to rowLimit. If maxRows is reached, then a data.Notice with a warning severity
// will be attached to the frame. If rowLimit is less than 0, there is no limit.
//
// Fields will be named to match name of the SQL columns.
//
// A converter must be supplied in order to support data types that are scanned from sql.Rows, but not supported in data.Frame.
// The converter defines what type to use for scanning, what type to place in the data frame, and a function for converting from one to the other.
// If you find yourself here after upgrading, you can continue to your StringConverters here by using the `ToConverters` function.
func FrameFromRows(rows *sql.Rows, rowLimit int64, converters ...Converter) (*data.Frame, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	// If there is a dynamic converter, we need to use the dynamic framer
	// and remove the dynamic converter from the list of converters ( it is not valid, just a flag )
	if isDynamic, converters := removeDynamicConverter(converters); isDynamic {
		rows := Rows{itr: rows}
		return frameDynamic(rows, rowLimit, types, converters)
	}

	names, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	scanRow, err := MakeScanRow(types, names, converters...)
	if err != nil {
		return nil, err
	}

	frame := NewFrame(names, scanRow.Converters...)

	var i int64
	for {
		// first iterate over rows may be nop if not switched result set to next
		for rows.Next() {
			if i == rowLimit {
				frame.AppendNotices(data.Notice{
					Severity: data.NoticeSeverityWarning,
					Text:     fmt.Sprintf("Results have been limited to %v because the SQL row limit was reached", rowLimit),
				})
				break
			}

			r := scanRow.NewScannableRow()
			if err := rows.Scan(r...); err != nil {
				return nil, err
			}

			if err := Append(frame, r, scanRow.Converters...); err != nil {
				return nil, err
			}

			i++
		}
		if i == rowLimit || !rows.NextResultSet() {
			break
		}
	}

	if err := rows.Err(); err != nil {
		return frame, backend.DownstreamError(err)
	}

	return frame, nil
}

// FrameFromRowsWithContext is an enhanced version of FrameFromRows that adds observability and cancellation support.
//
// Compared to FrameFromRows:
//   - Accepts a context.Context, enabling support for cancellation and timeouts.
//   - Emits Prometheus metrics for rows and cells (total and per-query histograms).
//   - Adds OpenTelemetry tracing with a span named "FrameFromRows", including row and cell counts.
//   - Aborts processing early if the context is canceled (e.g. timeout or client disconnect).
//
// Callers can use sqlutil.ContextWithMetricLabels(ctx, map[string]string{...}) to attach metric labels.
// Allowed labels: "query_type", "datasource_type".
func FrameFromRowsWithContext(ctx context.Context, rows *sql.Rows, rowLimit int64, converters ...Converter) (*data.Frame, error) {
	labels := getMetricLabels(ctx)

	var (
		rowCount  int64
		cellCount int64
	)

	// Start OpenTelemetry tracing span
	tracer := otel.Tracer("grafana/sqlutil")
	ctx, span := tracer.Start(ctx, "FrameFromRowsWithContext")
	defer span.End()

	// Emit metrics + span attributes at the end
	defer func() {
		rowsProcessed.With(labels).Add(float64(rowCount))
		rowCountHistogram.With(labels).Observe(float64(rowCount))

		cellsProcessed.With(labels).Add(float64(cellCount))
		cellCountHistogram.With(labels).Observe(float64(cellCount))

		span.SetAttributes(
			attribute.Int64("sqlutil.row_count", rowCount),
			attribute.Int64("sqlutil.cell_count", cellCount),
		)
	}()

	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	if isDynamic, converters := removeDynamicConverter(converters); isDynamic {
		rows := Rows{itr: rows}
		return frameDynamic(rows, rowLimit, types, converters)
	}

	names, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	scanRow, err := MakeScanRow(types, names, converters...)
	if err != nil {
		return nil, err
	}

	frame := NewFrame(names, scanRow.Converters...)

	for {
		for rows.Next() {
			// Abort if context was cancelled
			if ctx.Err() != nil {
				frame.AppendNotices(data.Notice{
					Severity: data.NoticeSeverityWarning,
					Text:     "Query was cancelled",
				})
				span.SetStatus(codes.Error, "context canceled")
				return frame, nil
			}

			if rowCount == rowLimit {
				frame.AppendNotices(data.Notice{
					Severity: data.NoticeSeverityWarning,
					Text:     fmt.Sprintf("Results have been limited to %v because the SQL row limit was reached", rowLimit),
				})
				return frame, nil
			}

			r := scanRow.NewScannableRow()
			if err := rows.Scan(r...); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}

			if err := Append(frame, r, scanRow.Converters...); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}

			rowCount++
			cellCount += int64(len(r))
		}

		if rowCount == rowLimit || !rows.NextResultSet() {
			break
		}
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return frame, backend.DownstreamError(err)
	}

	return frame, nil
}
