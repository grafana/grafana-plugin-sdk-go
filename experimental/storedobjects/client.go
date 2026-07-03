// Package storedobjects lets a plugin backend work with its own stored
// objects — the typed objects the plugin declares in its schema artifact and
// Grafana persists on the plugin's behalf. The contract is List, Get,
// WriteStatus, and Watch on typed items; how Grafana stores objects and
// delivers changes is an implementation detail of the platform and may change
// without affecting this API.
//
// The intended consumer is a background goroutine started in the app instance
// factory (and stopped in Dispose) that lists objects, reacts to change
// events, and writes status.
//
// EXPERIMENTAL: this package is under active development and its API is
// subject to breaking changes without notice.
package storedobjects

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/config"
)

// defaultVersion is the artifact targetApiVersion assumed when a caller does
// not specify one. It matches the only version the PoC schema pipeline emits.
const defaultVersion = "v0alpha1"

// maxErrorBodyBytes bounds how much of an error response body is echoed into
// the returned error, so a misbehaving server can't bloat logs.
const maxErrorBodyBytes = 512

// PluralOf derives the URL plural form of a declared object type name, e.g.
// "Watchlist" becomes "watchlists". It is the single source of the derivation
// shared by this client and the schema artifact builder.
func PluralOf(name string) string {
	return strings.ToLower(name) + "s"
}

// namespaceForOrgID maps a Grafana org ID to the tenancy namespace its stored
// objects live in: "default" for org 1, "org-<id>" otherwise. This mirrors
// Grafana's on-prem namespace mapping only; Grafana Cloud stacks use a
// stack-based namespace, which is why the request context's namespace is
// always preferred when present.
func namespaceForOrgID(orgID int64) string {
	if orgID == 1 {
		return "default"
	}
	return fmt.Sprintf("org-%d", orgID)
}

// Item is a single stored object as seen by the plugin: its identifying name
// and labels plus the typed spec (desired state, written by the object's
// author) and status (observed state, written by the plugin backend).
type Item[S, T any] struct {
	// Name uniquely identifies the item within the client's org namespace.
	Name string
	// Labels are user-defined key/value pairs attached to the item.
	Labels map[string]string
	// Spec is the desired state.
	Spec S
	// Status is the observed state.
	Status T
}

// Client accesses a plugin's own stored objects. It is safe for concurrent
// use. Use NewCollection to get typed access to a declared object type.
type Client struct {
	baseURL      string
	token        string
	group        string
	version      string
	orgNamespace string
	httpClient   *http.Client
}

// ClientOpts configures a Client.
type ClientOpts struct {
	// AppURL is the root URL of the Grafana instance, e.g.
	// "http://localhost:3000". Required. A trailing slash is trimmed.
	AppURL string
	// Token is the service-account token sent as a bearer token on every
	// request. Required.
	Token string
	// Group identifies the plugin's stored objects, which is the plugin ID.
	// Required.
	Group string
	// Version is the schema version of the plugin's stored objects, matching
	// the schema artifact's targetApiVersion. Defaults to "v0alpha1".
	Version string
	// OrgNamespace is the tenancy boundary the client operates in. Required.
	OrgNamespace string
	// HTTPClient overrides the HTTP client used for requests. Defaults to
	// http.DefaultClient.
	HTTPClient *http.Client
}

// NewClient creates a Client from explicit options.
func NewClient(opts ClientOpts) (*Client, error) {
	if opts.AppURL == "" {
		return nil, errors.New("storedobjects: AppURL is required")
	}
	if opts.Token == "" {
		return nil, errors.New("storedobjects: Token is required")
	}
	if opts.Group == "" {
		return nil, errors.New("storedobjects: Group is required")
	}
	if opts.OrgNamespace == "" {
		return nil, errors.New("storedobjects: OrgNamespace is required")
	}
	version := opts.Version
	if version == "" {
		version = defaultVersion
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:      strings.TrimRight(opts.AppURL, "/"),
		token:        opts.Token,
		group:        opts.Group,
		version:      version,
		orgNamespace: opts.OrgNamespace,
		httpClient:   httpClient,
	}, nil
}

// NewClientFromContext creates a Client for the plugin's own stored objects
// from the request context Grafana attaches to every plugin request. The
// group is the plugin ID and the org namespace comes from the request's
// plugin context. The token comes from the plugin's provisioned service
// account, which Grafana only supplies when its externalServiceAccounts
// feature toggle is enabled; without it this returns an error. Version
// defaults to "v0alpha1".
func NewClientFromContext(ctx context.Context) (*Client, error) {
	pluginCtx := backend.PluginConfigFromContext(ctx)
	cfg := config.GrafanaConfigFromContext(ctx)
	appURL, err := cfg.AppURL()
	if err != nil {
		return nil, fmt.Errorf("storedobjects: %w", err)
	}
	token, err := cfg.PluginAppClientSecret()
	if err != nil {
		return nil, fmt.Errorf("storedobjects: %w", err)
	}
	orgNamespace := pluginCtx.Namespace
	if orgNamespace == "" {
		orgNamespace = namespaceForOrgID(pluginCtx.OrgID)
	}
	return NewClient(ClientOpts{
		AppURL:       appURL,
		Token:        token,
		Group:        pluginCtx.PluginID,
		OrgNamespace: orgNamespace,
	})
}

// objectEnvelope is the wire shape of a stored object. It stays unexported:
// the dev-facing surface is Item, and the envelope exists only to decode
// server responses and change events.
type objectEnvelope struct {
	Metadata struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels,omitempty"`
	} `json:"metadata"`
	Spec   json.RawMessage `json:"spec,omitempty"`
	Status json.RawMessage `json:"status,omitempty"`
}

// listEnvelope is the wire shape of a list response.
type listEnvelope struct {
	Items []objectEnvelope `json:"items"`
}

// itemFromEnvelope decodes a wire envelope into a typed Item.
func itemFromEnvelope[S, T any](env objectEnvelope) (Item[S, T], error) {
	item := Item[S, T]{
		Name:   env.Metadata.Name,
		Labels: env.Metadata.Labels,
	}
	if len(env.Spec) > 0 {
		if err := json.Unmarshal(env.Spec, &item.Spec); err != nil {
			return item, fmt.Errorf("storedobjects: decode spec of %q: %w", item.Name, err)
		}
	}
	if len(env.Status) > 0 {
		if err := json.Unmarshal(env.Status, &item.Status); err != nil {
			return item, fmt.Errorf("storedobjects: decode status of %q: %w", item.Name, err)
		}
	}
	return item, nil
}

// Collection is typed access to one declared object type. S is the spec type
// and T the status type declared in the plugin's schema.
type Collection[S, T any] struct {
	client *Client
	name   string
	plural string
}

// NewCollection returns typed access to the declared object type with the
// given name, e.g. "Watchlist".
func NewCollection[S, T any](c *Client, name string) *Collection[S, T] {
	return &Collection[S, T]{
		client: c,
		name:   name,
		plural: PluralOf(name),
	}
}

// List returns all items of the collection's object type in the client's org
// namespace.
func (c *Collection[S, T]) List(ctx context.Context) ([]Item[S, T], error) {
	env := &listEnvelope{}
	if err := c.client.do(ctx, http.MethodGet, c.collectionPath(), "", nil, env); err != nil {
		return nil, err
	}
	items := make([]Item[S, T], 0, len(env.Items))
	for _, e := range env.Items {
		item, err := itemFromEnvelope[S, T](e)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// Get returns a single item by name.
func (c *Collection[S, T]) Get(ctx context.Context, name string) (Item[S, T], error) {
	env := objectEnvelope{}
	if err := c.client.do(ctx, http.MethodGet, c.itemPath(name), "", nil, &env); err != nil {
		return Item[S, T]{}, err
	}
	return itemFromEnvelope[S, T](env)
}

// WriteStatus replaces the named item's status with the given value. The
// status is sent as a merge patch, which applies unconditionally: a
// background reconciler doesn't have to re-read the item and retry on write
// conflicts the way a full replace would require.
func (c *Collection[S, T]) WriteStatus(ctx context.Context, name string, status T) error {
	raw, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("storedobjects: marshal status: %w", err)
	}
	body, err := json.Marshal(map[string]json.RawMessage{"status": raw})
	if err != nil {
		return fmt.Errorf("storedobjects: marshal status patch: %w", err)
	}
	env := objectEnvelope{}
	path := c.itemPath(name) + "/status"
	return c.client.do(ctx, http.MethodPatch, path, "application/merge-patch+json", bytes.NewReader(body), &env)
}

// collectionPath builds the URL for the collection in the client's org
// namespace.
func (c *Collection[S, T]) collectionPath() string {
	return fmt.Sprintf("%s/apis/%s/%s/namespaces/%s/%s",
		c.client.baseURL,
		url.PathEscape(c.client.group),
		url.PathEscape(c.client.version),
		url.PathEscape(c.client.orgNamespace),
		url.PathEscape(c.plural),
	)
}

// itemPath builds the URL for a single named item.
func (c *Collection[S, T]) itemPath(name string) string {
	return c.collectionPath() + "/" + url.PathEscape(name)
}

// do performs an authenticated request and decodes a 2xx JSON response into
// out. Non-2xx responses become an error carrying the status code and a
// truncated copy of the body, since the server's message is usually the only
// clue to what went wrong.
func (c *Client) do(ctx context.Context, method, u, contentType string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return fmt.Errorf("storedobjects: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("storedobjects: %s %s: %w", method, u, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		return fmt.Errorf("storedobjects: %s %s: unexpected status %d: %s",
			method, u, resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("storedobjects: %s %s: decode response: %w", method, u, err)
	}
	return nil
}
