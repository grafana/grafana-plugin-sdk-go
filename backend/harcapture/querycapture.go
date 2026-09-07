package harcapture

import (
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/querycapture"
)

// queryMimeType labels the statement carried in a query entry's postData. HAR has no notion of a
// non-HTTP protocol, so the statement is modelled as the request body.
const queryMimeType = "application/sql"

// querySummaryMimeType labels the response content of a query entry. The content is a summary of what
// the plugin got back (see querySummary), not the driver's raw reply: rows travel back to Grafana as
// frames and are reported in the QueryData artifact, so duplicating them here would double the size of
// a bundle to say the same thing twice.
const querySummaryMimeType = "application/json"

// AddQueryInteraction records a non-HTTP datasource exchange (a SQL statement, a native-protocol
// command) as a HAR entry, so that it lands in the same document as the HTTP traffic captured by
// AddEntry.
//
// The interaction is mapped, not translated: HAR is an HTTP format and a SQL query is not an HTTP
// request, so the fields HTTP would fill are either empty or carry the query's own equivalent.
// Everything that has no HTTP counterpart at all goes in the "_query" extension field, which canonical
// HAR parsers ignore and a bundle analyzer can read without parsing prose.
//
// A failed query is recorded with a zero-status response and the error in the entry's comment, the
// same shape AddEntry uses for a request that never produced an HTTP response.
func (b *Buffer) AddQueryInteraction(i querycapture.Interaction) {
	b.appendEntry(buildQueryHAREntry(i, b.maxBody()))
}

// queryInfoVersion is the current schema version of the "_query" object, bumped when its shape
// changes in a way a consumer has to know about.
const queryInfoVersion = 1

// sdkHARQueryInfo is the "_query" extension object on a query entry. HAR reserves underscore-prefixed
// fields for extensions, so this travels in a standard HAR document that any viewer can still open.
type sdkHARQueryInfo struct {
	// Version is the schema version of this object (see queryInfoVersion).
	Version int `json:"version"`
	// Kind is the capture point that produced the entry, e.g. querycapture.KindSQLQuery.
	Kind string `json:"kind"`
	// DatasourceUID, DatasourceType and DatasourceName identify which datasource instance ran the
	// statement, so a multi-datasource capture can be attributed panel by panel.
	DatasourceUID  string `json:"datasourceUid,omitempty"`
	DatasourceType string `json:"datasourceType,omitempty"`
	DatasourceName string `json:"datasourceName,omitempty"`
	// RefID ties the entry to the panel query that caused it. Empty for a statement the plugin ran
	// outside a panel query (a schema or completion lookup).
	RefID string `json:"refId,omitempty"`
	// Args are the statement's bind arguments, in the order the driver received them. They live here
	// rather than in the entry's postData because HAR defines postData.params for a URL-encoded body and
	// states that params and text are mutually exclusive.
	Args []string `json:"args,omitempty"`
	// StatementTruncated and ArgsTruncated report that the statement or the arguments were cut to fit a
	// size bound -- the capture point's own, or the buffer's cumulative budget -- so a reader can tell a
	// long statement from a clipped one.
	StatementTruncated bool `json:"statementTruncated,omitempty"`
	ArgsTruncated      bool `json:"argsTruncated,omitempty"`
	// FrameCount and RowCount summarise what the plugin returned; RowCount is -1 when the capture
	// point could not determine it.
	FrameCount int `json:"frameCount"`
	RowCount   int `json:"rowCount"`
	// Error is the error the query returned, or empty on success. It repeats the entry comment in a
	// field an analyzer can read without string matching.
	Error string `json:"error,omitempty"`
}

// payloadBytes is the "_query" object's share of the buffer's retained-payload budget (see
// Buffer.appendEntry): the bind arguments, which a bulk statement can make as large as the statement
// itself, and the error. A nil receiver is an HTTP entry, which has no "_query". Measured as
// jsonEscapedLen for the same reason as sdkHAREntry.payloadBytes -- the budget bounds the serialized
// size, not the raw byte count.
func (q *sdkHARQueryInfo) payloadBytes() int64 {
	if q == nil {
		return 0
	}
	n := jsonEscapedLen(q.Error)
	for _, a := range q.Args {
		n += jsonEscapedLen(a)
	}
	return n
}

// dropPayload drops the bind arguments, flagging them truncated so their absence does not read as a
// statement that had none. The error stays: it is what makes an over-budget entry worth keeping at
// all, and it is small next to the arguments it accompanies.
func (q *sdkHARQueryInfo) dropPayload() {
	if q == nil || len(q.Args) == 0 {
		return
	}
	q.Args = nil
	q.ArgsTruncated = true
}

// querySummary is the response content of a query entry: what came back, counted rather than copied.
type querySummary struct {
	FrameCount int `json:"frameCount"`
	RowCount   int `json:"rowCount"`
}

func buildQueryHAREntry(i querycapture.Interaction, maxBodyBytes int64) sdkHAREntry {
	elapsedMs := durationMs(i.Duration)

	statement, encoding := encodeBody([]byte(i.Statement), maxBodyBytes)
	// Report -1 ("unavailable" in HAR) when only a prefix of the statement was kept, symmetric with
	// how an over-cap HTTP body is reported.
	statementSize := int64(len(i.Statement))
	if i.StatementTruncated {
		statementSize = -1
	}
	postData := &sdkHARPostData{
		MimeType: queryMimeType,
		Text:     statement,
		Encoding: encoding,
	}

	// Success is a 200 with a counted summary; a failure is a zero-status response, matching how
	// AddEntry records a request that never got one. Default bodySize -1 so an absent response is not
	// misread as an empty one.
	resp := sdkHARResponse{Status: 0, HeadersSize: -1, BodySize: -1, Headers: []sdkHARNameValue{}, Cookies: []sdkHARCookie{}}
	var comment string
	if i.Err == "" {
		summary, err := json.Marshal(querySummary{FrameCount: i.FrameCount, RowCount: i.RowCount})
		if err != nil {
			// Cannot happen for two ints, but a capture must never be the reason a query looks broken:
			// fall back to no content rather than dropping the entry.
			summary = nil
		}
		resp.Status = 200
		resp.StatusText = "OK"
		resp.BodySize = int64(len(summary))
		resp.Content = sdkHARContent{
			Size:     int64(len(summary)),
			MimeType: querySummaryMimeType,
			Text:     string(summary),
		}
	} else {
		comment = "query error: " + i.Err
	}

	return sdkHAREntry{
		StartedDateTime: i.StartedAt.UTC().Format(time.RFC3339Nano),
		Time:            elapsedMs,
		Request: sdkHARRequest{
			Method: queryMethod,
			URL:    queryURL(i),
			// No HTTP version, headers or cookies: this exchange never was an HTTP request, and
			// inventing plausible-looking ones would make the capture lie about what was on the wire.
			Headers:     []sdkHARNameValue{},
			QueryString: []sdkHARNameValue{},
			Cookies:     []sdkHARCookie{},
			PostData:    postData,
			BodySize:    statementSize,
			HeadersSize: -1,
		},
		Response: resp,
		Cache:    sdkHARCache{},
		Timings:  sdkHARTimings{Send: 0, Wait: elapsedMs, Receive: 0},
		Comment:  comment,
		Query: &sdkHARQueryInfo{
			Version:        queryInfoVersion,
			Kind:           i.Kind,
			DatasourceUID:  i.DatasourceUID,
			DatasourceType: i.DatasourceType,
			DatasourceName: i.DatasourceName,
			RefID:          i.RefID,
			// Copied because the buffer outlives the Record call and ToHARString marshals outside the
			// lock: a capture point that reuses its args slice would otherwise race the marshaller.
			Args:               slices.Clone(i.Args),
			StatementTruncated: i.StatementTruncated,
			ArgsTruncated:      i.ArgsTruncated,
			FrameCount:         i.FrameCount,
			RowCount:           i.RowCount,
			Error:              i.Err,
		},
	}
}

// queryMethod is the HAR request method of a query entry. HAR does not constrain the method string,
// and a statement is neither a GET nor a POST, so name the operation for what it is rather than
// dressing it up as an HTTP verb.
const queryMethod = "QUERY"

// queryURL builds the entry's URL. HAR requires one and there is no URL for a database call at this
// seam -- the capture point sees the statement, not the driver's connection target -- so it addresses
// the datasource that ran the statement instead: "sql://<type>/<uid>?refId=A". The scheme is taken
// from the capture kind so a MongoDB or Redis capture point reads as "mongo://" or "redis://" without
// this package having to know about it.
func queryURL(i querycapture.Interaction) string {
	scheme := "query"
	if kind, _, found := strings.Cut(i.Kind, "."); found && kind != "" {
		scheme = kind
	} else if i.Kind != "" {
		scheme = i.Kind
	}

	host := i.DatasourceType
	if host == "" {
		host = "datasource"
	}
	u := url.URL{Scheme: scheme, Host: host, Path: "/" + i.DatasourceUID}
	if i.RefID != "" {
		u.RawQuery = url.Values{"refId": []string{i.RefID}}.Encode()
	}
	return u.String()
}

// bufferRecorder adapts a Buffer to the querycapture.Recorder interface, so a capture point can write
// straight into the document the HAR middleware is going to return. Recording inline keeps the entries
// of one request in the order they happened, interleaved with the HTTP entries of the same request.
type bufferRecorder struct {
	buf *Buffer
}

// NewRecorder returns a querycapture.Recorder that appends each Interaction to buf. It is
// concurrency-safe, as Recorder requires, because Buffer is.
func NewRecorder(buf *Buffer) querycapture.Recorder {
	return bufferRecorder{buf: buf}
}

func (r bufferRecorder) Record(i querycapture.Interaction) {
	if r.buf == nil {
		return
	}
	r.buf.AddQueryInteraction(i)
}
