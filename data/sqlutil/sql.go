package sqlutil

import (
	"database/sql"
	"fmt"

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
func FrameFromRows(rows *sql.Rows, rowLimit int64, converters ...Converter) (*data.Frame, int64, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, 0, err
	}

	names, err := rows.Columns()
	if err != nil {
		return nil, 0, err
	}

	scanner, converters, err := MakeScanRow(types, names, converters...)
	if err != nil {
		return nil, 0, err
	}

	frame := NewFrame(names, converters...)

	var count int64
	for rows.Next() {
		if count == rowLimit {
			frame.AppendNotices(data.Notice{
				Severity: data.NoticeSeverityWarning,
				Text:     fmt.Sprintf("Results have been limited to %v because the SQL row limit was reached", rowLimit),
			})
			break
		}

		r := scanner.NewScannableRow()
		if err := rows.Scan(r...); err != nil {
			return nil, count, err
		}

		if err := Append(frame, r, converters...); err != nil {
			return nil, count, err
		}

		count++
	}

	return frame, count, nil
}
