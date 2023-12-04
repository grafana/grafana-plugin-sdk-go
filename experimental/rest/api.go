package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Client interface
type Client interface {
	Fetch(ctx context.Context, uriPath string, uriQuery string) (*http.Response, error)
}

// Input represents a rest api input
type Input struct {
	Name  string
	Value string
	Type  string
}

// API takes calls from the host, using the client to retrieve data from rest apis.
type API struct {
	Client         Client
	Routes         map[string]string
	DefaultParams  map[string]string
	ErrorFormatter func(body io.ReadCloser) error
	Framer         func(name string, results []Data) (data.Frames, error)
	IsError        func(resp http.Response) bool
}

// Call rest api and convert to dataframes.
func (api *API) Call(ctx context.Context, kind string, inputs []Input) ([]*data.Frame, error) {
	path, params := api.GetPathParams(kind, inputs)

	resp, err := api.Client.Fetch(ctx, path, params)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("failed to close response body: %w", closeErr)
		}
	}()

	if api.hasError(*resp) {
		return nil, api.error(*resp)
	}

	results := []Data{}
	err = json.NewDecoder(resp.Body).Decode(&results)
	if err != nil {
		return nil, err
	}

	if api.Framer != nil {
		return api.Framer(kind, results)
	}
	framer := JSONFramer{data: results, name: kind}
	return framer.Frames()
}

// GetPathParams takes the inputs and returns an appropriate url string
func (api *API) GetPathParams(kind string, inputs []Input) (string, string) {
	uriPathTemplate := api.Routes[kind]
	uriQuery := make(url.Values)
	for k, v := range api.DefaultParams {
		uriQuery.Set(k, v)
	}

	uriPath := uriPathTemplate
	for _, input := range inputs {
		if input.Type == "query" {
			uriQuery.Set(input.Name, input.Value)
		} else {
			uriPath = strings.ReplaceAll(uriPath, input.Name, input.Value)
		}
	}

	return uriPath, uriQuery.Encode()
}

func (api *API) hasError(resp http.Response) bool {
	if api.IsError == nil {
		return isError(resp)
	}
	return api.IsError(resp)
}

func (api *API) error(resp http.Response) error {
	if api.ErrorFormatter != nil {
		return api.ErrorFormatter(resp.Body)
	}
	return errorFormatter(resp.Body)
}

func isError(resp http.Response) bool {
	return resp.StatusCode >= 400
}

func errorFormatter(body io.ReadCloser) error {
	d, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	return errors.New(string(d))
}
