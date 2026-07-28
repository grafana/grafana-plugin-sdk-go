package sqlutil_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend/querycapture"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
)

func cells(t *testing.T, row []*string) []string {
	t.Helper()
	out := make([]string, len(row))
	for i, c := range row {
		if c == nil {
			out[i] = "<NULL>"
			continue
		}
		out[i] = *c
	}
	return out
}

func captureCtx(t *testing.T) (context.Context, *querycapture.ResultCapture) {
	t.Helper()
	capture := querycapture.NewResultCapture()
	return querycapture.WithResultCapture(context.Background(), capture), capture
}

func TestFrameFromRowsWithContext_capturesReturnedRows(t *testing.T) {
	rows := makeSingleResultSet( //nolint:rowserrcheck
		[]string{"host", "value"},
		[]interface{}{"host-a", int64(1)},
		[]interface{}{"host-b", int64(2)},
	)

	ctx, capture := captureCtx(t)
	frame, err := sqlutil.FrameFromRowsWithContext(ctx, rows, -1, 0)
	require.NoError(t, err)
	require.NotNil(t, frame)

	got := capture.Result()
	assert.Equal(t, []string{"host", "value"}, got.Columns)
	assert.False(t, got.Truncated)
	assert.Equal(t, 2, got.TotalRows)
	require.Len(t, got.Rows, 2)
	assert.Equal(t, []string{"host-a", "1"}, cells(t, got.Rows[0]))
	assert.Equal(t, []string{"host-b", "2"}, cells(t, got.Rows[1]))
}

func TestFrameFromRowsWithContext_capturesBeforeConverters(t *testing.T) {
	// The whole point of capturing at the scan is that converters are supplied by the plugin: a capture
	// taken after conversion could not tell a database that returned the wrong value from a converter
	// that mangled the right one. This converter rewrites every value, so the capture must disagree with
	// the frame.
	rows := makeSingleResultSet([]string{"host"}, []interface{}{"host-b"}) //nolint:rowserrcheck
	mangling := sqlutil.Converter{
		Name:             "mangle",
		InputScanType:    reflect.TypeOf(""),
		InputTypeMatcher: func(string) bool { return true },
		FrameConverter: sqlutil.FrameConverter{
			FieldType: data.FieldTypeString,
			ConverterFunc: func(interface{}) (interface{}, error) {
				return "host-a", nil
			},
		},
	}

	ctx, capture := captureCtx(t)
	frame, err := sqlutil.FrameFromRowsWithContext(ctx, rows, -1, 0, mangling)
	require.NoError(t, err)

	value, ok := frame.Fields[0].ConcreteAt(0)
	require.True(t, ok)
	assert.Equal(t, "host-a", value, "the converter rewrote the value, standing in for a plugin-side bug")

	captured := capture.Result().Rows
	require.Len(t, captured, 1)
	assert.Equal(t, []string{"host-b"}, cells(t, captured[0]),
		"the capture must hold what the database sent, which is what localizes the bug to the plugin")
}

func TestFrameFromRowsWithContext_nullIsDistinctFromEmpty(t *testing.T) {
	// "" and NULL are different answers to a wrong-data question, so they must not render alike.
	rows := makeSingleResultSet( //nolint:rowserrcheck
		[]string{"host"},
		[]interface{}{nil},
		[]interface{}{""},
	)

	ctx, capture := captureCtx(t)
	_, err := sqlutil.FrameFromRowsWithContext(ctx, rows, -1, 0)
	require.NoError(t, err)

	captured := capture.Result().Rows
	require.Len(t, captured, 2)
	assert.Nil(t, captured[0][0], "a NULL is captured as null")
	require.NotNil(t, captured[1][0])
	assert.Equal(t, "", *captured[1][0], "an empty string is captured as an empty string")
}

func TestFrameFromRowsWithContext_noCaptureOnContextIsUnchanged(t *testing.T) {
	// Capture is off by default and must not alter the frame in any way.
	row := func() []interface{} { return []interface{}{"host-a", int64(1)} }
	plainRows := makeSingleResultSet([]string{"host", "value"}, row()) //nolint:rowserrcheck
	plain, err := sqlutil.FrameFromRows(plainRows, -1)
	require.NoError(t, err)

	ctxRows := makeSingleResultSet([]string{"host", "value"}, row()) //nolint:rowserrcheck
	viaCtx, err := sqlutil.FrameFromRowsWithContext(context.Background(), ctxRows, -1, 0)
	require.NoError(t, err)

	assert.Equal(t, plain.Fields[0].Len(), viaCtx.Fields[0].Len())
	assert.Equal(t, plain.Fields[1].At(0), viaCtx.Fields[1].At(0))
}

func TestFrameFromRowsWithContext_boundedByMaxResultBytes(t *testing.T) {
	// A pathological result must not be able to grow the capture without limit, and the capture must
	// say so rather than presenting a prefix as the whole result.
	wide := strings.Repeat("x", 1<<20) // 1 MiB per row
	rowsData := make([][]interface{}, 0, 16)
	for i := 0; i < 16; i++ {
		rowsData = append(rowsData, []interface{}{wide})
	}
	rows := makeSingleResultSet([]string{"blob"}, rowsData...) //nolint:rowserrcheck

	ctx, capture := captureCtx(t)
	frame, err := sqlutil.FrameFromRowsWithContext(ctx, rows, -1, 0)
	require.NoError(t, err)
	assert.Equal(t, 16, frame.Fields[0].Len(),
		"every row still reaches the frame; capture never drops data from the query")

	got := capture.Result()
	assert.True(t, got.Truncated)
	assert.Equal(t, 16, got.TotalRows, "the total counts what the database returned, not what was kept")
	assert.Less(t, len(got.Rows), 16)
	assert.LessOrEqual(t, len(got.Rows)*(1<<20), querycapture.MaxResultBytes)
}
