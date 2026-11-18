package backend

import (
	"context"
	"encoding/json"
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
