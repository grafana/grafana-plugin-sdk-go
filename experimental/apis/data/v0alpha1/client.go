package v0alpha1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

var (
	ErrTransport = errors.New("failed to execute request")
)

type QueryDataClient interface {
	QueryData(ctx context.Context, req QueryDataRequest) (*backend.QueryDataResponse, error)
}

type simpleHTTPClient struct {
	url     string
	client  *http.Client
	headers map[string]string
}

func ResponseFromCode(err error, code int, queries []DataQuery) *backend.QueryDataResponse {
	responses := make(backend.Responses)

	for _, query := range queries {
		responses[query.RefID] = backend.DataResponse{
			Status: backend.Status(code),
			Error:  err,
		}
	}

	return &backend.QueryDataResponse{
		Responses: responses,
	}
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

func (c *simpleHTTPClient) QueryData(ctx context.Context, query QueryDataRequest) (*backend.QueryDataResponse, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return ResponseFromCode(err, http.StatusBadRequest, query.Queries), nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewBuffer(body))
	if err != nil {
		return ResponseFromCode(err, http.StatusBadRequest, query.Queries), nil
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	qdr := &backend.QueryDataResponse{}
	rsp, err := c.client.Do(req)
	if err != nil {
		return qdr, fmt.Errorf("%w: %w", ErrTransport, err)
	}

	code := rsp.StatusCode
	defer rsp.Body.Close()

	if code/100 != 2 {
		apiErr := Status{}
		iter, err := jsoniter.Parse(jsoniter.ConfigCompatibleWithStandardLibrary, rsp.Body, 1024*10)
		if err == nil {
			err = iter.ReadVal(&apiErr)
			if err == nil {
				return qdr, apiErr.Error()
			}
		}
		return qdr, fmt.Errorf("failed to deserialize response: %w", err)
	}

	iter, e2 := jsoniter.Parse(jsoniter.ConfigCompatibleWithStandardLibrary, rsp.Body, 1024*10)
	if e2 == nil {
		err = iter.ReadVal(qdr)
	}
	return qdr, err
}
