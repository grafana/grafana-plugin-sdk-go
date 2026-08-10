package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/slo"
)

func NewDS(t *testing.T) (*DS, error) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client, clientErr := slo.NewClient()
	return &DS{
		client: client,
		url:    server.URL,
	}, clientErr
}

type DS struct {
	client *http.Client
	url    string
}

func (m DS) QueryData(ctx context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	err := callGet(ctx, m)
	if err != nil {
		return nil, err
	}
	return &backend.QueryDataResponse{}, nil
}

func (m DS) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	err := callGet(ctx, m)
	if err != nil {
		return nil, err
	}
	return &backend.CheckHealthResult{
		Status: backend.HealthStatusOk,
	}, nil
}

func (m DS) CallResource(ctx context.Context, _ *backend.CallResourceRequest, _ backend.CallResourceResponseSender) error {
	err := callGet(ctx, m)
	if err != nil {
		return err
	}
	return nil
}

func callGet(ctx context.Context, m DS) error {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, m.url, nil)
	if err != nil {
		return err
	}
	res, err := m.client.Do(r)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	return nil
}
