package live

import (
	"strings"
)

// Channel is the channel split by parts.
type Channel struct {
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

// ParseChannel parses the parts from a channel ID:
//   ${scope} / ${namespace} / ${path}.
func ParseChannel(channelID string) Channel {
	ch := Channel{}
	parts := strings.SplitN(channelID, "/", 3)
	length := len(parts)
	if length > 0 {
		ch.Scope = parts[0]
	}
	if length > 1 {
		ch.Namespace = parts[1]
	}
	if length > 2 {
		ch.Path = parts[2]
	}
	return ch
}

// IsValid checks if all parts of the address are valid.
func (ch *Channel) IsValid() bool {
	return ch.Scope != "" && ch.Namespace != "" && ch.Path != ""
}

// String converts Channel to a single string for requests.
func (ch *Channel) String() string {
	return ch.Scope + "/" + ch.Namespace + "/" + ch.Path
}
