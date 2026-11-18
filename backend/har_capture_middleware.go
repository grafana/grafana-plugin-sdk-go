package backend

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/harcapture"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/querycapture"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

const harCaptureRequestHeader = "X-Grafana-HAR-Capture"

// harResponseRefIDPrefix is the reserved prefix for the synthetic capture response. The full refID
// is namespaced by datasource UID (see harResponseRefID) so that when Grafana merges the responses
// of a multi-datasource query into a single flat map, the capture frames from different datasources
// do not collide. Grafana matches captured responses by this prefix.
const harResponseRefIDPrefix = "__har__"

// harResponseRefID returns the refID the capture frame is stored under, namespaced by datasource
// UID: e.g. "__har__P1234". Namespacing is required because Grafana merges the per-datasource plugin
// responses of a multi-datasource query into one flat map keyed by refID (query.go), with no
// collision handling -- a single fixed refID would make all but one datasource's captured traffic
// (and any stashed queryError) get silently overwritten. Falls back to the bare prefix when no
// datasource UID is available (should not happen for a datasource query, but keeps this total).
func harResponseRefID(pc PluginContext) string {
	if pc.DataSourceInstanceSettings != nil {
		if uid := pc.DataSourceInstanceSettings.UID; uid != "" {
			return harResponseRefIDPrefix + uid
		}
	}
	return harResponseRefIDPrefix
}

// newHARCaptureMiddleware returns a HandlerMiddleware that captures HTTP traffic
// when the QueryDataRequest carries an X-Grafana-HAR-Capture header.
//
// On completion the captured HAR JSON is appended to the QueryDataResponse as a special frame under
// a datasource-namespaced refID (prefix "__har__", see harResponseRefID) so Grafana can extract it
// across the GRPC boundary without cross-datasource collisions.
//
// Backward compatibility contract:
//   - The middleware only acts when the X-Grafana-HAR-Capture header is exactly "true".
//     For any other value, or when the header is absent (as sent by older Grafana versions),
//     it is a pure pass-through: no capture occurs and the response is returned unmodified.
//   - Plugins built against an SDK that predates this middleware do not have it in their handler
//     chain, so the header is simply an unused entry in QueryDataRequest.Headers and is ignored.
//   - When it does act, it only adds the reserved "__har__"-prefixed response and never modifies the
//     responses produced by the plugin, so enabling capture cannot alter existing query results.
func newHARCaptureMiddleware() HandlerMiddleware {
	return HandlerMiddlewareFunc(func(next Handler) Handler {
		return &harCaptureHandler{BaseHandler: NewBaseHandler(next)}
	})
}

type harCaptureHandler struct {
	BaseHandler
}

func (h *harCaptureHandler) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	// Raw map read, not req.GetHTTPHeader: Grafana core sets this header with a plain, unprefixed
	// map assignment (req.Headers[harCaptureHeader] = "true" in
	// clientmiddleware.HTTPCaptureMiddleware), the same producer contract FromAlertHeaderName
	// relies on (see alert_forwarder_middleware.go). GetHTTPHeader only surfaces
	// Authorization/X-Id-Token/Cookie or "http_"-prefixed keys, so it would never see this header.
	if req == nil || req.Headers[harCaptureRequestHeader] != "true" {
		return h.BaseHandler.QueryData(ctx, req)
	}

	ctx, buf := withQueryCapture(ctx)

	resp, err := h.BaseHandler.QueryData(ctx, req)

	harRefID := harResponseRefID(req.PluginContext)
	harFrame := queryCaptureFrame(buf, harRefID, err)
	if harFrame == nil {
		return resp, err
	}

	if resp == nil {
		resp = &QueryDataResponse{Responses: make(Responses)}
	}
	if resp.Responses == nil {
		resp.Responses = make(Responses)
	}
	// refIDs are user-controlled (panel query editor), so a plugin response could already occupy the
	// reserved refID. Don't clobber the panel's real data: skip attaching capture for this
	// request and leave the plugin's result (and error) untouched.
	if _, taken := resp.Responses[harRefID]; taken {
		warnHARRefIDCollision(harRefID)
		return resp, err
	}

	dr := DataResponse{Frames: data.Frames{harFrame}}
	if err != nil {
		// Also surface the failure on the synthetic response so the SDK's own middlewares
		// (loggerMiddleware, and metrics/tracing via RequestStatusFromQueryDataResponse) still
		// observe it. Otherwise the nil top-level error we return below -- needed so the response
		// survives the gRPC boundary -- would make those middlewares log/report the failed query as
		// successful. Grafana treats __har__ as synthetic and reads the error from queryError, not
		// this field.
		dr.Error = err
	}
	resp.Responses[harRefID] = dr

	// Return a nil error so the response (and the captured __har__ frame) survives the gRPC boundary.
	// Only in capture mode (header-gated); the failure is still visible via dr.Error / queryError.
	return resp, nil
}

func warnHARRefIDCollision(refID string) {
	Logger.Warn("HAR capture: reserved refID already used by the plugin response; skipping capture frame to avoid overwriting real data", "refID", refID)
}

func withQueryCapture(ctx context.Context) (context.Context, *harcapture.Buffer) {
	buf := harcapture.NewBuffer()

	captureMW := httpclient.NamedMiddlewareFunc("sdk-har-capture", func(_ httpclient.Options, next http.RoundTripper) http.RoundTripper {
		return httpclient.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			// Capture the request body before RoundTrip: the transport drains r.Body while
			// sending, so reading it afterwards would yield nothing for POST/PUT/PATCH calls.
			reqBody, reqTruncated := buf.DrainRequestBody(r)
			started := time.Now()
			resp, err := next.RoundTrip(r)
			elapsed := time.Since(started)
			// Capture on failure too: a failed dial/timeout/TLS error is exactly the traffic a
			// diagnostics tool needs to see. On error resp is nil, so the entry records the request
			// and the error (in Comment) with a zero-status response.
			buf.AddEntry(r, reqBody, reqTruncated, resp, err, started, elapsed)
			return resp, err
		})
	})
	ctx = httpclient.WithContextualMiddleware(ctx, captureMW)

	// Activate non-HTTP capture on the same context and into the same buffer. A datasource that speaks
	// its own protocol over a database/sql driver has no http.RoundTripper to wrap, so the middleware
	// above never sees it; the recorder is the seam such a plugin reports through (see package
	// querycapture). Sharing the buffer is what keeps SQL and HTTP evidence in one document instead of
	// racing for the reserved refID -- which matters for the plugins whose driver is itself HTTP-backed
	// (ClickHouse, BigQuery, Athena), since those produce both kinds at once. A plugin with no capture
	// point never reads the recorder back, so this is inert for every plugin that predates the seam.
	ctx = querycapture.WithRecorder(ctx, harcapture.NewRecorder(buf))

	return ctx, buf
}

func queryCaptureFrame(buf *harcapture.Buffer, refID string, err error) *data.Frame {
	if buf.Len() == 0 {
		return nil
	}
	payload, encodeErr := buf.ToHARString()
	if encodeErr != nil {
		return nil
	}
	custom := map[string]any{"har": payload}
	if err != nil {
		custom["queryError"] = err.Error()
	}
	f := data.NewFrame(refID)
	f.Meta = &data.FrameMeta{Custom: custom}
	return f
}

func (h *harCaptureHandler) QueryChunkedData(ctx context.Context, req *QueryChunkedDataRequest, w ChunkedDataWriter) error {
	if req == nil || req.Headers[harCaptureRequestHeader] != "true" {
		return h.BaseHandler.QueryChunkedData(ctx, req, w)
	}
	ctx, buf := withQueryCapture(ctx)
	refID := harResponseRefID(req.PluginContext)
	var collision atomic.Bool
	writer := observeChunkedWrites(w, func(id string, _ Status, _ error, _ ErrorSource) {
		if id == refID {
			collision.Store(true)
		}
	})
	err := h.BaseHandler.QueryChunkedData(ctx, req, writer)
	f := queryCaptureFrame(buf, refID, err)
	if f == nil {
		return err
	}
	if collision.Load() {
		warnHARRefIDCollision(refID)
		return err
	}
	// A raw receiver can reject typed writes. Encode the synthetic response just
	// like a plugin response before sending it through that receiver.
	if raw, ok := w.(rawChunkSink); ok {
		w = NewChunkedDataWriter(req.Format, raw.OnChunk)
	}
	if writeErr := w.WriteFrame(ctx, refID, refID, f); writeErr != nil {
		return errors.Join(err, writeErr)
	}
	if err != nil {
		// Keep the capture frame readable when the query itself fails.
		if writeErr := w.WriteError(ctx, refID, statusFromError(err), err); writeErr != nil {
			return errors.Join(err, writeErr)
		}
	}
	return nil
}
