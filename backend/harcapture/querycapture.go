package harcapture

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/querycapture"
)

// queryMimeType labels the statement carried in a query entry's postData. HAR has no notion of a
// non-HTTP protocol, so the statement is modelled as the request body -- which is what it is: the
// bytes the plugin handed to the driver.
const queryMimeType = "application/sql"

// querySummaryMimeType labels the response content of a query entry. The content is a summary of what
// the plugin got back (see querySummary), not the driver's raw reply: rows travel back to Grafana as
// frames and are reported in the QueryData artifact, so duplicating them here would double the size of
// a bundle to say the same thing twice.
const querySummaryMimeType = "application/json"

// AddQueryInteraction records a non-HTTP datasource exchange (a SQL statement, a native-protocol
// command) as a HAR entry, so that it lands in the same document as the HTTP traffic captured by
// AddEntry. It is what makes the capture usable for a datasource with no HTTP hop to wrap.
//
// The interaction is mapped, not translated: HAR is an HTTP format and a SQL query is not an HTTP
// request, so the fields HTTP would fill are either honestly empty (no headers, no cookies, no HTTP
// version) or carry the query's own equivalent -- the statement as the request body, the bind
// arguments as postData params, the row/frame counts as the response content. Everything that has no
// HTTP counterpart at all (the capture kind, the datasource identity, the refID, truncation flags)
// goes in the "_query" extension field, which canonical HAR parsers ignore and a bundle analyzer can
// read without parsing prose.
//
// A failed query is recorded with a zero-status response and the error in the entry's comment, the
// same shape AddEntry uses for a request that never produced an HTTP response. A failure is the most
// valuable case for diagnostics, so it must be visible rather than absent.
func (b *Buffer) AddQueryInteraction(i querycapture.Interaction) {
	b.appendEntry(buildQueryHAREntry(i))
}

// sdkHARQueryInfo is the "_query" extension object on a query entry. HAR reserves underscore-prefixed
// fields for extensions, so this travels in a standard HAR document that any viewer can still open.
type sdkHARQueryInfo struct {
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
	// StatementTruncated and ArgsTruncated report that the capture point cut the statement or the
	// arguments to fit its size bounds, so a reader can tell a long statement from a clipped one.
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

// querySummary is the response content of a query entry: what came back, counted rather than copied.
type querySummary struct {
	FrameCount int `json:"frameCount"`
	RowCount   int `json:"rowCount"`
}

func buildQueryHAREntry(i querycapture.Interaction) sdkHAREntry {
	elapsedMs := float64(i.Duration.Milliseconds())

	statement, encoding := encodeBody([]byte(i.Statement))
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
		Params:   queryParams(i.Args),
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
		StartedDateTime: i.StartedAt.UTC().Format(time.RFC3339),
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
			Kind:               i.Kind,
			DatasourceUID:      i.DatasourceUID,
			DatasourceType:     i.DatasourceType,
			DatasourceName:     i.DatasourceName,
			RefID:              i.RefID,
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

// queryParams renders bind arguments as HAR postData params, which is where the spec puts a request
// body's parameters and therefore where a HAR viewer already shows them. Positional arguments are
// named "$1", "$2", ... in the order the driver received them; a capture point that knows an
// argument's name has already rendered it as "name=value" and that is kept as the param value, since
// re-splitting it here would corrupt any value containing "=".
func queryParams(args []string) []sdkHARNameValue {
	if len(args) == 0 {
		return nil
	}
	params := make([]sdkHARNameValue, 0, len(args))
	for idx, a := range args {
		params = append(params, sdkHARNameValue{Name: fmt.Sprintf("$%d", idx+1), Value: a})
	}
	return params
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
