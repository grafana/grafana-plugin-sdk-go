package httpclient

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/errorsource"
)

// New wraps the existing http client constructor and adds the error source middleware
func New(opts *httpclient.Options) (*http.Client, error) {
	id := uuid.New()
	opts.Middlewares = append(opts.Middlewares, errorsource.Middleware(id.String()))
	c, err := httpclient.New(*opts)
	if err != nil {
		return nil, err
	}

	return c, nil
}
