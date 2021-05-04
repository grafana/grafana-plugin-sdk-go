package sqlutil

import (
	"database/sql"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func FrameFromRows(rows *sql.Rows, rowLimit int64, converters ...Converter) (*data.Frame, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	names, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	scanner, converters, err := MakeScanRow(types, names, converters...)
	if err != nil {
		return nil, err
	}

	frame := NewFrame(names, converters...)

	for rows.Next() {
		r := scanner.NewScannableRow()
		if err := rows.Scan(r...); err != nil {
			return nil, err
		}

		if err := Append(frame, r, converters...); err != nil {
			return nil, err
		}
	}

	return frame, nil
}
