package sqlutil_test

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
)

// makeWideRows is a small test helper that builds a *sql.Rows backed by
// the existing fakeDB infrastructure with a fixed schema and the requested
// number of rows. Each cell is a non-nil int64 so the DefaultConverterFunc
// path is exercised end-to-end without nullable handling muddying the
// measurement.
func makeWideRows(tb testing.TB, cols, rows int) *sql.Rows {
	tb.Helper()
	colNames := make([]string, cols)
	scanTypes := make([]reflect.Type, cols)
	int64T := reflect.TypeOf(int64(0))
	for i := range colNames {
		colNames[i] = string(rune('a' + i))
		scanTypes[i] = int64T
	}
	data := make([][]interface{}, rows)
	for r := range data {
		row := make([]interface{}, cols)
		for c := range row {
			row[c] = int64(r*cols + c)
		}
		data[r] = row
	}
	return makeSingleResultSetWithScanTypes(colNames, scanTypes, data...) //nolint:rowserrcheck
}

// TestDefaultConverterFunc_BehaviorPreserved is a regression guard for Fix 2
// (lifting reflect.PointerTo out of the per-cell closure). It asserts that
// the converter returned by DefaultConverterFunc still produces identical
// outputs for every code path it covers.
func TestDefaultConverterFunc_BehaviorPreserved(t *testing.T) {
	t.Run("matching pointer type returns dereferenced value", func(t *testing.T) {
		fn := sqlutil.DefaultConverterFunc(reflect.TypeOf(int64(0)))
		v := int64(42)
		got, err := fn(&v)
		require.NoError(t, err)
		require.Equal(t, int64(42), got, "DefaultConverterFunc should dereference matching *T inputs")
	})

	t.Run("non-matching type passes through unchanged", func(t *testing.T) {
		fn := sqlutil.DefaultConverterFunc(reflect.TypeOf(int64(0)))
		s := "not-an-int"
		got, err := fn(&s)
		require.NoError(t, err)
		require.Equal(t, &s, got, "DefaultConverterFunc should pass through non-matching inputs")
	})

	t.Run("string column", func(t *testing.T) {
		fn := sqlutil.DefaultConverterFunc(reflect.TypeOf(""))
		s := "hello"
		got, err := fn(&s)
		require.NoError(t, err)
		require.Equal(t, "hello", got)
	})

	t.Run("repeated calls are stable", func(t *testing.T) {
		fn := sqlutil.DefaultConverterFunc(reflect.TypeOf(int64(0)))
		for i := int64(0); i < 100; i++ {
			v := i
			got, err := fn(&v)
			require.NoError(t, err)
			require.Equal(t, i, got)
		}
	})
}

// TestDefaultConverterFunc_PointerToCachedAcrossCalls is a regression
// guard for Fix 2 (lifting reflect.PointerTo out of the per-call closure).
// reflect.PointerTo is internally cached by the Go runtime, so the
// optimization shows up as ns/op savings rather than allocs/op savings.
// This test asserts the cached form does not allocate any *more* than the
// per-call recompute form — anything else would indicate a regression.
// The actual ns/op proof lives in BenchmarkDefaultConverterFunc.
func TestDefaultConverterFunc_PointerToCachedAcrossCalls(t *testing.T) {
	const calls = 1000
	t64 := reflect.TypeOf(int64(0))
	v := int64(7)

	uncached := func(in interface{}) (interface{}, error) { //nolint:unparam // mirrors the shape of DefaultConverterFunc for apples-to-apples comparison
		if reflect.TypeOf(in) == reflect.PointerTo(t64) {
			return reflect.ValueOf(in).Elem().Interface(), nil
		}
		return in, nil
	}
	cached := sqlutil.DefaultConverterFunc(t64)

	uncachedAllocs := testing.AllocsPerRun(3, func() {
		for i := 0; i < calls; i++ {
			_, _ = uncached(&v)
		}
	})
	cachedAllocs := testing.AllocsPerRun(3, func() {
		for i := 0; i < calls; i++ {
			_, _ = cached(&v)
		}
	})

	t.Logf("DefaultConverterFunc allocs over %d calls: cached=%.0f uncached=%.0f", calls, cachedAllocs, uncachedAllocs)

	require.LessOrEqual(t, cachedAllocs, uncachedAllocs,
		"the cached version must never allocate more than the per-call recompute version")
}

// TestFrameFromRows_ScanBufferReuseProducesIdenticalOutput is a regression
// guard for Fix 3 (scan buffer reuse): the optimized loop must produce the
// exact same Frame as the pre-optimization "fresh buffer per row" pattern.
// We construct the baseline output by running a reference loop that calls
// NewScannableRow per row and the public Append helper (the previous behavior),
// then call FrameFromRows on a freshly built Rows and compare.
func TestFrameFromRows_ScanBufferReuseProducesIdenticalOutput(t *testing.T) {
	const cols, rows = 5, 50

	// Build the reference frame using the pre-optimization buffer-per-row pattern.
	refRows := makeWideRows(t, cols, rows) //nolint:rowserrcheck // refRows.Err() is checked after the Next() loop below
	t.Cleanup(func() { _ = refRows.Close() })
	refTypes, err := refRows.ColumnTypes()
	require.NoError(t, err)
	refNames, err := refRows.Columns()
	require.NoError(t, err)
	refScanRow, err := sqlutil.MakeScanRow(refTypes, refNames)
	require.NoError(t, err)
	refFrame := sqlutil.NewFrame(refNames, refScanRow.Converters...)
	for refRows.Next() {
		r := refScanRow.NewScannableRow() // fresh buffer per row (old behavior)
		require.NoError(t, refRows.Scan(r...))
		require.NoError(t, sqlutil.Append(refFrame, r, refScanRow.Converters...))
	}
	require.NoError(t, refRows.Err())

	// Now run the optimized FrameFromRows on equivalent data.
	optRows := makeWideRows(t, cols, rows) //nolint:rowserrcheck // sqlutil.FrameFromRows checks optRows.Err() internally
	t.Cleanup(func() { _ = optRows.Close() })
	optFrame, err := sqlutil.FrameFromRows(optRows, -1)
	require.NoError(t, err)

	require.Equal(t, refFrame, optFrame, "scan-buffer-reused FrameFromRows must produce the same Frame as the buffer-per-row baseline")
}

// TestFrameFromRows_ScanBufferReuseSavesAllocations proves Fix 3 by
// directly comparing the optimized FrameFromRows against a local baseline
// that reproduces the pre-optimization "fresh scan buffer per row" pattern.
// Both versions process the same rows; the alloc delta isolates the
// optimization. Each saved row should eliminate at least cols+1 allocations
// (one for the []interface{} buffer, one for each per-column reflect.New).
func TestFrameFromRows_ScanBufferReuseSavesAllocations(t *testing.T) {
	const cols, rows = 5, 200

	optimized := testing.AllocsPerRun(3, func() {
		r := makeWideRows(t, cols, rows) //nolint:rowserrcheck // sqlutil.FrameFromRowsWithCapacity checks r.Err() internally
		defer func() { _ = r.Close() }()
		_, err := sqlutil.FrameFromRowsWithCapacity(r, -1, rows)
		if err != nil {
			t.Fatal(err)
		}
	})

	baseline := testing.AllocsPerRun(3, func() {
		r := makeWideRows(t, cols, rows) //nolint:rowserrcheck // frameFromRowsNoBufferReuse checks r.Err() internally
		defer func() { _ = r.Close() }()
		err := frameFromRowsNoBufferReuse(r, rows)
		if err != nil {
			t.Fatal(err)
		}
	})

	saved := baseline - optimized
	expectedFloor := float64(rows*(cols+1)) * 0.5 // be generous: at least half the theoretical max

	t.Logf("FrameFromRows allocs: optimized=%.0f baseline=%.0f saved=%.0f (rows=%d cols=%d, theoretical max savings=%d)",
		optimized, baseline, saved, rows, cols, rows*(cols+1))

	require.Greater(t, saved, expectedFloor,
		"scan-buffer reuse must save at least %.0f allocations across %d rows × %d cols", expectedFloor, rows, cols)
}

// frameFromRowsNoBufferReuse reproduces the pre-Fix-3 pattern: a fresh
// scan buffer is allocated for every row. It exists only as a baseline
// for the alloc-saving benchmarks and tests — only the error return and
// the allocation profile of constructing the frame are observed, so the
// built frame itself is intentionally discarded.
func frameFromRowsNoBufferReuse(rows *sql.Rows, rowLimit int) error {
	types, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	names, err := rows.Columns()
	if err != nil {
		return err
	}
	scanRow, err := sqlutil.MakeScanRow(types, names)
	if err != nil {
		return err
	}
	frame := sqlutil.NewFrame(names, scanRow.Converters...)
	var i int
	for rows.Next() {
		r := scanRow.NewScannableRow() // fresh buffer per row (pre-optimization)
		if err := rows.Scan(r...); err != nil {
			return err
		}
		if err := sqlutil.Append(frame, r, scanRow.Converters...); err != nil {
			return err
		}
		i++
		if rowLimit > 0 && i >= rowLimit {
			break
		}
	}
	return rows.Err()
}

// TestFrameFromRowsWithCapacity_PresizesAllFields proves the end-to-end
// capacity API (Fix 1 plumbed through FrameFromRowsWithCapacity).
func TestFrameFromRowsWithCapacity_PresizesAllFields(t *testing.T) {
	const cols, rows, capHint = 4, 10, 1000

	r := makeWideRows(t, cols, rows) //nolint:rowserrcheck // sqlutil.FrameFromRowsWithCapacity checks r.Err() internally
	defer func() { _ = r.Close() }()

	frame, err := sqlutil.FrameFromRowsWithCapacity(r, -1, capHint)
	require.NoError(t, err)
	require.Equal(t, rows, frame.Rows())

	for _, f := range frame.Fields {
		require.GreaterOrEqual(t, f.Capacity(), capHint,
			"FrameFromRowsWithCapacity must reserve >= cap on every Field; field %q has cap %d", f.Name, f.Capacity())
	}
}

// TestFrameFromRowsWithCapacity_ZeroCapBehaviorIdenticalToFrameFromRows
// asserts that passing capacity=0 to the new entry point produces the
// identical Frame as the original FrameFromRows.
func TestFrameFromRowsWithCapacity_ZeroCapBehaviorIdenticalToFrameFromRows(t *testing.T) {
	const cols, rows = 3, 20

	withCapRows := makeWideRows(t, cols, rows) //nolint:rowserrcheck // sqlutil.FrameFromRowsWithCapacity checks withCapRows.Err() internally
	defer func() { _ = withCapRows.Close() }()
	withCapFrame, err := sqlutil.FrameFromRowsWithCapacity(withCapRows, -1, 0)
	require.NoError(t, err)

	plainRows := makeWideRows(t, cols, rows) //nolint:rowserrcheck // sqlutil.FrameFromRows checks plainRows.Err() internally
	defer func() { _ = plainRows.Close() }()
	plainFrame, err := sqlutil.FrameFromRows(plainRows, -1)
	require.NoError(t, err)

	require.Equal(t, plainFrame, withCapFrame)
}
