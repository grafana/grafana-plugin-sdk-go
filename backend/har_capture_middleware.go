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

	ctx, buf := withSDKHARCapture(ctx)

	captureMW := httpclient.NamedMiddlewareFunc("sdk-har-capture", func(_ httpclient.Options, next http.RoundTripper) http.RoundTripper {
		return httpclient.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			// Capture the request body before RoundTrip: the transport drains r.Body while
			// sending, so reading it afterwards would yield nothing for POST/PUT/PATCH calls.
			reqBody := drainRequestBody(r)
			started := time.Now()
			resp, err := next.RoundTrip(r)
			elapsed := time.Since(started)
			if err == nil {
				buf.addEntry(r, reqBody, resp, started, elapsed)
			}
			return resp, err
		})
	})
	ctx = httpclient.WithContextualMiddleware(ctx, captureMW)

	resp, err := h.BaseHandler.QueryData(ctx, req)

	// Attach captured traffic even when QueryData returned an error: a failing datasource call is
	// exactly the traffic diagnostics needs to see, so we must not discard the buffer on error.
	// The plugin's own responses and error are preserved unchanged.
	if buf.len() > 0 {
		if harStr, serErr := buf.toHARString(); serErr == nil {
			if resp == nil {
				resp = &QueryDataResponse{Responses: make(Responses)}
			}
			if resp.Responses == nil {
				resp.Responses = make(Responses)
			}
			harFrame := data.NewFrame(harResponseKey)
			harFrame.Meta = &data.FrameMeta{
				Custom: map[string]interface{}{
					"har": harStr,
				},
			}
			resp.Responses[harResponseKey] = DataResponse{
				Frames: data.Frames{harFrame},
			}
		}
	}

	return resp, err
}
