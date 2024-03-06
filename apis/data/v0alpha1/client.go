package v0alpha1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

type QueryDataClient interface {
	QueryData(ctx context.Context, req QueryDataRequest) (int, *backend.QueryDataResponse, error)
}

type simpleHTTPClient struct {
	url     string
	client  *http.Client
	headers map[string]string
}

func NewQueryDataClient(url string, client *http.Client, headers map[string]string) QueryDataClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &simpleHTTPClient{
		url:     url,
		client:  client,
		headers: headers,
	}
}

func (c *simpleHTTPClient) QueryData(ctx context.Context, query QueryDataRequest) (int, *backend.QueryDataResponse, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewBuffer(body))
	if err != nil {
		return http.StatusBadRequest, nil, err
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	rsp, err := c.client.Do(req)
	if err != nil {
		return rsp.StatusCode, nil, err
	}
	defer rsp.Body.Close()

	qdr := &backend.QueryDataResponse{}
	iter, err := jsoniter.Parse(jsoniter.ConfigCompatibleWithStandardLibrary, rsp.Body, 1024*10)
	if err == nil {
		err = iter.ReadVal(qdr)
	}
	return rsp.StatusCode, qdr, err
}
