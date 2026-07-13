package backend_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/handlertest"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

func TestHARCaptureMiddleware_noHeader_passthrough(t *testing.T) {
	var called bool
	cdt := handlertest.NewHandlerMiddlewareTest(t, handlertest.WithMiddlewares(backend.NewHARCaptureMiddlewareForTest()))
	cdt.TestHandler.QueryDataFunc = func(_ context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		called = true
		return &backend.QueryDataResponse{Responses: backend.Responses{}}, nil
	}

	resp, err := cdt.MiddlewareHandler.QueryData(context.Background(), &backend.QueryDataRequest{})
	require.NoError(t, err)
	assert.True(t, called)
	_, hasHARFrame := resp.Responses["__har__"]
	assert.False(t, hasHARFrame)
}

func TestHARCaptureMiddleware_withHeader_appendsHARFrame(t *testing.T) {
	cdt := handlertest.NewHandlerMiddlewareTest(t, handlertest.WithMiddlewares(backend.NewHARCaptureMiddlewareForTest()))
	cdt.TestHandler.QueryDataFunc = func(ctx context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		makeHTTPCall(ctx, t, http.MethodGet, "http://ds.example.com", nil)
		return &backend.QueryDataResponse{Responses: backend.Responses{}}, nil
	}

	resp, err := cdt.MiddlewareHandler.QueryData(context.Background(), &backend.QueryDataRequest{
		Headers: map[string]string{"X-Grafana-HAR-Capture": "true"},
	})
	require.NoError(t, err)

	harResp, ok := resp.Responses["__har__"]
	require.True(t, ok, "expected __har__ frame in response")
	require.Len(t, harResp.Frames, 1)
	assert.Equal(t, "__har__", harResp.Frames[0].Name)

	custom, ok := harResp.Frames[0].Meta.Custom.(map[string]interface{})
	require.True(t, ok)
	harStr, ok := custom["har"].(string)
	require.True(t, ok)

	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(harStr), &doc))
	log := doc["log"].(map[string]interface{})
	assert.Equal(t, "1.2", log["version"])
	assert.Len(t, log["entries"].([]interface{}), 1)
}

func TestHARCaptureMiddleware_withHeader_noHTTPCalls_noFrame(t *testing.T) {
	cdt := handlertest.NewHandlerMiddlewareTest(t, handlertest.WithMiddlewares(backend.NewHARCaptureMiddlewareForTest()))
	cdt.TestHandler.QueryDataFunc = func(_ context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		return &backend.QueryDataResponse{Responses: backend.Responses{}}, nil
	}

	resp, err := cdt.MiddlewareHandler.QueryData(context.Background(), &backend.QueryDataRequest{
		Headers: map[string]string{"X-Grafana-HAR-Capture": "true"},
	})
	require.NoError(t, err)
	_, hasHARFrame := resp.Responses["__har__"]
	assert.False(t, hasHARFrame)
}

// TestHARCaptureMiddleware_oldPluginWithoutMiddleware_ignoresHeader simulates a plugin built
// against an SDK that predates the HAR capture middleware: the handler chain has no HAR middleware,
// so the X-Grafana-HAR-Capture header must be received but ignored, with the response unchanged.
func TestHARCaptureMiddleware_oldPluginWithoutMiddleware_ignoresHeader(t *testing.T) {
	// No WithMiddlewares: the bare handler stands in for an old plugin build.
	cdt := handlertest.NewHandlerMiddlewareTest(t)

	var seenReq *backend.QueryDataRequest
	cdt.TestHandler.QueryDataFunc = func(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		seenReq = req
		// An old plugin may still make outbound HTTP calls; none should be captured.
		makeHTTPCall(ctx, t, http.MethodGet, "http://ds.example.com", nil)
		return &backend.QueryDataResponse{Responses: backend.Responses{"A": backend.DataResponse{}}}, nil
	}

	resp, err := cdt.MiddlewareHandler.QueryData(context.Background(), &backend.QueryDataRequest{
		Headers: map[string]string{"X-Grafana-HAR-Capture": "true"},
	})
	require.NoError(t, err)

	// The header is passed through untouched (received, not stripped) ...
	require.NotNil(t, seenReq)
	assert.Equal(t, "true", seenReq.Headers["X-Grafana-HAR-Capture"])

	// ... but ignored: no capture frame is added and the plugin's responses are intact.
	_, hasHARFrame := resp.Responses["__har__"]
	assert.False(t, hasHARFrame)
	_, hasA := resp.Responses["A"]
	assert.True(t, hasA)
}

// TestHARCaptureMiddleware_nonTrueHeaderValues_noCapture asserts the new middleware only captures
// on an exact "true" value; any other value is a safe no-op even if the plugin makes HTTP calls.
func TestHARCaptureMiddleware_nonTrueHeaderValues_noCapture(t *testing.T) {
	for _, value := range []string{"", "false", "1", "TRUE", "yes"} {
		t.Run("value="+value, func(t *testing.T) {
			cdt := handlertest.NewHandlerMiddlewareTest(t, handlertest.WithMiddlewares(backend.NewHARCaptureMiddlewareForTest()))
			cdt.TestHandler.QueryDataFunc = func(ctx context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
				makeHTTPCall(ctx, t, http.MethodGet, "http://ds.example.com", nil)
				return &backend.QueryDataResponse{Responses: backend.Responses{}}, nil
			}

			resp, err := cdt.MiddlewareHandler.QueryData(context.Background(), &backend.QueryDataRequest{
				Headers: map[string]string{"X-Grafana-HAR-Capture": value},
			})
			require.NoError(t, err)
			_, hasHARFrame := resp.Responses["__har__"]
			assert.False(t, hasHARFrame, "value %q must not trigger capture", value)
		})
	}
}

// TestHARCaptureMiddleware_withHeader_preservesExistingResponses asserts that appending the HAR
// frame does not modify or drop the responses the plugin produced.
func TestHARCaptureMiddleware_withHeader_preservesExistingResponses(t *testing.T) {
	cdt := handlertest.NewHandlerMiddlewareTest(t, handlertest.WithMiddlewares(backend.NewHARCaptureMiddlewareForTest()))
	cdt.TestHandler.QueryDataFunc = func(ctx context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		makeHTTPCall(ctx, t, http.MethodGet, "http://ds.example.com", nil)
		return &backend.QueryDataResponse{Responses: backend.Responses{
			"A": {Error: nil},
			"B": {Error: context.DeadlineExceeded},
		}}, nil
	}

	resp, err := cdt.MiddlewareHandler.QueryData(context.Background(), &backend.QueryDataRequest{
		Headers: map[string]string{"X-Grafana-HAR-Capture": "true"},
	})
	require.NoError(t, err)

	_, hasHARFrame := resp.Responses["__har__"]
	require.True(t, hasHARFrame, "expected __har__ frame")

	respA, hasA := resp.Responses["A"]
	require.True(t, hasA)
	assert.NoError(t, respA.Error)

	respB, hasB := resp.Responses["B"]
	require.True(t, hasB)
	assert.ErrorIs(t, respB.Error, context.DeadlineExceeded)
}

// TestHARCaptureMiddleware_capturesRequestBody asserts the request body is captured for methods
// that carry one: capture must read it before RoundTrip, since the transport drains r.Body while
// sending (a GET-only test would not catch a regression here).
func TestHARCaptureMiddleware_capturesRequestBody(t *testing.T) {
	const reqBody = `{"query":"up"}`
	cdt := handlertest.NewHandlerMiddlewareTest(t, handlertest.WithMiddlewares(backend.NewHARCaptureMiddlewareForTest()))
	cdt.TestHandler.QueryDataFunc = func(ctx context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		makeHTTPCall(ctx, t, http.MethodPost, "http://ds.example.com/api/v1/query", bytes.NewBufferString(reqBody))
		return &backend.QueryDataResponse{Responses: backend.Responses{}}, nil
	}

	resp, err := cdt.MiddlewareHandler.QueryData(context.Background(), &backend.QueryDataRequest{
		Headers: map[string]string{"X-Grafana-HAR-Capture": "true"},
	})
	require.NoError(t, err)

	harResp, ok := resp.Responses["__har__"]
	require.True(t, ok, "expected __har__ frame in response")
	custom := harResp.Frames[0].Meta.Custom.(map[string]interface{})
	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(custom["har"].(string)), &doc))
	entries := doc["log"].(map[string]interface{})["entries"].([]interface{})
	require.Len(t, entries, 1)
	postData, ok := entries[0].(map[string]interface{})["request"].(map[string]interface{})["postData"].(map[string]interface{})
	require.True(t, ok, "expected postData in captured request")
	assert.Equal(t, reqBody, postData["text"])
}

// TestHARCaptureMiddleware_appendsHARFrameOnQueryError asserts captured traffic is still returned
// when QueryData fails -- a failing datasource call is exactly what diagnostics needs to capture.
func TestHARCaptureMiddleware_appendsHARFrameOnQueryError(t *testing.T) {
	wantErr := errors.New("datasource boom")
	cdt := handlertest.NewHandlerMiddlewareTest(t, handlertest.WithMiddlewares(backend.NewHARCaptureMiddlewareForTest()))
	cdt.TestHandler.QueryDataFunc = func(ctx context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		makeHTTPCall(ctx, t, http.MethodGet, "http://ds.example.com", nil)
		return nil, wantErr
	}

	resp, err := cdt.MiddlewareHandler.QueryData(context.Background(), &backend.QueryDataRequest{
		Headers: map[string]string{"X-Grafana-HAR-Capture": "true"},
	})

	require.ErrorIs(t, err, wantErr, "the plugin's error must be preserved")
	require.NotNil(t, resp, "captured traffic must be returned even on error")
	_, hasHARFrame := resp.Responses["__har__"]
	assert.True(t, hasHARFrame, "expected __har__ frame despite QueryData error")
}

// makeHTTPCall simulates a plugin making an outbound HTTP call using the contextual middleware
// chain. The fake transport reads and discards the request body (as a real transport would), so
// capture must read it before RoundTrip.
func makeHTTPCall(ctx context.Context, t *testing.T, method, url string, body io.Reader) {
	t.Helper()
	mws := httpclient.ContextualMiddlewareFromContext(ctx)
	var rt http.RoundTripper = httpclient.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.Body != nil {
			_, _ = io.ReadAll(r.Body)
			_ = r.Body.Close()
		}
		return &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Proto:      "HTTP/1.1",
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
		}, nil
	})
	for _, mw := range mws {
		rt = mw.CreateMiddleware(httpclient.Options{}, rt)
	}
	req, _ := http.NewRequestWithContext(ctx, method, url, body)
	resp, err := rt.RoundTrip(req)
	if err == nil && resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}
