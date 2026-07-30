// Package querycapture defines the capture seam for datasources that do not
// speak HTTP.
//
// Grafana's on-demand datasource diagnostics can show the traffic behind a
// failing panel because this SDK wraps the datasource's http.RoundTripper (see
// the HAR capture middleware in package backend). A datasource that speaks its
// own protocol over a database/sql driver, or a native wire protocol of its
// own, has no round tripper to wrap, so its bundle shows the frames the plugin
// returned with nothing to compare them against: "the database returned the
// wrong rows" is indistinguishable from "the plugin mangled correct rows".
//
// This package is the other half of that middleware. A capture point (today
// sqlds' DBQuery.Run; a MongoDB or Redis adapter tomorrow) looks for a Recorder
// in the query context and, if one is there, reports what it sent and what came
// back as an Interaction. The SDK's HAR capture middleware installs the
// Recorder and maps the Interactions onto the same HAR document it already
// returns, so protocol evidence and HTTP evidence land in one artifact.
//
// The division of labour is deliberate:
//
//   - Capture is off unless a Recorder is present in the context, and a nil or
//     absent Recorder is a pure pass-through. Nothing about query execution or
//     its results changes when capture is off, so a plugin gains capture from a
//     dependency bump alone.
//   - A capture point only produces Interactions. It does not decide how they
//     are encoded or returned to Grafana, and in particular it must not attach
//     them to the QueryDataResponse itself: the HAR middleware skips its own
//     capture when the reserved refID is already taken, so a plugin that emitted
//     its own capture frame would silently trade away its HTTP evidence.
//   - The vocabulary lives here, in the SDK, rather than in any one adapter, so
//     that every non-HTTP capture point describes itself the same way and the
//     host needs to understand only one shape.
//
// # Experimental
//
// This package is experimental and its API may change without the usual
// deprecation cycle. It exists to serve Grafana's on-demand datasource
// diagnostics, and the vocabulary is expected to move as capture points beyond
// SQL are written against it.
//
// # Sensitive data
//
// Interactions carry the statement that was executed, its bind arguments and a
// summary of what came back. All three can contain customer data and
// credentials. Capture points bound their size (see MaxStatementBytes and
// MaxArgsBytes) but do not redact them; redaction and retention are the
// Recorder's responsibility, and a Recorder must not be installed on a path
// where the result is not treated as sensitive.
//
// # Known limitations
//
// The size bounds below apply to a single Interaction. Nothing bounds their sum
// across the queries of one request, which run in parallel, so peak memory
// scales with the number of refIDs a panel sends. The Recorder bounds the
// document it produces, not the captures feeding it.
//
// Capture is off unless a request asked for it, and in Grafana the path that
// does the asking is feature-toggled, restricted to admins and on-prem only.
// That is what makes these acceptable for now rather than limits worth
// engineering around.
package querycapture

import (
	"context"
	"time"
)

// Interaction is one recorded exchange with a datasource. It is intentionally
// protocol-neutral: Kind identifies what was captured, so that a SQL query, a
// MongoDB command and an HTTP round trip can feed the same Recorder.
type Interaction struct {
	// Kind identifies the capture point, e.g. KindSQLQuery.
	Kind string
	// StartedAt is when the call to the datasource began.
	StartedAt time.Time
	// Duration is the wall-clock time spent on the call, including converting
	// the result into frames.
	Duration time.Duration

	// DatasourceUID, DatasourceType and DatasourceName identify the datasource
	// instance, so a consumer can correlate an Interaction with the panel query
	// that produced it in a multi-datasource capture.
	DatasourceUID  string
	DatasourceType string
	DatasourceName string
	// RefID is the query's refID, as set by the panel query editor. It is empty
	// for a capture point that runs outside a panel query, such as a schema or
	// completion lookup.
	RefID string

	// Statement is the query as it was handed to the driver: after macro
	// interpolation, which is the form the datasource actually saw. Truncated to
	// MaxStatementBytes; StatementTruncated reports whether that happened.
	Statement string
	// StatementTruncated reports whether Statement was cut to fit
	// MaxStatementBytes.
	StatementTruncated bool
	// Args are the bind arguments, rendered for display. Named arguments are
	// rendered as "name=value". Truncated to MaxArgsBytes in aggregate;
	// ArgsTruncated reports whether that happened.
	Args []string
	// ArgsTruncated reports whether Args was cut to fit MaxArgsBytes. When true,
	// the arguments present are a prefix of those actually sent.
	ArgsTruncated bool

	// FrameCount and RowCount summarise what the plugin returned. RowCount is -1
	// when it could not be determined: frames with inconsistent field lengths,
	// or a capture point that hands back an unconsumed result set.
	FrameCount int
	RowCount   int

	// Err is the error the call returned, or empty on success. A recorded
	// failure is the most valuable case for diagnostics, so capture points
	// record an Interaction on the error path too.
	Err string
}

// Kind values for the capture points that feed a Recorder. Each capture point
// emits its own; the set is declared here so consumers can switch on Kind
// without redefining the vocabulary.
const (
	// KindSQLQuery is a statement executed against a database/sql connection.
	KindSQLQuery = "sql.query"
)

// Size bounds a capture point applies before handing an Interaction to a
// Recorder. A single pathological query (a large IN clause, a bulk INSERT) must
// not be able to grow the capture without limit, and the whole
// QueryDataResponse the capture rides back on has to fit inside the SDK's gRPC
// message size.
//
// These bound one Interaction. A Recorder is still responsible for bounding the
// total across interactions.
const (
	// MaxStatementBytes caps Interaction.Statement.
	MaxStatementBytes = 64 * 1024
	// MaxArgsBytes caps the aggregate size of Interaction.Args.
	MaxArgsBytes = 16 * 1024
)

type recorderContextKey struct{}

// Recorder receives Interactions. Implementations must be safe for concurrent
// use: the queries of one QueryDataRequest run in parallel, one goroutine per
// refID, so a single Recorder is written to from several goroutines at once.
//
// Record must not block for long and must not panic. It runs inline on the
// query path, so a slow Recorder slows the user's query.
type Recorder interface {
	Record(Interaction)
}

// WithRecorder returns a context that activates capture for every capture point
// reached under it. Passing a nil Recorder returns ctx unchanged, so a caller
// can wire capture unconditionally and decide later whether to enable it. A nil
// ctx is likewise returned unchanged, so that -- as with RecorderFromContext --
// no part of this seam is the thing that panics.
func WithRecorder(ctx context.Context, r Recorder) context.Context {
	if ctx == nil || r == nil {
		return ctx
	}
	return context.WithValue(ctx, recorderContextKey{}, r)
}

// RecorderFromContext returns the Recorder activated for ctx, if any. The
// second result reports whether capture is on; when it is false the caller must
// behave exactly as it did before capture existed.
func RecorderFromContext(ctx context.Context) (Recorder, bool) {
	if ctx == nil {
		return nil, false
	}
	r, ok := ctx.Value(recorderContextKey{}).(Recorder)
	if !ok || r == nil {
		return nil, false
	}
	return r, true
}
