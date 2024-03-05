package v0alpha1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

type QueryDataClient interface {
	QueryData(ctx context.Context, req QueryDataRequest, headers ...string) (int, *backend.QueryDataResponse, error)
}

type simpleHTTPClient struct {
	url     string
	client  *http.Client
	headers []string
}

func NewQueryDataClient(url string, client *http.Client, headers ...string) QueryDataClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &simpleHTTPClient{
		url:     url,
		client:  client,
		headers: headers,
	}
}

func (c *simpleHTTPClient) QueryData(ctx context.Context, query QueryDataRequest, headers ...string) (int, *backend.QueryDataResponse, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewBuffer(body))
	if err != nil {
		return http.StatusBadRequest, nil, err
	}
	headers = append(c.headers, headers...)
	if (len(headers) % 2) != 0 {
		return http.StatusBadRequest, nil, fmt.Errorf("headers must be in pairs of two")
	}
	for i := 0; i < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
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
