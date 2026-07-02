// Package storedobjects provides a small HTTP client that a plugin backend
// can use to read and update its own stored objects — the typed objects the
// plugin declares in its schema artifact and Grafana persists and serves at
// /apis/<group>/<version>/namespaces/<namespace>/<plural> on its aggregated
// API server. The intended consumer is a background goroutine started in the
// app instance factory (and stopped in Dispose) that lists objects and writes
// status.
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

	"github.com/grafana/grafana-plugin-sdk-go/config"
)

// defaultVersion is the artifact targetApiVersion assumed when a caller does
// not specify one. It matches the only version the PoC schema pipeline emits.
const defaultVersion = "v0alpha1"

// maxErrorBodyBytes bounds how much of an error response body is echoed into
// the returned error, so a misbehaving server can't bloat logs.
const maxErrorBodyBytes = 512

// Metadata carries the common identifying fields of a stored object. Only
// the fields a plugin backend typically needs are surfaced; anything else in
// the wire metadata is dropped on decode.
type Metadata struct {
	// Name uniquely identifies the object within its namespace.
	Name string `json:"name"`
	// Namespace is the tenancy boundary the object lives in.
	Namespace string `json:"namespace,omitempty"`
	// ResourceVersion is the server-assigned opaque version, changed on
	// every write. It is informational for this client: UpdateStatus uses a
	// merge patch precisely so callers don't have to manage it.
	ResourceVersion string `json:"resourceVersion,omitempty"`
	// UID is the server-assigned unique identifier for the object's
	// lifetime, stable across updates but not across delete/recreate.
	UID string `json:"uid,omitempty"`
	// Generation is incremented by the server on spec changes, and is the
	// usual input for "have I already reconciled this?" checks.
	Generation int64 `json:"generation,omitempty"`
	// Labels are user-defined key/value pairs attached to the object.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations are user-defined key/value pairs attached to the object.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Object is the wire shape of a stored object. Spec and Status are kept raw
// because the SDK cannot know the plugin's schema types; use SpecInto and
// StatusInto to decode them into the plugin's own structs.
type Object struct {
	// APIVersion is "<group>/<version>", where the group is the plugin ID.
	APIVersion string `json:"apiVersion,omitempty"`
	// Kind is the declared stored object kind name.
	Kind string `json:"kind,omitempty"`
	// Metadata identifies the object.
	Metadata Metadata `json:"metadata"`
	// Spec is the desired state as written by the object's author.
	Spec json.RawMessage `json:"spec,omitempty"`
	// Status is the observed state as reported by the plugin backend.
	Status json.RawMessage `json:"status,omitempty"`
}

// SpecInto decodes the object's spec into v, which follows the usual
// json.Unmarshal rules (v must be a non-nil pointer).
func (o *Object) SpecInto(v any) error {
	if len(o.Spec) == 0 {
		return errors.New("object has no spec")
	}
	return json.Unmarshal(o.Spec, v)
}

// StatusInto decodes the object's status into v, which follows the usual
// json.Unmarshal rules (v must be a non-nil pointer).
func (o *Object) StatusInto(v any) error {
	if len(o.Status) == 0 {
		return errors.New("object has no status")
	}
	return json.Unmarshal(o.Status, v)
}

// List is a page of stored objects, decoded from the server's list envelope.
type List struct {
	// Items are the objects in the list.
	Items []Object `json:"items"`
}

// Client reads and updates a plugin's own stored objects over the Grafana
// HTTP API. It is safe for concurrent use.
type Client struct {
	baseURL    string
	token      string
	group      string
	version    string
	httpClient *http.Client
}

// ClientOpts configures a Client.
type ClientOpts struct {
	// AppURL is the root URL of the Grafana instance, e.g.
	// "http://localhost:3000". Required. A trailing slash is trimmed.
	AppURL string
	// Token is the service-account token sent as a bearer token on every
	// request. Required.
	Token string
	// Group is the API group of the plugin's stored objects, which is the
	// plugin ID. Required.
	Group string
	// Version is the API version of the plugin's stored objects, matching
	// the schema artifact's targetApiVersion. Defaults to "v0alpha1".
	Version string
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
	version := opts.Version
	if version == "" {
		version = defaultVersion
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:    strings.TrimRight(opts.AppURL, "/"),
		token:      opts.Token,
		group:      opts.Group,
		version:    version,
		httpClient: httpClient,
	}, nil
}

// NewClientFromContext creates a Client for the plugin's own stored objects
// from the Grafana config that Grafana attaches to every plugin request
// context. group is the plugin ID. The token comes from the plugin's
// provisioned service account, which Grafana only supplies when its
// externalServiceAccounts feature toggle is enabled; without it this returns
// an error. Version defaults to "v0alpha1".
func NewClientFromContext(ctx context.Context, group string) (*Client, error) {
	cfg := config.GrafanaConfigFromContext(ctx)
	appURL, err := cfg.AppURL()
	if err != nil {
		return nil, fmt.Errorf("storedobjects: %w", err)
	}
	token, err := cfg.PluginAppClientSecret()
	if err != nil {
		return nil, fmt.Errorf("storedobjects: %w", err)
	}
	return NewClient(ClientOpts{
		AppURL: appURL,
		Token:  token,
		Group:  group,
	})
}

// NamespaceForOrgID maps a Grafana org ID to the namespace its stored
// objects live in: "default" for org 1, "org-<id>" otherwise. This mirrors
// Grafana's on-prem namespace mapping only; Grafana Cloud stacks use a
// stack-based namespace, so a production version of this would come from the
// request context rather than a local computation.
func NamespaceForOrgID(orgID int64) string {
	if orgID == 1 {
		return "default"
	}
	return fmt.Sprintf("org-%d", orgID)
}

// List returns all stored objects of the given plural kind in a namespace.
func (c *Client) List(ctx context.Context, namespace, plural string) (*List, error) {
	list := &List{}
	if err := c.do(ctx, http.MethodGet, c.collectionPath(namespace, plural), "", nil, list); err != nil {
		return nil, err
	}
	return list, nil
}

// Get returns a single stored object by name.
func (c *Client) Get(ctx context.Context, namespace, plural, name string) (*Object, error) {
	obj := &Object{}
	if err := c.do(ctx, http.MethodGet, c.objectPath(namespace, plural, name), "", nil, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// UpdateStatus replaces the object's status with the given value and returns
// the updated object. status is marshaled to JSON and sent as a merge patch
// against the status subresource: a merge patch applies unconditionally, so
// a background reconciler doesn't have to re-read the object and retry on
// resourceVersion conflicts the way a full PUT would require.
func (c *Client) UpdateStatus(ctx context.Context, namespace, plural, name string, status any) (*Object, error) {
	raw, err := json.Marshal(status)
	if err != nil {
		return nil, fmt.Errorf("storedobjects: marshal status: %w", err)
	}
	body, err := json.Marshal(map[string]json.RawMessage{"status": raw})
	if err != nil {
		return nil, fmt.Errorf("storedobjects: marshal status patch: %w", err)
	}
	obj := &Object{}
	path := c.objectPath(namespace, plural, name) + "/status"
	if err := c.do(ctx, http.MethodPatch, path, "application/merge-patch+json", bytes.NewReader(body), obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// collectionPath builds the URL for a kind's collection in a namespace.
func (c *Client) collectionPath(namespace, plural string) string {
	return fmt.Sprintf("%s/apis/%s/%s/namespaces/%s/%s",
		c.baseURL,
		url.PathEscape(c.group),
		url.PathEscape(c.version),
		url.PathEscape(namespace),
		url.PathEscape(plural),
	)
}

// objectPath builds the URL for a single named object.
func (c *Client) objectPath(namespace, plural, name string) string {
	return c.collectionPath(namespace, plural) + "/" + url.PathEscape(name)
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
