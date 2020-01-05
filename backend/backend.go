package backend

import (
	plugin "github.com/hashicorp/go-plugin"
)

type CorePlugin interface {
	DiagnosticsHandler
	ResourceHandler
}

type BackendPlugin interface {
	CorePlugin
	QueryDataHandler
}

// backendWrapper converts to and from protobuf types.
type backendWrapper struct {
	plugin.NetRPCUnsupportedPlugin
	handlers BackendPlugin
}
