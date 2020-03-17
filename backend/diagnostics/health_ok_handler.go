package diagnostics

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type okHandler struct {
}

func (h *okHandler) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return &backend.CheckHealthResult{
		Status: backend.HealthStatusOk,
	}, nil
}

// OKCheckHealthHandler check health handler that returns backend.HealthStatusOk status.
func OKCheckHealthHandler() backend.CheckHealthHandler {
	return &okHandler{}
}
