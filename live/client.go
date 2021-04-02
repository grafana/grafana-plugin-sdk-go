package live

import (
	"context"
	"encoding/json"
	"log"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/server"

	"google.golang.org/grpc"
)

// Client allows communicating with Live API.
type Client struct {
	grafanaURL string
	apiKey     string
	conn       *grpc.ClientConn
	client     server.GrafanaClient
}

// ClientOption modifies Client behaviour.
type ClientOption func(*Client)

// WithAPIKey allows setting API key to use.
func WithAPIKey(apiKey string) ClientOption {
	return func(h *Client) {
		h.apiKey = apiKey
	}
}

// NewClient initializes Client.
func NewClient(opts ...ClientOption) (*Client, error) {
	var grpcOpts []grpc.DialOption
	grpcOpts = append(grpcOpts, grpc.WithInsecure())
	conn, err := grpc.Dial("localhost:10000", grpcOpts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	client := server.NewGrafanaClient(conn)
	c := &Client{
		conn:   conn,
		client: client,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// PublishResult returned from Live server. This is empty at the moment,
// but can be extended with fields later.
type PublishResult struct{}

// Publish data to a Live channel.
func (c *Client) PublishStream(ctx context.Context, channel string, data json.RawMessage) (PublishResult, error) {
	cmd := &server.PublishStreamRequest{
		Channel: channel,
		Data:    data,
	}
	_, err := c.client.PublishStream(ctx, cmd)
	return PublishResult{}, err
}
