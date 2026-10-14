package sqlexprtest

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// recorder is a testing.TB that records failures instead of failing.
type recorder struct {
	testing.TB
	errors []string
}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func droppedFieldFrames() data.Frames {
	return data.Frames{typed(data.FrameTypeTimeSeriesWide, data.NewFrame("",
		data.NewField("time", nil, []time.Time{t0}),
		data.NewField("state", nil, []string{"ok"}),
		data.NewField("cpu", nil, []float64{1}),
	))}
}

func TestAssertFrames(t *testing.T) {
	t.Run("fails on warnings by default", func(t *testing.T) {
		rec := &recorder{TB: t}
		AssertFrames(rec, droppedFieldFrames())
		assert.Len(t, rec.errors, 1)
		assert.Contains(t, rec.errors[0], "dropped-field")
	})

	t.Run("ignores advice by default", func(t *testing.T) {
		rec := &recorder{TB: t}
		AssertFrames(rec, data.Frames{data.NewFrame("", data.NewField("v", nil, []float64{1}))})
		assert.Empty(t, rec.errors)
	})

	t.Run("FailOn", func(t *testing.T) {
		rec := &recorder{TB: t}
		AssertFrames(rec, droppedFieldFrames(), FailOn(SeverityError))
		assert.Empty(t, rec.errors)
	})

	t.Run("Ignore", func(t *testing.T) {
		rec := &recorder{TB: t}
		AssertFrames(rec, droppedFieldFrames(), Ignore(CodeDroppedField))
		assert.Empty(t, rec.errors)
	})
}

func TestAssertResponse(t *testing.T) {
	rec := &recorder{TB: t}
	resp := &backend.QueryDataResponse{Responses: backend.Responses{
		"A": {Frames: droppedFieldFrames()},
		"B": {Frames: data.Frames{tableFrame()}},
	}}

	reports := AssertResponse(rec, resp, []string{"A", "B", "C"})

	assert.Len(t, reports, 3)
	assert.Len(t, rec.errors, 2, "one failure for A, one for the missing C")
}
