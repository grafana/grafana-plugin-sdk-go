package backend

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestMetricsMiddlewareRequestStatus(t *testing.T) {
	newHandler := func(t *testing.T, reg *prometheus.Registry, handlers *Handlers) *MiddlewareHandler {
		t.Helper()
		h, err := HandlerFromMiddlewares(handlers, NewMetricsMiddleware(reg, "test", false))
		require.NoError(t, err)
		return h
	}

	requireCount := func(t *testing.T, reg *prometheus.Registry, endpoint, status string) {
		t.Helper()
		families, err := reg.Gather()
		require.NoError(t, err)

		got := map[string]float64{}
		for _, mf := range families {
			if mf.GetName() != "test_plugin_request_total" {
				continue
			}
			for _, m := range mf.GetMetric() {
				labels := map[string]string{}
				for _, l := range m.GetLabel() {
					labels[l.GetName()] = l.GetValue()
				}
				got[labels["endpoint"]+"/"+labels["status"]] += m.GetCounter().GetValue()
			}
		}
		require.Equal(t, map[string]float64{endpoint + "/" + status: 1}, got)
	}

	t.Run("CheckHealth", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			result   *CheckHealthResult
			err      error
			expected string
		}{
			{name: "ok result", result: &CheckHealthResult{Status: HealthStatusOk}, expected: "ok"},
			{name: "error result", result: &CheckHealthResult{Status: HealthStatusError, Message: "auth failed"}, expected: "error"},
			{name: "unknown result", result: &CheckHealthResult{Status: HealthStatusUnknown}, expected: "ok"},
			{name: "returned error", err: errors.New("boom"), expected: "error"},
			{name: "returned cancellation", err: context.Canceled, expected: "cancelled"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				reg := prometheus.NewRegistry()
				h := newHandler(t, reg, &Handlers{
					CheckHealthHandler: CheckHealthHandlerFunc(func(_ context.Context, _ *CheckHealthRequest) (*CheckHealthResult, error) {
						return tc.result, tc.err
					}),
				})

				_, _ = h.CheckHealth(context.Background(), &CheckHealthRequest{})
				requireCount(t, reg, "checkHealth", tc.expected)
			})
		}
	})

	t.Run("CallResource", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			status   int
			noSend   bool
			err      error
			expected string
		}{
			{name: "success response", status: http.StatusOK, expected: "ok"},
			{name: "client error response", status: http.StatusBadRequest, expected: "error"},
			{name: "server error response", status: http.StatusInternalServerError, expected: "error"},
			{name: "returned error takes precedence over response", status: http.StatusOK, err: context.Canceled, expected: "cancelled"},
			{name: "no response sent", noSend: true, expected: "ok"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				reg := prometheus.NewRegistry()
				h := newHandler(t, reg, &Handlers{
					CallResourceHandler: CallResourceHandlerFunc(func(_ context.Context, _ *CallResourceRequest, sender CallResourceResponseSender) error {
						if !tc.noSend {
							require.NoError(t, sender.Send(&CallResourceResponse{Status: tc.status}))
						}
						return tc.err
					}),
				})

				_ = h.CallResource(context.Background(), &CallResourceRequest{}, CallResourceResponseSenderFunc(func(_ *CallResourceResponse) error {
					return nil
				}))
				requireCount(t, reg, "callResource", tc.expected)
			})
		}
	})
}
