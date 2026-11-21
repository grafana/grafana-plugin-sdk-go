package backend

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// TestChunkedDataWriterSendsSchemaOncePerFrame verifies that the JSON writer sends the schema in the first chunk of
// each (refID, frameID) pair, also when the joined pairs are the same string.
func TestChunkedDataWriterSendsSchemaOncePerFrame(t *testing.T) {
	var chunks []*pluginv2.QueryChunkedDataResponse
	w := NewChunkedDataWriter(DataFrameFormat_JSON, func(chunk *pluginv2.QueryChunkedDataResponse) error {
		chunks = append(chunks, chunk)
		return nil
	})

	ctx := context.Background()
	newFrame := func() *data.Frame { return data.NewFrame("frame", data.NewField("value", nil, []int64{1})) }
	require.NoError(t, w.WriteFrame(ctx, "a-b", "c", newFrame()))
	require.NoError(t, w.WriteFrame(ctx, "a", "b-c", newFrame()))
	require.NoError(t, w.WriteFrame(ctx, "a", "b-c", newFrame()))

	hasSchema := func(chunk *pluginv2.QueryChunkedDataResponse) bool {
		var keys map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(chunk.Frame, &keys))
		_, ok := keys["schema"]
		return ok
	}
	require.Len(t, chunks, 3)
	require.True(t, hasSchema(chunks[0]), "first chunk of (a-b, c)")
	require.True(t, hasSchema(chunks[1]), "first chunk of (a, b-c)")
	require.False(t, hasSchema(chunks[2]), "second chunk of (a, b-c)")
}

// TestChunkedDataWriterContinuedFrames verifies that a frame can be continued once, and that its frame ID takes no
// more rows after that.
func TestChunkedDataWriterContinuedFrames(t *testing.T) {
	ctx := context.Background()
	frame := func(continues *data.FrameContinuation, fields ...*data.Field) *data.Frame {
		return data.NewFrame("frame", fields...).SetMeta(&data.FrameMeta{Continues: continues})
	}
	a := func() *data.Field { return data.NewField("a", nil, []int64{1}) }
	b := func() *data.Field { return data.NewField("b", nil, []int64{2}) }
	continuesOne := func() *data.FrameContinuation { return &data.FrameContinuation{FrameID: "1", Fields: []int{0, -1}} }

	for name, format := range map[string]DataFrameFormat{"arrow": DataFrameFormat_ARROW, "json": DataFrameFormat_JSON} {
		t.Run(name, func(t *testing.T) {
			var chunks int
			w := NewChunkedDataWriter(format, func(*pluginv2.QueryChunkedDataResponse) error {
				chunks++
				return nil
			})

			require.NoError(t, w.WriteFrame(ctx, "A", "1", frame(nil, a())))
			require.NoError(t, w.WriteFrame(ctx, "A", "2", frame(continuesOne(), a(), b())))
			// The writer reads the meta only in the first call of a frame ID, so this call does not continue frame 1 again.
			require.NoError(t, w.WriteFrame(ctx, "A", "2", frame(continuesOne(), a(), b())))
			// Frame IDs are separate for each refID.
			require.NoError(t, w.WriteFrame(ctx, "B", "1", frame(nil, a())))

			require.EqualError(t, w.WriteFrame(ctx, "A", "1", frame(nil, a())),
				`frame "1" of refID "A" was continued by frame "2", so it takes no more rows`)
			require.EqualError(t, w.WriteFrame(ctx, "A", "3", frame(continuesOne(), a(), b())),
				`frame "3" of refID "A" continues frame "1", which frame "2" already continued`)
			require.EqualError(t, w.WriteFrame(ctx, "A", "4", frame(&data.FrameContinuation{FrameID: "9", Fields: []int{0}}, a())),
				`frame "4" of refID "A" continues frame "9", which was not written`)
			require.Equal(t, 4, chunks)
		})
	}
}

// TestChunkedDataWriterKeepsStateWhenWriteFails verifies that a failed write does not retire the earlier frame,
// and that a retry of the continuing frame sends its schema again.
func TestChunkedDataWriterKeepsStateWhenWriteFails(t *testing.T) {
	ctx := context.Background()
	a := func() *data.Field { return data.NewField("a", nil, []int64{1}) }
	b := func() *data.Field { return data.NewField("b", nil, []int64{2}) }
	continuing := func() *data.Frame {
		return data.NewFrame("frame", a(), b()).SetMeta(&data.FrameMeta{
			Continues: &data.FrameContinuation{FrameID: "1", Fields: []int{0, -1}},
		})
	}
	errRejected := errors.New("rejected")

	for name, format := range map[string]DataFrameFormat{"arrow": DataFrameFormat_ARROW, "json": DataFrameFormat_JSON} {
		t.Run(name, func(t *testing.T) {
			var chunks []*pluginv2.QueryChunkedDataResponse
			rejectNext := false
			w := NewChunkedDataWriter(format, func(chunk *pluginv2.QueryChunkedDataResponse) error {
				if rejectNext {
					rejectNext = false
					return errRejected
				}
				chunks = append(chunks, chunk)
				return nil
			})

			require.NoError(t, w.WriteFrame(ctx, "A", "1", data.NewFrame("frame", a())))
			rejectNext = true
			require.ErrorIs(t, w.WriteFrame(ctx, "A", "2", continuing()), errRejected)
			// Frame 1 is not continued yet, so it still takes rows.
			require.NoError(t, w.WriteFrame(ctx, "A", "1", data.NewFrame("frame", a())))
			require.NoError(t, w.WriteFrame(ctx, "A", "2", continuing()))

			require.Len(t, chunks, 3)
			if format == DataFrameFormat_JSON {
				var keys map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(chunks[2].Frame, &keys))
				require.Contains(t, keys, "schema", "the retried first chunk of frame 2 must have the schema")
			}
		})
	}
}
