package backend

import (
	"context"
	"net/http"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

const harCaptureRequestHeader = "X-Grafana-HAR-Capture"
const harResponseKey = "__har__"

// newHARCaptureMiddleware returns a HandlerMiddleware that captures HTTP traffic
// when the QueryDataRequest carries an X-Grafana-HAR-Capture header.
//
// On completion the captured HAR JSON is appended to the QueryDataResponse as a
// special frame (refId "__har__") so Grafana can extract it across the GRPC boundary.
//
// Backward compatibility contract:
//   - The middleware only acts when the X-Grafana-HAR-Capture header is exactly "true".
//     For any other value, or when the header is absent (as sent by older Grafana versions),
//     it is a pure pass-through: no capture occurs and the response is returned unmodified.
//   - Plugins built against an SDK that predates this middleware do not have it in their handler
//     chain, so the header is simply an unused entry in QueryDataRequest.Headers and is ignored.
//   - When it does act, it only adds the reserved "__har__" response and never modifies the
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
	if req == nil || req.Headers[harCaptureRequestHeader] != "true" {
		return h.BaseHandler.QueryData(ctx, req)
	}

	buf := newSDKHARCaptureBuffer()

	captureMW := httpclient.NamedMiddlewareFunc("sdk-har-capture", func(_ httpclient.Options, next http.RoundTripper) http.RoundTripper {
		return httpclient.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			// Capture the request body before RoundTrip: the transport drains r.Body while
			// sending, so reading it afterwards would yield nothing for POST/PUT/PATCH calls.
			reqBody, reqTruncated := drainRequestBody(r)
			started := time.Now()
			resp, err := next.RoundTrip(r)
			elapsed := time.Since(started)
			// Capture on failure too: a failed dial/timeout/TLS error is exactly the traffic a
			// diagnostics tool needs to see. On error resp is nil, so the entry records the request
			// and the error (in Comment) with a zero-status response.
			buf.addEntry(r, reqBody, reqTruncated, resp, err, started, elapsed)
			return resp, err
		})
	})
	ctx = httpclient.WithContextualMiddleware(ctx, captureMW)

	resp, err := h.BaseHandler.QueryData(ctx, req)

	// Nothing captured: pass the plugin's result through untouched.
	if buf.len() == 0 {
		return resp, err
	}
	harStr, serErr := buf.toHARString()
	if serErr != nil {
		return resp, err
	}

	if resp == nil {
		resp = &QueryDataResponse{Responses: make(Responses)}
	}
	if resp.Responses == nil {
		resp.Responses = make(Responses)
	}
	custom := map[string]interface{}{"har": harStr}
	if err != nil {
		// Preserve the top-level error inside the frame. Returning a non-nil error here would make
		// the SDK's gRPC adapter (data_adapter.go) discard the whole response -- including this
		// __har__ frame -- so the captured traffic for a failed call would never reach Grafana. We
		// instead carry the error across in the frame and return nil below; Grafana reads it back.
		custom["queryError"] = err.Error()
	}
	harFrame := data.NewFrame(harResponseKey)
	harFrame.Meta = &data.FrameMeta{Custom: custom}
	resp.Responses[harResponseKey] = DataResponse{Frames: data.Frames{harFrame}}

	// Return a nil error so the response (and the captured __har__ frame) survives the gRPC boundary.
	// This only happens in capture mode (guarded by the X-Grafana-HAR-Capture header at the top), so
	// normal query error semantics are unaffected; the original error is preserved in queryError.
	return resp, nil
}
