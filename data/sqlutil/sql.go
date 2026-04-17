package sqlutil

import (
	"database/sql"
	"fmt"
	"math"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
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
	return frameFromRows(rows, rowLimit, 0, converters...)
}

// FrameFromRowsWithCapacity is like FrameFromRows but reserves capacity for the
// expected number of rows on every Field, eliminating repeated slice growth on
// large result sets. Pass the row count returned by the database (or an upper
// bound). A capacity of 0 behaves identically to FrameFromRows.
func FrameFromRowsWithCapacity(rows *sql.Rows, rowLimit int64, capacity int, converters ...Converter) (*data.Frame, error) {
	return frameFromRows(rows, rowLimit, capacity, converters...)
}

func frameFromRows(rows *sql.Rows, rowLimit int64, capacity int, converters ...Converter) (*data.Frame, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	// Set rowLimit to the maximum possible value if no limit is specified (negative value)
	if rowLimit < 0 {
		rowLimit = math.MaxInt64
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
	if capacity > 0 {
		frame.SetRowCapacity(capacity)
	}

	// Allocate scan and converted-value buffers once, outside the per-row loop.
	// rows.Scan writes through the pointers in scannable; converters read those
	// pointed-to values and emit converted values into converted before they are
	// appended via frame.AppendRow. Both buffers are safe to reuse because the
	// converters either return a value-copy (DefaultConverterFunc via
	// reflect.Value.Interface()) or a pointer to a fresh local copy (the
	// Null* converters), so no aliasing leaks across iterations.
	scannable := scanRow.NewScannableRow()
	converted := make([]interface{}, len(scannable))

	var i int64

outer:
	for i < rowLimit {
		// first iterate over rows may be nop if not switched result set to next
		for rows.Next() {
			if err := rows.Scan(scannable...); err != nil {
				return nil, err
			}

			if err := appendConvertedRow(frame, scannable, converted, scanRow.Converters); err != nil {
				return nil, err
			}

			i++
			if i == rowLimit {
				frame.AppendNotices(data.Notice{
					Severity: data.NoticeSeverityWarning,
					Text:     fmt.Sprintf("Results have been limited to %v because the SQL row limit was reached", rowLimit),
				})
				break outer
			}
		}

		if !rows.NextResultSet() {
			break
		}
	}

	if err := rows.Err(); err != nil {
		return frame, backend.DownstreamError(err)
	}

	return frame, nil
}

// appendConvertedRow runs the per-column converters over a scanned row,
// writing the converted values into the caller-supplied converted buffer
// before passing them to frame.AppendRow. It exists so FrameFromRows can
// reuse a single converted buffer across rows; the public Append function
// allocates a fresh buffer per call and is preserved for back-compat.
func appendConvertedRow(frame *data.Frame, scanned []interface{}, converted []interface{}, converters []Converter) error {
	for i, v := range scanned {
		conv := &converters[i]
		if conv.FrameConverter.ConvertWithColumn != nil {
			value, err := conv.FrameConverter.ConvertWithColumn(v, conv.colType)
			if err != nil {
				return err
			}
			converted[i] = value
			continue
		}
		value, err := conv.FrameConverter.ConverterFunc(v)
		if err != nil {
			return err
		}
		converted[i] = value
	}
	frame.AppendRow(converted...)
	return nil
}
