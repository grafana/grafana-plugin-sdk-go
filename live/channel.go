package live

import (
	"errors"
	"regexp"
	"strings"
)

// ErrInvalidChannelID returned when channel ID does not have valid
// format. Valid channel IDs have 3 parts (scope, namespace, path)
// delimited by "/". Each part can only have alphanumeric symbols,
// underscore and dash.
var ErrInvalidChannelID = errors.New("invalid channel ID")

// Channel is the channel ID split by parts.
type Channel struct {
	// Scope is one of available channel scopes:
	// like ScopeGrafana, ScopePlugin, ScopeDatasource, ScopeStream.
	Scope string `json:"scope,omitempty"`

	// Namespace meaning depends on the scope.
	// * when ScopeGrafana, namespace is a "feature"
	// * when ScopePlugin, namespace is the plugin name
	// * when ScopeDatasource, namespace is the datasource uid
	// * when ScopeStream, namespace is the stream ID.
	Namespace string `json:"namespace,omitempty"`

	// Within each scope and namespace, the handler can process the path as needed.
	Path string `json:"path,omitempty"`
}

// ParseChannel parses the parts from a channel ID:
//   ${scope} / ${namespace} / ${path}.
// Channel parts allowed to have alphanumeric symbols, underscore and dash.
// For invalid channel IDs function returns ErrInvalidChannelID.
func ParseChannel(chID string) (Channel, error) {
	parts := strings.SplitN(chID, "/", 3)
	if len(parts) != 3 {
		return Channel{}, ErrInvalidChannelID
	}
	ch := Channel{
		Scope:     parts[0],
		Namespace: parts[1],
		Path:      parts[2],
	}
	if !ch.IsValid() {
		return ch, ErrInvalidChannelID
	}
	return ch, nil
}

// String converts Channel to a string representation (channel ID).
func (c Channel) String() string {
	ch := c.Scope
	if c.Namespace != "" {
		ch += "/" + c.Namespace
	}
	if c.Path != "" {
		ch += "/" + c.Path
	}
	return ch
}

var channelPattern = regexp.MustCompile("^[A-Za-z0-9_-]*$")

// IsValid checks if all parts of the Channel are valid.
func (c *Channel) IsValid() bool {
	allPartsExist := c.Scope != "" && c.Namespace != "" && c.Path != ""
	if !allPartsExist {
		return false
	}
	ok := channelPattern.MatchString(c.Scope)
	if !ok {
		return false
	}
	ok = channelPattern.MatchString(c.Namespace)
	if !ok {
		return false
	}
	return channelPattern.MatchString(c.Path)
}
