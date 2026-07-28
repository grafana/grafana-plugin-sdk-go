package querycapture

import (
	"context"
	"sync"
)

// ResultCapture collects the rows a datasource returned, bounded by MaxResultBytes.
//
// It is separate from Recorder because the two have different owners and different
// lifetimes. A Recorder is installed once per request by the host that activated
// capture; a ResultCapture is created per query by the capture point itself, which
// is the only code positioned between the driver and the conversion into frames.
// The capture point installs it on the context it passes down into row scanning,
// then reads it back and puts the rows on the Interaction it reports.
//
// Rows are recorded as rendered strings rather than driver values: a result set can
// hold any type a driver invents, and a capture is read by a human comparing it
// against returned frames, not re-executed. A nil cell means NULL.
//
// Safe for concurrent use, since scanning and reading back can happen on different
// goroutines, and cheap to leave installed: with no capture on the context the
// scanning path skips it entirely.
type ResultCapture struct {
	mu        sync.Mutex
	columns   []string
	rows      [][]*string
	bytes     int
	total     int
	truncated bool
}

// NewResultCapture returns an empty ResultCapture.
func NewResultCapture() *ResultCapture {
	return &ResultCapture{total: 0}
}

// SetColumns records the column names of the result set. Calling it more than once
// keeps the first set: a multi-result-set query reports the shape of the first,
// which is the one the row cells were rendered against.
func (c *ResultCapture) SetColumns(names []string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.columns != nil {
		return
	}
	c.columns = append([]string(nil), names...)
}

// AddRow records one scanned row. Cells must not be retained by the caller after
// the call -- AddRow takes ownership of the slice.
//
// Once MaxResultBytes is reached the row is counted but not kept, so the capture
// degrades into "here is the beginning, and here is how much there was" rather than
// silently presenting a prefix as the whole result.
func (c *ResultCapture) AddRow(cells []*string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.total++
	if c.truncated {
		return
	}
	size := 0
	for _, cell := range cells {
		if cell != nil {
			size += len(*cell)
		}
		// Per-cell allowance for the JSON punctuation this row will serialize into, so
		// the cap bounds the emitted document and not just the sum of the payloads.
		size += 4
	}
	if c.bytes+size > MaxResultBytes {
		c.truncated = true
		return
	}
	c.bytes += size
	c.rows = append(c.rows, cells)
}

// CapturedResult is what a ResultCapture collected.
type CapturedResult struct {
	// Columns are the result set's column names, empty when nothing was captured.
	Columns []string
	// Rows are the retained rows. A nil cell is a NULL.
	Rows [][]*string
	// Truncated reports that Rows is a prefix, because the result exceeded
	// MaxResultBytes.
	Truncated bool
	// TotalRows is how many rows the capture point saw, which exceeds len(Rows)
	// when Truncated is set. It is -1 when the capture point could not tell.
	TotalRows int
}

// Result returns what was captured. The rows are copied, so a caller reading them
// back while another goroutine is still scanning is safe. A nil ResultCapture
// reports a total of -1 ("not determined"), which is not the same as zero rows.
func (c *ResultCapture) Result() CapturedResult {
	if c == nil {
		return CapturedResult{TotalRows: -1}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	rows := make([][]*string, len(c.rows))
	for i, r := range c.rows {
		rows[i] = append([]*string(nil), r...)
	}
	return CapturedResult{
		Columns:   append([]string(nil), c.columns...),
		Rows:      rows,
		Truncated: c.truncated,
		TotalRows: c.total,
	}
}

type resultCaptureContextKey struct{}

// WithResultCapture returns a context that collects returned rows into c. Passing a
// nil ResultCapture returns ctx unchanged, so a capture point can wire it
// unconditionally and decide later whether capture is on.
func WithResultCapture(ctx context.Context, c *ResultCapture) context.Context {
	if c == nil {
		return ctx
	}
	return context.WithValue(ctx, resultCaptureContextKey{}, c)
}

// ResultCaptureFromContext returns the ResultCapture rows should be collected into,
// if any. When the second result is false the scanning path must behave exactly as
// it did before capture existed.
func ResultCaptureFromContext(ctx context.Context) (*ResultCapture, bool) {
	if ctx == nil {
		return nil, false
	}
	c, ok := ctx.Value(resultCaptureContextKey{}).(*ResultCapture)
	if !ok || c == nil {
		return nil, false
	}
	return c, true
}
