package data_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// TestField_Grow_ReservesCapacity proves Fix 1 (capacity primitive) at the
// Field level: Grow must extend capacity without changing length.
func TestField_Grow_ReservesCapacity(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func() *data.Field
	}{
		{"int64", func() *data.Field { return data.NewField("v", nil, []int64{}) }},
		{"nullable_int64", func() *data.Field { return data.NewField("v", nil, []*int64{}) }},
		{"string", func() *data.Field { return data.NewField("v", nil, []string{}) }},
		{"nullable_string", func() *data.Field { return data.NewField("v", nil, []*string{}) }},
		{"time", func() *data.Field { return data.NewField("v", nil, []time.Time{}) }},
		{"float64", func() *data.Field { return data.NewField("v", nil, []float64{}) }},
		{"bool", func() *data.Field { return data.NewField("v", nil, []bool{}) }},
		{"enum", func() *data.Field { return data.NewField("v", nil, []data.EnumItemIndex{}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.make()
			require.Equal(t, 0, f.Len(), "fresh field should have len 0")
			require.Equal(t, 0, f.Capacity(), "fresh field should have cap 0")

			f.Grow(1000)
			require.Equal(t, 0, f.Len(), "Grow must not change length")
			require.GreaterOrEqual(t, f.Capacity(), 1000, "Grow(1000) must reserve >= 1000 cap")
		})
	}
}

// TestField_Grow_NoOpWhenSufficient proves Grow does nothing if existing
// capacity already fits, even when called repeatedly.
func TestField_Grow_NoOpWhenSufficient(t *testing.T) {
	f := data.NewField("v", nil, []int64{})
	f.Grow(1000)
	capAfterFirst := f.Capacity()

	f.Grow(500)
	require.Equal(t, capAfterFirst, f.Capacity(), "Grow with smaller n should be a no-op")

	f.Grow(0)
	require.Equal(t, capAfterFirst, f.Capacity(), "Grow(0) should be a no-op")

	f.Grow(-1)
	require.Equal(t, capAfterFirst, f.Capacity(), "Grow(<0) should be a no-op")
}

// TestField_Grow_NoReallocOnAppend proves the practical payoff: after Grow(n),
// n appends do not cause the underlying slice to reallocate.
func TestField_Grow_NoReallocOnAppend(t *testing.T) {
	const n = 1000
	f := data.NewField("v", nil, []int64{})
	f.Grow(n)
	capBefore := f.Capacity()

	for i := int64(0); i < n; i++ {
		f.Append(i)
	}

	require.Equal(t, n, f.Len())
	require.Equal(t, capBefore, f.Capacity(), "appending n items after Grow(n) must not reallocate the backing slice")
}

// TestFrame_SetRowCapacity_GrowsAllFields proves Fix 1 at the Frame level:
// SetRowCapacity must reserve capacity on every Field.
func TestFrame_SetRowCapacity_GrowsAllFields(t *testing.T) {
	frame := data.NewFrame("f",
		data.NewField("a", nil, []int64{}),
		data.NewField("b", nil, []*string{}),
		data.NewField("c", nil, []float64{}),
	)
	for _, f := range frame.Fields {
		require.Equal(t, 0, f.Capacity())
	}

	frame.SetRowCapacity(500)

	for _, f := range frame.Fields {
		require.Equal(t, 0, f.Len(), "SetRowCapacity must not change any Field's length")
		require.GreaterOrEqual(t, f.Capacity(), 500, "SetRowCapacity must reserve >= n cap on every Field")
	}
}

// TestFrame_SetRowCapacity_NoReallocOnAppendRow proves the payoff at the Frame
// level: after SetRowCapacity(n), n calls to AppendRow must not reallocate any
// Field's backing slice.
func TestFrame_SetRowCapacity_NoReallocOnAppendRow(t *testing.T) {
	const n = 1000
	frame := data.NewFrame("f",
		data.NewField("a", nil, []int64{}),
		data.NewField("b", nil, []*string{}),
	)
	frame.SetRowCapacity(n)

	caps := make([]int, len(frame.Fields))
	for i, f := range frame.Fields {
		caps[i] = f.Capacity()
	}

	s := "x"
	for i := int64(0); i < n; i++ {
		frame.AppendRow(i, &s)
	}

	for i, f := range frame.Fields {
		require.Equal(t, caps[i], f.Capacity(), "Field %d (%s) must not have reallocated after SetRowCapacity(n) + n AppendRow", i, f.Name)
	}
}

// TestField_Grow_NilFieldsTolerated guards against the Frame.SetRowCapacity
// nil-check path (in case a caller has placeholder nil entries in Fields).
func TestFrame_SetRowCapacity_NilField(t *testing.T) {
	frame := &data.Frame{
		Fields: data.Fields{
			data.NewField("a", nil, []int64{}),
			nil,
			data.NewField("b", nil, []float64{}),
		},
	}
	require.NotPanics(t, func() { frame.SetRowCapacity(100) })
	require.GreaterOrEqual(t, frame.Fields[0].Capacity(), 100)
	require.GreaterOrEqual(t, frame.Fields[2].Capacity(), 100)
}
