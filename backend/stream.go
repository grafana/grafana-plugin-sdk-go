package backend

import (
	"context"
	"encoding/json"
)

// StreamHandler handles streams.
// This is EXPERIMENTAL and is a subject to change till Grafana 8.
type StreamHandler interface {
	// SubscribeStream called when a user tries to subscribe to a plugin/datasource
	// managed channel path – thus plugin can check subscribe permissions and communicate
	// options with Grafana Core.
	SubscribeStream(context.Context, *SubscribeStreamRequest) (*SubscribeStreamResponse, error)
	// PublishStream called when a user tries to publish to a plugin/datasource
	// managed channel path. Here plugin can check publish permissions and
	// modify publication data if required.
	PublishStream(context.Context, *PublishStreamRequest) (*PublishStreamResponse, error)
	// RunStream will be initiated by Grafana to consume a stream where use_run_stream
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

// SubscribeStreamStatus is a status of subscription response.
type SubscribeStreamStatus int32

const (
	// SubscribeStreamOK means subscription is allowed.
	SubscribeStreamOK SubscribeStreamStatus = 0
	// SubscribeStreamNotFound means stream does not exist at all.
	SubscribeStreamNotFound = 1
	// SubscribeStreamPermissionDenied means that user is not allowed to subscribe.
	SubscribeStreamPermissionDenied = 2
)

// SubscribeStreamResponse is EXPERIMENTAL and is a subject to change till Grafana 8.
type SubscribeStreamResponse struct {
	Status       SubscribeStreamStatus
	Data         json.RawMessage
	UseRunStream bool
}

// PublishStreamRequest is EXPERIMENTAL and is a subject to change till Grafana 8.
type PublishStreamRequest struct {
	PluginContext PluginContext
	Path          string
	Data          json.RawMessage
}

// PublishStreamStatus is a status of publication response.
type PublishStreamStatus int32

const (
	// PublishStreamOK means publication is allowed.
	PublishStreamOK PublishStreamStatus = 0
	// PublishStreamNotFound means stream does not exist at all.
	PublishStreamNotFound = 1
	// PublishStreamPermissionDenied means that user is not allowed to publish.
	PublishStreamPermissionDenied = 2
)

// PublishStreamResponse is EXPERIMENTAL and is a subject to change till Grafana 8.
type PublishStreamResponse struct {
	Status PublishStreamStatus
	Data   json.RawMessage
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
	Data json.RawMessage
}

// StreamPacketSender is EXPERIMENTAL and is a subject to change till Grafana 8.
type StreamPacketSender interface {
	Send(*StreamPacket) error
}
