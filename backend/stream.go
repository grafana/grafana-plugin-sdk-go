package backend

import (
	"context"
)

// StreamHandler handles streams.
type StreamHandler interface {
	// Called when a user tries to connect to a plugin/datasource managed channel.
	CanSubscribeToStream(context.Context, *SubscribeToStreamRequest) (*SubscribeToStreamResponse, error)
	// RunStream will be initiated by Grafana to consume a stream from a plugin.
	// For streams with keepalive set this will only be called once the first client
	// successfully subscribed to a stream channel. And when there are no longer any
	// subscribers, the call will be terminated by Grafana.
	RunStream(context.Context, *RunStreamRequest, StreamPacketSender) error
}

// SubscribeToStreamRequest ...
type SubscribeToStreamRequest struct {
	PluginContext PluginContext
	Path          string
}

type SubscribeToStreamResponse struct {
	OK      bool
	Message string
}

type RunStreamRequest struct {
	PluginContext PluginContext
	Path          string
}

type StreamPacketType int32

type StreamPacket struct {
	Type    StreamPacketType
	Header  []byte
	Payload []byte
}

type StreamPacketSender interface {
	Send(*StreamPacket) error
}
