package querycapture

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cell(s string) *string { return &s }

func TestResultCapture_keepsRowsInOrder(t *testing.T) {
	c := NewResultCapture()
	c.SetColumns([]string{"host", "value"})
	c.AddRow([]*string{cell("host-a"), cell("1")})
	c.AddRow([]*string{cell("host-b"), nil})

	got := c.Result()
	assert.Equal(t, []string{"host", "value"}, got.Columns)
	assert.False(t, got.Truncated)
	assert.Equal(t, 2, got.TotalRows)
	require.Len(t, got.Rows, 2)
	assert.Equal(t, "host-a", *got.Rows[0][0])
	assert.Nil(t, got.Rows[1][1], "a NULL survives as a nil cell rather than becoming an empty string")
}

func TestResultCapture_dropsAllRowsPastTheCapButKeepsCounting(t *testing.T) {
	// Past the cap the capture must become "here is how much there was", never a prefix presented as
	// the whole result -- so every row is dropped, including the one already collected before the row
	// that crosses the budget, not just the one that overran it.
	c := NewResultCapture()
	c.AddRow([]*string{cell("host-a")})
	c.AddRow([]*string{cell(strings.Repeat("x", MaxResultBytes))})

	got := c.Result()
	assert.True(t, got.Truncated)
	assert.Equal(t, 2, got.TotalRows)
	assert.Empty(t, got.Rows, "a partial result is not evidence of the whole result")
}

func TestResultCapture_firstColumnsWin(t *testing.T) {
	// The cells were rendered against the first result set's shape, so a later set's columns must not
	// relabel them.
	c := NewResultCapture()
	c.SetColumns([]string{"host"})
	c.SetColumns([]string{"something", "else"})

	assert.Equal(t, []string{"host"}, c.Result().Columns)
}

func TestResultCapture_resultIsACopy(t *testing.T) {
	c := NewResultCapture()
	c.AddRow([]*string{cell("host-a")})

	got := c.Result()
	got.Rows[0][0] = cell("mutated")

	assert.Equal(t, "host-a", *c.Result().Rows[0][0])
}

func TestResultCapture_nilIsInert(t *testing.T) {
	// sqlds passes whatever it has; a nil capture means capture is off and must never panic on the
	// query path. A total of -1 says "not determined", which is not the same as zero rows.
	var c *ResultCapture
	assert.NotPanics(t, func() {
		c.SetColumns([]string{"host"})
		c.AddRow([]*string{cell("host-a")})
	})
	got := c.Result()
	assert.Nil(t, got.Columns)
	assert.Nil(t, got.Rows)
	assert.False(t, got.Truncated)
	assert.Equal(t, -1, got.TotalRows)
}

func TestResultCapture_concurrentUse(t *testing.T) {
	// Scanning and reading back can happen on different goroutines. Run with -race.
	c := NewResultCapture()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.SetColumns([]string{"host"})
			c.AddRow([]*string{cell("host-a")})
			_ = c.Result()
		}()
	}
	wg.Wait()

	got := c.Result()
	assert.Equal(t, 50, got.TotalRows)
	assert.Len(t, got.Rows, 50)
}

func TestResultCaptureFromContext(t *testing.T) {
	t.Run("absent by default", func(t *testing.T) {
		got, ok := ResultCaptureFromContext(t.Context())
		assert.False(t, ok)
		assert.Nil(t, got)
	})

	t.Run("nil capture leaves the context alone", func(t *testing.T) {
		ctx := t.Context()
		assert.Equal(t, ctx, WithResultCapture(ctx, nil))
	})

	t.Run("round-trips", func(t *testing.T) {
		c := NewResultCapture()
		got, ok := ResultCaptureFromContext(WithResultCapture(t.Context(), c))
		require.True(t, ok)
		assert.Same(t, c, got)
	})
}
