package live

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client allows communicating with Live API.
type Client struct {
	grafanaURL string
	apiKey     string
	httpClient *http.Client
}

// ClientOption modifies Client behaviour.
type ClientOption func(*Client)

// WithAPIKey allows setting API key to use.
func WithAPIKey(apiKey string) ClientOption {
	return func(h *Client) {
		h.apiKey = apiKey
	}
}

// WithHTTPClient allows setting custom http.Client to use.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(h *Client) {
		h.httpClient = client
	}
}

// DefaultHTTPClient will be used by default for HTTP requests.
var DefaultHTTPClient = &http.Client{Transport: &http.Transport{
	MaxIdleConnsPerHost: 100,
}, Timeout: time.Second}

// NewClient initializes Client.
func NewClient(grafanaURL string, opts ...ClientOption) (*Client, error) {
	_, err := url.Parse(grafanaURL)
	if err != nil {
		return nil, err
	}
	c := &Client{
		grafanaURL: strings.TrimSuffix(grafanaURL, "/"),
		httpClient: DefaultHTTPClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// PublishResult returned from Live server. This is empty at the moment,
// but can be extended with fields later.
type PublishResult struct{}

type publishRequest struct {
	Channel string          `json:"string"`
	Data    json.RawMessage `json:"data"`
}

const apiPublishPath = "/api/live/publish"

// Publish data to a Live channel.
func (c *Client) Publish(ctx context.Context, channel string, data json.RawMessage) (PublishResult, error) {
	cmd := publishRequest{
		Channel: channel,
		Data:    data,
	}
	jsonData, err := json.Marshal(cmd)
	if err != nil {
		return PublishResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.grafanaURL+apiPublishPath, bytes.NewReader(jsonData))
	if err != nil {
		return PublishResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return PublishResult{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return PublishResult{}, &StatusCodeError{Code: resp.StatusCode}
	}
	return PublishResult{}, nil
}
