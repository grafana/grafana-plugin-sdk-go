package backend

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// StreamHandler handles streams.
// This is EXPERIMENTAL and is a subject to change till Grafana 8.
type StreamHandler interface {
	// SubscribeStream called when a user tries to subscribe to a plugin/datasource
	// managed channel path – thus plugin can check subscribe permissions and communicate
	// options with Grafana Core. As soon as first subscriber joins channel RunStream
	// will be called.
	SubscribeStream(context.Context, *SubscribeStreamRequest) (*SubscribeStreamResponse, error)
	// PublishStream called when a user tries to publish to a plugin/datasource
	// managed channel path. Here plugin can check publish permissions and
	// modify publication data if required.
	PublishStream(context.Context, *PublishStreamRequest) (*PublishStreamResponse, error)
	// RunStream will be initiated by Grafana to consume a stream. RunStream will be
	// called once for the first client successfully subscribed to a channel path.
	// When Grafana detects that there are no longer any subscribers inside a channel,
	// the call will be terminated until next active subscriber appears. Call termination
	// can happen with a delay.
	RunStream(context.Context, *RunStreamRequest, StreamSender) error
}

// SubscribeStreamRequest is EXPERIMENTAL and is a subject to change till Grafana 8.
type SubscribeStreamRequest struct {
	PluginContext PluginContext
	Path          string
}

// SubscribeStreamStatus is a status of subscription response.
type SubscribeStreamStatus int32

const (
	// SubscribeStreamStatusOK means subscription is allowed.
	SubscribeStreamStatusOK SubscribeStreamStatus = 0
	// SubscribeStreamStatusNotFound means stream does not exist at all.
	SubscribeStreamStatusNotFound SubscribeStreamStatus = 1
	// SubscribeStreamStatusPermissionDenied means that user is not allowed to subscribe.
	SubscribeStreamStatusPermissionDenied SubscribeStreamStatus = 2
)

// SubscribeStreamResponse is EXPERIMENTAL and is a subject to change till Grafana 8.
type SubscribeStreamResponse struct {
	Status      SubscribeStreamStatus
	InitialData *InitialData
}

type InitialData struct {
	data []byte
}

func (d *InitialData) Data() []byte {
	return d.data
}

func FrameInitialData(frame *data.Frame) (*InitialData, error) {
	frameJSON, err := json.Marshal(frame)
	if err != nil {
		return nil, err
	}
	return &InitialData{
		data: frameJSON,
	}, nil
}

func FrameSchemaInitialData(frame *data.Frame) (*InitialData, error) {
	jsonData, err := data.FrameToJSON(frame, true, false)
	if err != nil {
		return nil, err
	}
	return &InitialData{
		data: jsonData,
	}, nil
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
	// PublishStreamStatusOK means publication is allowed.
	PublishStreamStatusOK PublishStreamStatus = 0
	// PublishStreamStatusNotFound means stream does not exist at all.
	PublishStreamStatusNotFound PublishStreamStatus = 1
	// PublishStreamStatusPermissionDenied means that user is not allowed to publish.
	PublishStreamStatusPermissionDenied PublishStreamStatus = 2
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

// StreamSender allows sending data to a stream.
// StreamSender is EXPERIMENTAL and is a subject to change till Grafana 8.
type StreamSender interface {
	// SendFrame allows sending the entire data frame to a stream.
	SendFrame(*data.Frame) error
	// SendFrameSchema allows sending the schema part of data frame to a stream (without data).
	SendFrameSchema(*data.Frame) error
	// SendFrameData allows sending the data part of data frame to a stream (without schema).
	SendFrameData(*data.Frame) error
	// SendJSON allows sending arbitrary JSON payload to a stream.
	SendJSON([]byte) error
}

type streamSender struct {
	srv pluginv2.Stream_RunStreamServer
}

func (s *streamSender) SendFrame(frame *data.Frame) error {
	frameJSON, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	packet := &pluginv2.StreamPacket{
		Data: frameJSON,
	}
	return s.srv.Send(packet)
}

func (s *streamSender) SendFrameSchema(frame *data.Frame) error {
	frameJSON, err := data.FrameToJSON(frame, true, false)
	if err != nil {
		return err
	}
	packet := &pluginv2.StreamPacket{
		Data: frameJSON,
	}
	return s.srv.Send(packet)
}

func (s *streamSender) SendFrameData(frame *data.Frame) error {
	frameJSON, err := data.FrameToJSON(frame, false, true)
	if err != nil {
		return err
	}
	packet := &pluginv2.StreamPacket{
		Data: frameJSON,
	}
	return s.srv.Send(packet)
}

func (s *streamSender) SendJSON(data []byte) error {
	if !json.Valid(data) {
		return fmt.Errorf("invalid JSON data")
	}
	packet := &pluginv2.StreamPacket{
		Data: data,
	}
	return s.srv.Send(packet)
}
