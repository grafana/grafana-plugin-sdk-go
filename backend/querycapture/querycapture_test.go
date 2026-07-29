package querycapture

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubRecorder struct{ got []Interaction }

func (s *stubRecorder) Record(i Interaction) { s.got = append(s.got, i) }

func TestRecorderFromContext(t *testing.T) {
	t.Run("a plain context has no recorder", func(t *testing.T) {
		// Capture is off by default: every capture point must behave exactly as it did before capture
		// existed unless a host asked for it.
		got, ok := RecorderFromContext(context.Background())
		require.False(t, ok)
		require.Nil(t, got)
	})

	t.Run("a nil context has no recorder", func(t *testing.T) {
		// Capture points pass whatever context the query arrived with; a nil context must read as
		// "capture off" rather than panic on the query path.
		//nolint:staticcheck // deliberately exercising the nil-context case
		got, ok := RecorderFromContext(nil)
		require.False(t, ok)
		require.Nil(t, got)
	})

	t.Run("round-trips the installed recorder", func(t *testing.T) {
		rec := &stubRecorder{}
		got, ok := RecorderFromContext(WithRecorder(context.Background(), rec))
		require.True(t, ok)
		require.Same(t, rec, got)
	})
}

func TestWithRecorder_nilIsNotCaptureOn(t *testing.T) {
	// Wiring capture unconditionally is a common host pattern, so a nil Recorder must leave the context
	// untouched instead of installing a typed nil that later reads as "capture on".
	ctx := context.Background()
	got := WithRecorder(ctx, nil)
	require.Equal(t, ctx, got)

	_, ok := RecorderFromContext(got)
	require.False(t, ok)
}

func TestWithRecorder_innermostRecorderWins(t *testing.T) {
	// A host may install capture more than once on nested contexts (a dashboard-wide capture around a
	// per-panel one); the closest one to the capture point is the one that receives the interaction.
	outer, inner := &stubRecorder{}, &stubRecorder{}
	ctx := WithRecorder(WithRecorder(context.Background(), outer), inner)

	rec, ok := RecorderFromContext(ctx)
	require.True(t, ok)
	rec.Record(Interaction{Kind: KindSQLQuery})

	require.Len(t, inner.got, 1)
	require.Empty(t, outer.got)
}
