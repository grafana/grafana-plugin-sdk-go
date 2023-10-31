package httpclient

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/errorsource"
)

const name = "errorsource"

// New wraps the existing http client constructor and adds the error source middleware
func New(opts *httpclient.Options) (*http.Client, error) {
	opts.Middlewares = append(opts.Middlewares, errorsource.Middleware(name))
	c, err := httpclient.New(*opts)
	if err != nil {
		return nil, err
	}

	return c, nil
}
