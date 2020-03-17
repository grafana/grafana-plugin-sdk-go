package resource

import (
	"context"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type notFoundHandler struct {
}

func (h *notFoundHandler) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	return sender.Send(&backend.CallResourceResponse{
		Status: http.StatusNotFound,
	})
}

// NotFoundHandler call resource handler that returns HTTP 404 status code.
func NotFoundHandler() backend.CallResourceHandler {
	return &notFoundHandler{}
}
