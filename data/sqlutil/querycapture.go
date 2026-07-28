package sqlutil

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/querycapture"
)

// captureScannedRow renders one just-scanned row for a querycapture.ResultCapture.
//
// It is called with the scan buffer, i.e. the values as the driver produced them and
// before any Converter has run. That ordering is the point: a converter is supplied by
// the plugin, so a capture taken after conversion could not tell a datasource that
// returned the wrong rows from a converter that mangled the right ones -- which is the
// question the capture exists to answer.
func captureScannedRow(c *querycapture.ResultCapture, scanned []interface{}) {
	cells := make([]*string, len(scanned))
	for i, v := range scanned {
		cells[i] = renderCell(v)
	}
	c.AddRow(cells)
}

// columnNamesFromTypes returns the column names of a result set described by its
// column types, for the scanning paths that hold types rather than names.
func columnNamesFromTypes(types []*sql.ColumnType) []string {
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = t.Name()
	}
	return names
}

// renderCell renders a scanned value as the text a capture carries, or nil for NULL.
//
// Scan buffers hold pointers (rows.Scan writes through them) and driver types vary
// wildly -- sql.NullString, custom decimal types, []byte for strings -- so the value is
// unwrapped in the order that preserves meaning: dereference the scan pointer, ask a
// driver.Valuer what it holds (this is what distinguishes a NULL sql.NullString from an
// empty one), then render.
func renderCell(v interface{}) *string {
	if v == nil {
		return nil
	}

	// Unwrap scan pointers, including the *interface{} the dynamic framer scans into.
	for {
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Pointer {
			break
		}
		if rv.IsNil() {
			return nil
		}
		v = rv.Elem().Interface()
		if v == nil {
			return nil
		}
	}

	// sql.NullString and friends implement driver.Valuer: a NULL is a nil value, which
	// must render as NULL rather than as the zero value it wraps.
	if valuer, ok := v.(driver.Valuer); ok {
		got, err := valuer.Value()
		if err != nil {
			// A Valuer that fails says nothing useful about the row; record the failure
			// in place of the cell rather than dropping the row or failing the query.
			return strPtr(fmt.Sprintf("<value error: %v>", err))
		}
		if got == nil {
			return nil
		}
		v = got
	}

	switch t := v.(type) {
	case string:
		return strPtr(t)
	case []byte:
		// Drivers return text columns as bytes; rendering them as text is what makes the
		// capture comparable against the returned frames.
		return strPtr(string(t))
	case time.Time:
		// A fixed, sortable rendering, so a timestamp in the capture can be matched
		// against the same timestamp in the returned frames without reformatting.
		return strPtr(t.UTC().Format(time.RFC3339Nano))
	default:
		return strPtr(fmt.Sprintf("%v", v))
	}
}

func strPtr(s string) *string { return &s }
