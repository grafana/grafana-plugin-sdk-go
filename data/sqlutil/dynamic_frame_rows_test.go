package sqlutil

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// sliceRows is a RowIterator over in-memory rows. Err reports err once the rows run out.
type sliceRows struct {
	rows [][]any
	next int
	err  error
}

func (r *sliceRows) Next() bool {
	r.next++
	return r.next <= len(r.rows)
}

func (r *sliceRows) Scan(dest ...any) error {
	for i, d := range dest {
		*d.(*any) = r.rows[r.next-1][i]
	}
	return nil
}

func (r *sliceRows) Err() error { return r.err }

// sliceRowsNoErr hides Err, like an iterator that predates it.
type sliceRowsNoErr struct{ itr *sliceRows }

func (r sliceRowsNoErr) Next() bool             { return r.itr.Next() }
func (r sliceRowsNoErr) Scan(dest ...any) error { return r.itr.Scan(dest...) }

func columnTypes(n int) []*sql.ColumnType {
	types := make([]*sql.ColumnType, n)
	for i := range types {
		types[i] = &sql.ColumnType{}
	}
	return types
}

// limitNotices counts the row limit notices that FrameFromRows adds on the static path.
func limitNotices(frame *data.Frame, limit int64) int {
	if frame.Meta == nil {
		return 0
	}
	want := data.Notice{
		Severity: data.NoticeSeverityWarning,
		Text:     fmt.Sprintf("Results have been limited to %v because the SQL row limit was reached", limit),
	}
	n := 0
	for _, notice := range frame.Meta.Notices {
		if notice == want {
			n++
		}
	}
	return n
}

func TestFrameDynamicRowLimit(t *testing.T) {
	full := [][]any{{"a", 1.0}, {"b", 2.0}, {"c", 3.0}, {"d", 4.0}, {"e", 5.0}}
	leadingNils := [][]any{{"a", nil}, {"b", nil}, {"c", 3.0}, {"d", 4.0}, {"e", 5.0}}

	tests := []struct {
		name        string
		rows        [][]any
		limit       int64
		wantRows    int
		wantNotices int
	}{
		{name: "no limit", rows: full, limit: -1, wantRows: 5},
		{name: "limit 0", rows: full, limit: 0, wantRows: 0},
		{name: "limit 1", rows: full, limit: 1, wantRows: 1, wantNotices: 1},
		{name: "limit below the row count", rows: full, limit: 3, wantRows: 3, wantNotices: 1},
		{name: "limit equal to the row count", rows: full, limit: 5, wantRows: 5, wantNotices: 1},
		{name: "limit above the row count", rows: full, limit: 10, wantRows: 5},
		{name: "limit after leading nulls", rows: leadingNils, limit: 4, wantRows: 4, wantNotices: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			frame, err := frameDynamic(Rows{itr: &sliceRows{rows: tc.rows}}, tc.limit, columnTypes(2), nil)
			if err != nil {
				t.Fatalf("frameDynamic() error = %v", err)
			}
			if got := frame.Rows(); got != tc.wantRows {
				t.Errorf("frameDynamic() rows = %d, want %d", got, tc.wantRows)
			}
			if got := limitNotices(frame, tc.limit); got != tc.wantNotices {
				t.Errorf("frameDynamic() row limit notices = %d, want %d", got, tc.wantNotices)
			}
		})
	}
}

func TestFrameDynamicRowsErr(t *testing.T) {
	errReset := errors.New("connection reset")
	rows := [][]any{{"a"}, {"b"}}

	tests := []struct {
		name    string
		itr     RowIterator
		wantErr error
	}{
		{name: "iterator without Err", itr: sliceRowsNoErr{itr: &sliceRows{rows: rows, err: errReset}}},
		{name: "Err returns nil", itr: &sliceRows{rows: rows}},
		{name: "Err returns an error", itr: &sliceRows{rows: rows, err: errReset}, wantErr: errReset},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			frame, err := frameDynamic(Rows{itr: tc.itr}, -1, columnTypes(1), nil)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("frameDynamic() error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil && !backend.IsDownstreamError(err) {
				t.Errorf("frameDynamic() error = %v, want a downstream error", err)
			}
			if frame == nil || frame.Rows() != len(rows) {
				t.Fatalf("frameDynamic() frame = %v, want %d rows", frame, len(rows))
			}
		})
	}
}
