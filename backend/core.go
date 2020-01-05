package backend

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	plugin "github.com/hashicorp/go-plugin"
)

// PluginConfig holds configuration for the queried plugin.
type PluginConfig struct {
	ID       int64
	OrgID    int64
	Name     string
	Type     string
	URL      string
	JSONData json.RawMessage
}

// PluginConfigFromProto converts the generated protobuf PluginConfig to this
// package's PluginConfig.
func pluginConfigFromProto(pc *pluginv2.PluginConfig) PluginConfig {
	return PluginConfig{
		ID:       pc.Id,
		OrgID:    pc.OrgId,
		Name:     pc.Name,
		Type:     pc.Type,
		URL:      pc.Url,
		JSONData: json.RawMessage(pc.JsonData),
	}
}

// coreWrapper converts to and from protobuf types.
type coreWrapper struct {
	plugin.NetRPCUnsupportedPlugin

	handlers PluginHandlers
}

// PluginHandlers is the collection of handlers that corresponds to the
// grpc "service BackendPlugin".
type PluginHandlers interface {
	QueryDataHandler
	ResourceHandler
}
