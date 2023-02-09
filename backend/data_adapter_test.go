package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/stretchr/testify/require"
)

type fakeDataHandlerWithOAuth struct {
	cli *http.Client
	svr *httptest.Server
}

func newFakeDataHandlerWithOAuth() *fakeDataHandlerWithOAuth {
	settings := DataSourceInstanceSettings{}
	opts, err := settings.HTTPClientOptions()
	if err != nil {
		panic("http client options: " + err.Error())
	}
	cli, err := httpclient.New(opts)
	if err != nil {
		panic("httpclient new: " + err.Error())
	}

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(authHeader) == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get(xIDTokenHeader) == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	return &fakeDataHandlerWithOAuth{
		cli: cli,
		svr: svr,
	}
}

func (f *fakeDataHandlerWithOAuth) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", f.svr.URL, nil)
	if err != nil {
		return nil, err
	}

	res, err := f.cli.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return &QueryDataResponse{}, nil
}

func TestQueryData(t *testing.T) {
	adapter := newDataSDKAdapter(newFakeDataHandlerWithOAuth())
	ctx := context.Background()
	_, err := adapter.QueryData(ctx, &pluginv2.QueryDataRequest{
		Headers: map[string]string{
			authHeader:     "Bearer 123",
			xIDTokenHeader: "456",
		},
		PluginContext: &pluginv2.PluginContext{},
	})
	require.NoError(t, err)
}
