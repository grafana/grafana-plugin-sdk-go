package live

import (
	"net/url"
	"path"
	"strings"
)

// ConnectionInfo contains everything a client needs to connect to the server.
type ConnectionInfo struct {
	// URL is the root Grafana HTTP URL.
	URL string `json:"host,omitempty"`

	// TODO: API KEY
}

// ToWebSocketURL converts the standard HTTP URL to the expected WS URL.
func (c *ConnectionInfo) ToWebSocketURL() (string, error) {
	u, err := url.Parse(c.URL)
	if err != nil {
		return "", err
	}

	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = path.Join(u.Path, "live", "ws")
	u.RawQuery = "format=protobuf"
	return u.String(), nil
}

// ChannelAddress is the channel ID split by parts.
type ChannelAddress struct {
	// Scope is "grafana", "ds", or "plugin".
	Scope string `json:"scope,omitempty"`

	// Namespace meaning depends on the scope.
	// * when Grafana, namespace is a "feature"
	// * when DS, namespace is the datasource ID
	// * when plugin, namespace is the plugin name
	Namespace string `json:"namespace,omitempty"`

	// Path is the channel path.
	// Within each namespace, the handler can process the path as needed.
	Path string `json:"path,omitempty"`
}

// ParseChannelAddress parses the parts from a channel ID:
//   ${scope} / ${namespace} / ${path}.
func ParseChannelAddress(id string) ChannelAddress {
	addr := ChannelAddress{}
	parts := strings.SplitN(id, "/", 3)
	length := len(parts)
	if length > 0 {
		addr.Scope = parts[0]
	}
	if length > 1 {
		addr.Namespace = parts[1]
	}
	if length > 2 {
		addr.Path = parts[2]
	}
	return addr
}

// IsValid checks if all parts of the address are valid.
func (ca *ChannelAddress) IsValid() bool {
	return ca.Scope != "" && ca.Namespace != "" && ca.Path != ""
}

// ToChannelID converts this to a single string.
func (ca *ChannelAddress) ToChannelID() string {
	return ca.Scope + "/" + ca.Namespace + "/" + ca.Path
}
