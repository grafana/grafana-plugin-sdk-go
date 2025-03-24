package sqlutil_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
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

	rows       [][]interface{}
	currentRow int
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

type cancellingRows struct {
	baseRows
	ctx        context.Context
	cancel     context.CancelFunc
	rows       [][]interface{}
	currentRow int
}

func (r *cancellingRows) Next(dest []driver.Value) error {
	r.currentRow++
	if r.currentRow >= len(r.rows) {
		return io.EOF
	}

	// Cancel context after yielding first row
	if r.currentRow == 1 {
		r.cancel()
	}

	for i := range dest {
		dest[i] = r.rows[r.currentRow][i]
	}
	return nil
}

func makeCancellingRows(ctx context.Context, cancel context.CancelFunc, columnNames []string, data ...[]interface{}) *sql.Rows {
	db := &fakeDB{
		rows: &cancellingRows{
			baseRows:   baseRows{columnNames: columnNames},
			ctx:        ctx,
			cancel:     cancel,
			rows:       data,
			currentRow: -1,
		},
	}
	rows, _ := sql.OpenDB(db).Query("")
	return rows
}

func TestFrameFromRows(t *testing.T) {
	ptr := func(s string) *string {
		return &s
	}
	for _, tt := range []struct {
		name       string
		makeRows   func() *sql.Rows
		rowLimit   int64
		converters []sqlutil.Converter
		frame      *data.Frame
		err        bool
	}{
		{
			name: "rows not implements driver.RowsNextResultSet",
			makeRows: func() *sql.Rows {
				return makeSingleResultSet(
					[]string{"a", "b", "c"},
					[]interface{}{1, 2, 3},
					[]interface{}{4, 5, 6},
					[]interface{}{7, 8, 9},
				)
			},
			rowLimit:   100,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{ptr("1"), ptr("4"), ptr("7")}),
					data.NewField("b", nil, []*string{ptr("2"), ptr("5"), ptr("8")}),
					data.NewField("c", nil, []*string{ptr("3"), ptr("6"), ptr("9")}),
				},
			},
			err: false,
		},
		{
			name: "rows not implements driver.RowsNextResultSet, limit reached",
			makeRows: func() *sql.Rows {
				return makeSingleResultSet(
					[]string{"a", "b", "c"},
					[]interface{}{1, 2, 3},
					[]interface{}{4, 5, 6},
					[]interface{}{7, 8, 9},
				)
			},
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
			err: false,
		},
		{
			name: "rows implements driver.RowsNextResultSet, but contains only one result set",
			makeRows: func() *sql.Rows {
				return makeMultipleResultSets(
					[]string{"a", "b", "c"},
					[][]interface{}{
						{1, 2, 3},
						{4, 5, 6},
						{7, 8, 9},
					},
				)
			},
			rowLimit:   100,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{ptr("1"), ptr("4"), ptr("7")}),
					data.NewField("b", nil, []*string{ptr("2"), ptr("5"), ptr("8")}),
					data.NewField("c", nil, []*string{ptr("3"), ptr("6"), ptr("9")}),
				},
			},
			err: false,
		},
		{
			name: "rows implements driver.RowsNextResultSet, but contains more then one result set",
			makeRows: func() *sql.Rows {
				return makeMultipleResultSets(
					[]string{"a", "b", "c"},
					[][]interface{}{
						{1, 2, 3},
						{4, 5, 6},
					},
					[][]interface{}{
						{7, 8, 9},
					},
				)
			},
			rowLimit:   100,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{ptr("1"), ptr("4"), ptr("7")}),
					data.NewField("b", nil, []*string{ptr("2"), ptr("5"), ptr("8")}),
					data.NewField("c", nil, []*string{ptr("3"), ptr("6"), ptr("9")}),
				},
			},
			err: false,
		},
		{
			name: "rows implements driver.RowsNextResultSet, limit reached",
			makeRows: func() *sql.Rows {
				return makeMultipleResultSets(
					[]string{"a", "b", "c"},
					[][]interface{}{
						{1, 2, 3},
						{4, 5, 6},
					},
					[][]interface{}{
						{7, 8, 9},
					},
				)
			},
			rowLimit:   2,
			converters: nil,
			frame: &data.Frame{
				Fields: []*data.Field{
					data.NewField("a", nil, []*string{ptr("1"), ptr("4")}),
					data.NewField("b", nil, []*string{ptr("2"), ptr("5")}),
					data.NewField("c", nil, []*string{ptr("3"), ptr("6")}),
				},
			},
			err: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rows := tt.makeRows()
			frame, err := sqlutil.FrameFromRows(rows, tt.rowLimit, tt.converters...)
			require.NoError(t, rows.Err())
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.frame, frame)
			}
		})
		t.Run(tt.name+" (FrameFromRowsWithContext)", func(t *testing.T) {
			rows := tt.makeRows()
			frame, err := sqlutil.FrameFromRowsWithContext(context.Background(), rows, tt.rowLimit, tt.converters...)
			require.NoError(t, rows.Err())
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.frame, frame)
			}
		})
	}
}

func TestFrameFromRowsWithContext_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	columnNames := []string{"x"}
	in := [][]interface{}{
		{1},
		{2}, // should be skipped due to context cancel
	}

	rows := makeCancellingRows(ctx, cancel, columnNames, in...)

	frame, err := sqlutil.FrameFromRowsWithContext(ctx, rows, 100)
	require.NoError(t, err)
	require.NotNil(t, frame)

	require.Len(t, frame.Fields, 1)
	require.Equal(t, 1, frame.Fields[0].Len()) // Only 1 row processed

	require.NotNil(t, frame.Meta)
	require.NotEmpty(t, frame.Meta.Notices)

	notice := frame.Meta.Notices[0]
	require.Equal(t, data.NoticeSeverityWarning, notice.Severity)
	require.Contains(t, notice.Text, "cancelled")
	require.Empty(t, notice.Link)
}
