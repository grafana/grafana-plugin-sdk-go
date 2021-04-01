package backend

import (
	"context"
	"encoding/json"
)

// StreamHandler handles streams.
// This is EXPERIMENTAL and is a subject to change till Grafana 8.
type StreamHandler interface {
	// SubscribeStream called when a user tries to subscribe to a plugin/datasource
	// managed channel path.
	SubscribeStream(context.Context, *SubscribeStreamRequest) (*SubscribeStreamResponse, error)
	// PublishStream called when a user tries to publish to a plugin/datasource
	// managed channel path.
	PublishStream(context.Context, *PublishStreamRequest) (*PublishStreamResponse, error)
	// RunStream will be initiated by Grafana to consume a stream where keepalive
	// option set to true. In this case RunStream will only be called once for the
	// first client successfully subscribed to a channel path. When Grafana detects
	// that there are no longer any subscribers inside a channel, the call will be
	// terminated until next active subscriber appears. Call termination can happen
	// with a delay.
	RunStream(context.Context, *RunStreamRequest, StreamPacketSender) error
}

// SubscribeStreamRequest is EXPERIMENTAL and is a subject to change till Grafana 8.
type SubscribeStreamRequest struct {
	PluginContext PluginContext
	Path          string
}

// SubscribeStreamResponse is EXPERIMENTAL and is a subject to change till Grafana 8.
type SubscribeStreamResponse struct {
	OK           bool
	ErrorMessage string
	Schema       json.RawMessage
	Keepalive    bool
}

// PublishStreamRequest is EXPERIMENTAL and is a subject to change till Grafana 8.
type PublishStreamRequest struct {
	PluginContext PluginContext
	Path          string
	Data          json.RawMessage
}

// PublishStreamResponse is EXPERIMENTAL and is a subject to change till Grafana 8.
type PublishStreamResponse struct {
	OK           bool
	ErrorMessage string
	Fallthrough  bool
}

// RunStreamRequest is EXPERIMENTAL and is a subject to change till Grafana 8.
type RunStreamRequest struct {
	PluginContext PluginContext
	Path          string
}

// StreamPacketType is EXPERIMENTAL and is a subject to change till Grafana 8.
type StreamPacketType int32

// StreamPacket is EXPERIMENTAL and is a subject to change till Grafana 8.
type StreamPacket struct {
	Type    StreamPacketType
	Header  []byte
	Payload []byte
}

// StreamPacketSender is EXPERIMENTAL and is a subject to change till Grafana 8.
type StreamPacketSender interface {
	Send(*StreamPacket) error
}
