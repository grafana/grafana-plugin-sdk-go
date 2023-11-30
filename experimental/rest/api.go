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
	ErrorFormatter func(string) string
	Framer         data.Framer
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

	if resp.StatusCode >= 400 {
		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		message := string(d)
		if api.ErrorFormatter != nil {
			message = api.ErrorFormatter(message)
		}
		return nil, errors.New(message)
	}

	results := []Data{}
	err = json.NewDecoder(resp.Body).Decode(&results)
	if err != nil {
		return nil, err
	}

	if api.Framer != nil {
		return api.Framer.Frames()
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
			uriPath = strings.Replace(uriPath, input.Name, input.Value, -1)
		}
	}

	return uriPath, uriQuery.Encode()
}
