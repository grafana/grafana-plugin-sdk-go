package tools

import (
	"context"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"

	"golang.org/x/net/context/ctxhttp"
)

/*
 * A rest client to make http calls to datasource rest apis
 * The client interface allows mocking from unit tests
 */

var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			Renegotiation: tls.RenegotiateFreelyAsClient,
		},
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).Dial,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
	},
	Timeout: time.Second * 30,
}

// Client interface
type Client interface {
	Fetch(ctx context.Context, uriPath, uriQuery string) ([]byte, error)
	Get(ctx context.Context, uriPath, uriQuery string) (*http.Response, error)
}

type restClient struct {
	url     string
	headers map[string]string
}

// NewRestClient ...
func NewRestClient(url string, headers map[string]string) Client {
	return &restClient{
		url,
		headers,
	}
}

// Fetch - perform an HTTP GET and return the body as []byte to prep for marshalling
func (c *restClient) Fetch(ctx context.Context, path string, params string) ([]byte, error) {
	resp, err := c.Get(ctx, path, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint
	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}
	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return responseData, err
}

// Get - perform an HTTP GET and return the http response
// This can be used directly from resource calls that don't need to marshal the data
func (c *restClient) Get(context context.Context, uriPath, uriQuery string) (*http.Response, error) {
	u, err := url.Parse(c.url)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, uriPath)
	u.RawQuery = uriQuery
	var req *http.Request
	req, err = http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	return ctxhttp.Do(context, httpClient, req)
}
