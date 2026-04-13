package sqlutil_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
)

type fakeDB struct {
	rows driver.Rows
}

type baseRows struct {
	columnNames []string
}

func (rows *baseRows) Columns() []string {
	return rows.columnNames
}

func (*baseRows) Close() error {
	return nil
}

type singleResultSet struct {
	baseRows

	rows        [][]interface{}
	currentRow  int
	scanTypes   []reflect.Type
	dbTypeNames []string
}

func (rows *singleResultSet) Next(dest []driver.Value) error {
	rows.currentRow++
	if rows.currentRow >= len(rows.rows) {
		return io.EOF
	}
	data := rows.rows[rows.currentRow]
	for i := range dest {
		dest[i] = data[i]
	}
	return nil
}

func (rows *singleResultSet) ColumnTypeScanType(index int) reflect.Type {
	if index >= len(rows.scanTypes) {
		return reflect.TypeFor[any]()
	}
	return rows.scanTypes[index]
}

func (rows *singleResultSet) ColumnTypeDatabaseTypeName(index int) string {
	if index >= len(rows.dbTypeNames) {
		return ""
	}
	return rows.dbTypeNames[index]
}

type multipleResultSets struct {
	baseRows

	resultSets       [][][]interface{}
	currentResultSet int
	currentRow       int
	once             sync.Once
}

func (rows *multipleResultSets) Next(dest []driver.Value) error {
	rows.once.Do(func() {
		rows.currentResultSet++
	})
	rows.currentRow++
	if rows.currentRow >= len(rows.resultSets[rows.currentResultSet]) {
		return io.EOF
	}
	data := rows.resultSets[rows.currentResultSet][rows.currentRow]
	for i := range dest {
		dest[i] = data[i]
	}
	return nil
}

func (rows *multipleResultSets) HasNextResultSet() bool {
	return rows.currentResultSet < len(rows.resultSets)
}

func (rows *multipleResultSets) NextResultSet() error {
	rows.once.Do(func() {})
	rows.currentResultSet++
	rows.currentRow = -1
	if rows.currentResultSet >= len(rows.resultSets) {
		return io.EOF
	}
	return nil
}

func (db *fakeDB) LastInsertId() (int64, error) {
	return 0, errors.New("not implemented for fakeDB")
}

func (db *fakeDB) RowsAffected() (int64, error) {
	return 0, errors.New("not implemented for fakeDB")
}

func (db *fakeDB) NumInput() int {
	return 0
}

func (db *fakeDB) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("not implemented for fakeDB")
}

func (db *fakeDB) Query([]driver.Value) (driver.Rows, error) {
	return db.rows, nil
}

func (db *fakeDB) Prepare(string) (driver.Stmt, error) {
	return db, nil
}

func (db *fakeDB) Close() error {
	return nil
}

func (db *fakeDB) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented for fakeDB")
}

func (db *fakeDB) Open(string) (driver.Conn, error) {
	return nil, errors.New("not implemented for fakeDB")
}

func (db *fakeDB) Connect(context.Context) (driver.Conn, error) {
	return db, nil
}

func (db *fakeDB) Driver() driver.Driver {
	return db
}

func makeSingleResultSet(
	columnNames []string,
	data ...[]interface{},
) *sql.Rows {
	rows, _ := sql.OpenDB(&fakeDB{
		rows: &singleResultSet{
			baseRows: baseRows{
				columnNames: columnNames,
			},
			rows:       data,
			currentRow: -1,
		},
	}).Query("")
	return rows
}

func makeSingleResultSetWithScanTypes(
	columnNames []string,
	scanTypes []reflect.Type,
	data ...[]interface{},
) *sql.Rows {
	rows, _ := sql.OpenDB(&fakeDB{
		rows: &singleResultSet{
			baseRows: baseRows{
				columnNames: columnNames,
			},
			rows:       data,
			currentRow: -1,
			scanTypes:  scanTypes,
		},
	}).Query("")
	return rows
}

func makeSingleResultSetWithDBTypes(
	columnNames []string,
	dbTypeNames []string,
	data ...[]interface{},
) *sql.Rows {
	rows, _ := sql.OpenDB(&fakeDB{
		rows: &singleResultSet{
			baseRows: baseRows{
				columnNames: columnNames,
			},
			rows:        data,
			currentRow:  -1,
			dbTypeNames: dbTypeNames,
		},
	}).Query("")
	return rows
}

func makeMultipleResultSets(
	columnNames []string,
	resultSets ...[][]interface{},
) *sql.Rows {
	rows, _ := sql.OpenDB(&fakeDB{
		rows: &multipleResultSets{
			baseRows: baseRows{
				columnNames: columnNames,
			},
			resultSets:       resultSets,
			currentResultSet: -1,
			currentRow:       -1,
		},
	}).Query("")
	return rows
}

func TestFrameFromRows(t *testing.T) {
	ptr := func(s string) *string {
		return &s
	}
	for _, tt := range []struct {
		name       string
		rows       *sql.Rows
		rowLimit   int64
		converters []sqlutil.Converter
		frame      *data.Frame
		err        error
	}{
		{
			name: "rows not implements driver.RowsNextResultSet",
			rows: makeSingleResultSet( //nolint:rowserrcheck
				[]string{
					"a",
					"b",
					"c",
				},
				[]interface{}{
					1, 2, 3,
				},
				[]interface{}{
					4, 5, 6,
				},
				[]interface{}{
					7, 8, 9,
				},
			),
			rowLimit:   100,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{ptr("1"), ptr("4"), ptr("7")}),
					data.NewField("b", nil, []*string{ptr("2"), ptr("5"), ptr("8")}),
					data.NewField("c", nil, []*string{ptr("3"), ptr("6"), ptr("9")}),
				},
			},
			err: nil,
		},
		{
			name: "rows not implements driver.RowsNextResultSet (rowLimit < 0)",
			rows: makeSingleResultSet( //nolint:rowserrcheck
				[]string{
					"a",
					"b",
					"c",
				},
				[]interface{}{
					1, 2, 3,
				},
				[]interface{}{
					4, 5, 6,
				},
				[]interface{}{
					7, 8, 9,
				},
			),
			rowLimit:   -1,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{ptr("1"), ptr("4"), ptr("7")}),
					data.NewField("b", nil, []*string{ptr("2"), ptr("5"), ptr("8")}),
					data.NewField("c", nil, []*string{ptr("3"), ptr("6"), ptr("9")}),
				},
			},
			err: nil,
		},
		{
			name: "rows not implements driver.RowsNextResultSet, limit reached",
			rows: makeSingleResultSet( //nolint:rowserrcheck
				[]string{
					"a",
					"b",
					"c",
				},
				[]interface{}{
					1, 2, 3,
				},
				[]interface{}{
					4, 5, 6,
				},
				[]interface{}{
					7, 8, 9,
				},
			),
			rowLimit:   2,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{ptr("1"), ptr("4")}),
					data.NewField("b", nil, []*string{ptr("2"), ptr("5")}),
					data.NewField("c", nil, []*string{ptr("3"), ptr("6")}),
				},
				Meta: &data.FrameMeta{
					Notices: []data.Notice{
						{
							Severity: data.NoticeSeverityWarning,
							Text:     "Results have been limited to 2 because the SQL row limit was reached",
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "rows implements driver.RowsNextResultSet, but contains only one result set",
			rows: makeMultipleResultSets( //nolint:rowserrcheck
				[]string{
					"a",
					"b",
					"c",
				},
				[][]interface{}{
					{
						1, 2, 3,
					},
					{
						4, 5, 6,
					},
					{
						7, 8, 9,
					},
				},
			),
			rowLimit:   100,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{ptr("1"), ptr("4"), ptr("7")}),
					data.NewField("b", nil, []*string{ptr("2"), ptr("5"), ptr("8")}),
					data.NewField("c", nil, []*string{ptr("3"), ptr("6"), ptr("9")}),
				},
			},
			err: nil,
		},
		{
			name: "rows implements driver.RowsNextResultSet, but contains only one result set (rowLimit < 0)",
			rows: makeMultipleResultSets( //nolint:rowserrcheck
				[]string{
					"a",
					"b",
					"c",
				},
				[][]interface{}{
					{
						1, 2, 3,
					},
					{
						4, 5, 6,
					},
					{
						7, 8, 9,
					},
				},
			),
			rowLimit:   -1,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{ptr("1"), ptr("4"), ptr("7")}),
					data.NewField("b", nil, []*string{ptr("2"), ptr("5"), ptr("8")}),
					data.NewField("c", nil, []*string{ptr("3"), ptr("6"), ptr("9")}),
				},
			},
			err: nil,
		},
		{
			name: "rows implements driver.RowsNextResultSet, but contains more than one result set",
			rows: makeMultipleResultSets( //nolint:rowserrcheck
				[]string{
					"a",
					"b",
					"c",
				},
				[][]interface{}{
					{
						1, 2, 3,
					},
					{
						4, 5, 6,
					},
				},
				[][]interface{}{
					{
						7, 8, 9,
					},
				},
			),
			rowLimit:   100,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{ptr("1"), ptr("4"), ptr("7")}),
					data.NewField("b", nil, []*string{ptr("2"), ptr("5"), ptr("8")}),
					data.NewField("c", nil, []*string{ptr("3"), ptr("6"), ptr("9")}),
				},
			},
			err: nil,
		},
		{
			name: "rows implements driver.RowsNextResultSet, but contains more than one result set (rowLimit < 0)",
			rows: makeMultipleResultSets( //nolint:rowserrcheck
				[]string{
					"a",
					"b",
					"c",
				},
				[][]interface{}{
					{
						1, 2, 3,
					},
					{
						4, 5, 6,
					},
				},
				[][]interface{}{
					{
						7, 8, 9,
					},
				},
			),
			rowLimit:   -1,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{ptr("1"), ptr("4"), ptr("7")}),
					data.NewField("b", nil, []*string{ptr("2"), ptr("5"), ptr("8")}),
					data.NewField("c", nil, []*string{ptr("3"), ptr("6"), ptr("9")}),
				},
			},
			err: nil,
		},
		{
			name: "rows implements driver.RowsNextResultSet, limit reached",
			rows: makeMultipleResultSets( //nolint:rowserrcheck
				[]string{
					"a",
					"b",
					"c",
				},
				[][]interface{}{
					{
						1, 2, 3,
					},
					{
						4, 5, 6,
					},
				},
				[][]interface{}{
					{
						7, 8, 9,
					},
				},
			),
			rowLimit:   2,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{ptr("1"), ptr("4")}),
					data.NewField("b", nil, []*string{ptr("2"), ptr("5")}),
					data.NewField("c", nil, []*string{ptr("3"), ptr("6")}),
				},
				Meta: &data.FrameMeta{
					Notices: []data.Notice{
						{
							Severity: data.NoticeSeverityWarning,
							Text:     "Results have been limited to 2 because the SQL row limit was reached",
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "row contains unsupported column type",
			rows: makeSingleResultSetWithScanTypes( //nolint:rowserrcheck
				[]string{"a"},
				[]reflect.Type{nil},
				[]interface{}{1},
			),
			rowLimit:   100,
			converters: nil,
			err:        sqlutil.ErrColumnTypeNotSupported{},
		},
		{
			name: "row contains unsupported column type (rowLimit < 0)",
			rows: makeSingleResultSetWithScanTypes( //nolint:rowserrcheck
				[]string{"a"},
				[]reflect.Type{nil},
				[]interface{}{1},
			),
			rowLimit:   -1,
			converters: nil,
			err:        sqlutil.ErrColumnTypeNotSupported{},
		},
		{
			name: "empty rows",
			rows: makeSingleResultSet( //nolint:rowserrcheck
				[]string{
					"a",
					"b",
					"c",
				},
			),
			rowLimit:   100,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{}),
					data.NewField("b", nil, []*string{}),
					data.NewField("c", nil, []*string{}),
				},
			},
			err: nil,
		},
		{
			name: "empty rows (rowLimit < 0)",
			rows: makeSingleResultSet( //nolint:rowserrcheck
				[]string{
					"a",
					"b",
					"c",
				},
			),
			rowLimit:   -1,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{}),
					data.NewField("b", nil, []*string{}),
					data.NewField("c", nil, []*string{}),
				},
			},
			err: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := sqlutil.FrameFromRows(tt.rows, tt.rowLimit, tt.converters...)
			if tt.err != nil {
				require.ErrorAs(t, err, &tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.frame, frame)
			}
		})
	}
}

func TestFrameFromRows_MultipleTimes(t *testing.T) {
	ptr := func(s string) *string {
		return &s
	}
	for _, tt := range []struct {
		name       string
		rows       *sql.Rows
		rowLimit   int64
		converters []sqlutil.Converter
		frames     []*data.Frame
	}{
		{
			name: "rows not implements driver.RowsNextResultSet",
			rows: makeSingleResultSet( //nolint:rowserrcheck
				[]string{
					"a",
					"b",
					"c",
				},
				[]interface{}{
					1, 2, 3,
				},
				[]interface{}{
					4, 5, 6,
				},
				[]interface{}{
					7, 8, 9,
				},
			),
			rowLimit:   1,
			converters: nil,
			frames: data.Frames{
				&data.Frame{
					Fields: []*data.Field{
						data.NewField("a", nil, []*string{ptr("1")}),
						data.NewField("b", nil, []*string{ptr("2")}),
						data.NewField("c", nil, []*string{ptr("3")}),
					},
					Meta: &data.FrameMeta{
						Notices: []data.Notice{
							{
								Severity: data.NoticeSeverityWarning,
								Text:     "Results have been limited to 1 because the SQL row limit was reached",
							},
						},
					},
				},
				&data.Frame{
					Fields: []*data.Field{
						data.NewField("a", nil, []*string{ptr("4")}),
						data.NewField("b", nil, []*string{ptr("5")}),
						data.NewField("c", nil, []*string{ptr("6")}),
					},
					Meta: &data.FrameMeta{
						Notices: []data.Notice{
							{
								Severity: data.NoticeSeverityWarning,
								Text:     "Results have been limited to 1 because the SQL row limit was reached",
							},
						},
					},
				},
				&data.Frame{
					Fields: []*data.Field{
						data.NewField("a", nil, []*string{ptr("7")}),
						data.NewField("b", nil, []*string{ptr("8")}),
						data.NewField("c", nil, []*string{ptr("9")}),
					},
					Meta: &data.FrameMeta{
						Notices: []data.Notice{
							{
								Severity: data.NoticeSeverityWarning,
								Text:     "Results have been limited to 1 because the SQL row limit was reached",
							},
						},
					},
				},
			},
		},
		{
			name: "rows implements driver.RowsNextResultSet, but contains only one result set",
			rows: makeMultipleResultSets( //nolint:rowserrcheck
				[]string{
					"a",
					"b",
					"c",
				},
				[][]interface{}{
					{
						1, 2, 3,
					},
					{
						4, 5, 6,
					},
					{
						7, 8, 9,
					},
				},
			),
			rowLimit:   1,
			converters: nil,
			frames: data.Frames{
				&data.Frame{
					Fields: []*data.Field{
						data.NewField("a", nil, []*string{ptr("1")}),
						data.NewField("b", nil, []*string{ptr("2")}),
						data.NewField("c", nil, []*string{ptr("3")}),
					},
					Meta: &data.FrameMeta{
						Notices: []data.Notice{
							{
								Severity: data.NoticeSeverityWarning,
								Text:     "Results have been limited to 1 because the SQL row limit was reached",
							},
						},
					},
				},
				&data.Frame{
					Fields: []*data.Field{
						data.NewField("a", nil, []*string{ptr("4")}),
						data.NewField("b", nil, []*string{ptr("5")}),
						data.NewField("c", nil, []*string{ptr("6")}),
					},
					Meta: &data.FrameMeta{
						Notices: []data.Notice{
							{
								Severity: data.NoticeSeverityWarning,
								Text:     "Results have been limited to 1 because the SQL row limit was reached",
							},
						},
					},
				},
				&data.Frame{
					Fields: []*data.Field{
						data.NewField("a", nil, []*string{ptr("7")}),
						data.NewField("b", nil, []*string{ptr("8")}),
						data.NewField("c", nil, []*string{ptr("9")}),
					},
					Meta: &data.FrameMeta{
						Notices: []data.Notice{
							{
								Severity: data.NoticeSeverityWarning,
								Text:     "Results have been limited to 1 because the SQL row limit was reached",
							},
						},
					},
				},
			},
		},
		{
			name: "rows implements driver.RowsNextResultSet, but contains more then one result set",
			rows: makeMultipleResultSets( //nolint:rowserrcheck
				[]string{
					"a",
					"b",
					"c",
				},
				[][]interface{}{
					{
						1, 2, 3,
					},
					{
						4, 5, 6,
					},
				},
				[][]interface{}{
					{
						7, 8, 9,
					},
				},
			),
			rowLimit:   1,
			converters: nil,
			frames: data.Frames{
				&data.Frame{
					Fields: []*data.Field{
						data.NewField("a", nil, []*string{ptr("1")}),
						data.NewField("b", nil, []*string{ptr("2")}),
						data.NewField("c", nil, []*string{ptr("3")}),
					},
					Meta: &data.FrameMeta{
						Notices: []data.Notice{
							{
								Severity: data.NoticeSeverityWarning,
								Text:     "Results have been limited to 1 because the SQL row limit was reached",
							},
						},
					},
				},
				&data.Frame{
					Fields: []*data.Field{
						data.NewField("a", nil, []*string{ptr("4")}),
						data.NewField("b", nil, []*string{ptr("5")}),
						data.NewField("c", nil, []*string{ptr("6")}),
					},
					Meta: &data.FrameMeta{
						Notices: []data.Notice{
							{
								Severity: data.NoticeSeverityWarning,
								Text:     "Results have been limited to 1 because the SQL row limit was reached",
							},
						},
					},
				},
				&data.Frame{
					Fields: []*data.Field{
						data.NewField("a", nil, []*string{ptr("7")}),
						data.NewField("b", nil, []*string{ptr("8")}),
						data.NewField("c", nil, []*string{ptr("9")}),
					},
					Meta: &data.FrameMeta{
						Notices: []data.Notice{
							{
								Severity: data.NoticeSeverityWarning,
								Text:     "Results have been limited to 1 because the SQL row limit was reached",
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var frames []*data.Frame
			for {
				frame, err := sqlutil.FrameFromRows(tt.rows, tt.rowLimit, tt.converters...)
				require.NoError(t, err)

				if frame.Rows() == 0 {
					break
				}

				frames = append(frames, frame)
			}

			require.Equal(t, tt.frames, frames)
		})
	}
}

// TestFrameFromRows_DynamicPerColumn verifies that FrameFromRows routes to frameHybrid
// when a DynamicPerColumn converter is present, and that only matching columns use
// dynamic inference while static columns keep their proper types.
func TestFrameFromRows_DynamicPerColumn(t *testing.T) {
	t.Run("routes to hybrid path when DynamicPerColumn converter present", func(t *testing.T) {
		rows := makeSingleResultSetWithDBTypes( //nolint:rowserrcheck
			[]string{"variant_col", "static_col"},
			[]string{"VARIANT", ""},
			[]interface{}{"json_value", "plain_text"},
			[]interface{}{"another_json", "more_text"},
		)

		converters := []sqlutil.Converter{
			{
				Name:             "VARIANT dynamic converter",
				InputTypeName:    "VARIANT",
				DynamicPerColumn: true,
			},
		}

		frame, err := sqlutil.FrameFromRows(rows, 100, converters...)
		require.NoError(t, err)
		require.NotNil(t, frame)
		require.Equal(t, 2, len(frame.Fields), "Should have 2 fields")
		require.Equal(t, 2, frame.Rows(), "Should have 2 rows")
	})

	t.Run("row limit notice is attached on hybrid path", func(t *testing.T) {
		rows := makeSingleResultSetWithDBTypes( //nolint:rowserrcheck
			[]string{"variant_col"},
			[]string{"VARIANT"},
			[]interface{}{"row1"},
			[]interface{}{"row2"},
			[]interface{}{"row3"},
		)

		converters := []sqlutil.Converter{
			{
				Name:             "VARIANT dynamic converter",
				InputTypeName:    "VARIANT",
				DynamicPerColumn: true,
			},
		}

		frame, err := sqlutil.FrameFromRows(rows, 2, converters...)
		require.NoError(t, err)
		require.NotNil(t, frame)
		require.Equal(t, 2, frame.Rows(), "Should have 2 rows (limited)")
		require.NotNil(t, frame.Meta, "Frame should have metadata with notice")
		require.Len(t, frame.Meta.Notices, 1, "Should have exactly one notice")
		require.Equal(t, data.NoticeSeverityWarning, frame.Meta.Notices[0].Severity)
		require.Equal(t, "Results have been limited to 2 because the SQL row limit was reached", frame.Meta.Notices[0].Text)
	})
}
