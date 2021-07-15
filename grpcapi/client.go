package grpcapi

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/server"

	"google.golang.org/grpc"
)

// Client allows communicating with Grafana GRPC API.
// This is EXPERIMENTAL and is a subject to change till Grafana 8.
type Client struct {
	address  string
	token    string
	insecure bool
	conn     *grpc.ClientConn
	client   server.GrafanaClient
}

// ClientOption modifies Client behaviour.
type ClientOption func(*Client)

// WithToken allows setting API token to use.
// By default plugin takes API token from environment.
func WithToken(token string) ClientOption {
	return func(h *Client) {
		h.token = token
	}
}

// WithAddress allows setting address of Grafana GRPC server to use.
// By default plugin takes API address from environment.
func WithAddress(address string) ClientOption {
	return func(h *Client) {
		h.address = address
	}
}

// WithInsecure allows setting insecure option when dialing to GRPC server.
func WithInsecure(insecure bool) ClientOption {
	return func(h *Client) {
		h.insecure = insecure
	}
}

func ClientForOrgID(ctx context.Context, orgID int64) (*Client, error) {
	client, err := NewClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetOrgToken(ctx, GetOrgTokenRequest{
		OrgID: orgID,
	})
	if err != nil {
		return nil, err
	}
	return NewClient(WithToken(resp.Token))
}

// NewClient initializes Client.
func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		address:  os.Getenv("GF_GRPC_API_ADDRESS"),
		token:    os.Getenv("GF_GRPC_API_TOKEN"),
		insecure: os.Getenv("GF_GRPC_API_INSECURE") != "",
	}
	for _, opt := range opts {
		opt(c)
	}
	var grpcOpts []grpc.DialOption
	if c.insecure {
		grpcOpts = append(grpcOpts, grpc.WithInsecure())
	}
	grpcOpts = append(grpcOpts, grpc.WithPerRPCCredentials(tokenAuth{
		token: c.token,
	}))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, c.address, grpcOpts...)
	if err != nil {
		return nil, fmt.Errorf("fail to dial: %w", err)
	}
	c.conn = conn
	client := server.NewGrafanaClient(conn)
	c.client = client
	return c, nil
}

// Close underlying GRPC connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// PublishStream allows publishing data to a Live channel.
func (c *Client) PublishStream(ctx context.Context, req PublishRequest) (*PublishResponse, error) {
	cmd := &server.PublishStreamRequest{
		Channel: req.Channel,
		Data:    req.Data,
	}
	_, err := c.client.PublishStream(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &PublishResponse{}, nil
}

// GetOrgToken allows getting token for specific organization.
func (c *Client) GetOrgToken(ctx context.Context, req GetOrgTokenRequest) (*GetOrgTokenResponse, error) {
	cmd := &server.GetOrgTokenRequest{
		OrgId: req.OrgID,
	}
	resp, err := c.client.GetOrgToken(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return &GetOrgTokenResponse{
		Token: resp.Token,
	}, nil
}
