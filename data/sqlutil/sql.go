package sqlutil

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var (
	// ErrorJSON is returned when the query couldn't be unmarshaled
	ErrorJSON = errors.New("error unmarshaling query JSON to the Query Model")
	// ErrorQuery is returned when the query could not complete / execute
	ErrorQuery = errors.New("error querying the database")
	// ErrorNoResults is returned if there were no results returned
	ErrorNoResults = errors.New("no results returned from query")
)

// Connection represents a SQL connection and is satisfied by the *sql.DB type
// For now, we only add the functions that we need/actively use.
type Connection interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

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

	if isDynamic(converters) {
		rows := Rows{itr: rows}
		return frameDynamic(rows, rowLimit, types, converters)
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

	var i int64
	for rows.Next() {
		if i == rowLimit {
			frame.AppendNotices(data.Notice{
				Severity: data.NoticeSeverityWarning,
				Text:     fmt.Sprintf("Results have been limited to %v because the SQL row limit was reached", rowLimit),
			})
			break
		}

		r := scanner.NewScannableRow()
		if err := rows.Scan(r...); err != nil {
			return nil, err
		}

		if err := Append(frame, r, converters...); err != nil {
			return nil, err
		}

		i++
	}

	if err := rows.Err(); err != nil {
		return frame, err
	}

	return frame, nil
}

// QueryDB sends the query to the connection and converts the rows to a dataframe.
func QueryDB(ctx context.Context, db Connection, converters []Converter, fillMode *data.FillMissing, query *Query, args ...interface{}) (data.Frames, error) {
	// Query the rows from the database
	rows, err := db.QueryContext(ctx, query.RawSQL, args...)
	if err != nil {
		errType := ErrorQuery
		if errors.Is(err, context.Canceled) {
			errType = context.Canceled
		}

		return ErrorFrameFromQuery(query), fmt.Errorf("%w: %s", errType, err.Error())
	}

	// Check for an error response
	if err := rows.Err(); err != nil {
		if err == sql.ErrNoRows {
			// Should we even response with an error here?
			// The panel will simply show "no data"
			return ErrorFrameFromQuery(query), fmt.Errorf("%s: %w", "No results from query", err)
		}
		return ErrorFrameFromQuery(query), fmt.Errorf("%s: %w", "Error response from database", err)
	}

	defer func() {
		if err := rows.Close(); err != nil {
			backend.Logger.Error(err.Error())
		}
	}()

	// Convert the response to frames
	res, err := getFrames(rows, -1, converters, fillMode, query)
	if err != nil {
		return ErrorFrameFromQuery(query), fmt.Errorf("%w: %s", err, "Could not process SQL results")
	}

	return res, nil
}

func getFrames(rows *sql.Rows, limit int64, converters []Converter, fillMode *data.FillMissing, query *Query) (data.Frames, error) {
	frame, err := FrameFromRows(rows, limit, converters...)
	if err != nil {
		return nil, err
	}
	frame.Name = query.RefID
	if frame.Meta == nil {
		frame.Meta = &data.FrameMeta{}
	}

	frame.Meta.ExecutedQueryString = query.RawSQL
	frame.Meta.PreferredVisualization = data.VisTypeGraph

	if query.Format == FormatOptionTable {
		frame.Meta.PreferredVisualization = data.VisTypeTable
		return data.Frames{frame}, nil
	}

	if query.Format == FormatOptionLogs {
		frame.Meta.PreferredVisualization = data.VisTypeLogs
		return data.Frames{frame}, nil
	}

	count, err := frame.RowLen()

	if err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, ErrorNoResults
	}

	if frame.TimeSeriesSchema().Type == data.TimeSeriesTypeLong {
		frame, err := data.LongToWide(frame, fillMode)
		if err != nil {
			return nil, err
		}
		return data.Frames{frame}, nil
	}

	return data.Frames{frame}, nil
}
