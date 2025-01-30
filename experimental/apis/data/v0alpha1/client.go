package v0alpha1

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type QueryDataClient interface {
	QueryData(ctx context.Context, req QueryDataRequest) (*backend.QueryDataResponse, error)
}

func NewErrorQDR(req QueryDataRequest, err error) *backend.QueryDataResponse {
	qdr := backend.NewQueryDataResponse()
	for _, q := range req.Queries {
		qdr.Responses[q.RefID] = backend.DataResponse{
			Status: backend.StatusBadRequest,
			Error:  err,
		}
	}
	return qdr
}
