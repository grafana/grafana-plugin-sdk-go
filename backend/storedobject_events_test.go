package backend

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

func TestConvertStoredObjectEvent(t *testing.T) {
	protoEvent := &pluginv2.StoredObjectEvent{
		PluginContext: &pluginv2.PluginContext{
			OrgId:     3,
			PluginId:  "my-app",
			Namespace: "org-3",
		},
		Kind:   "Watchlist",
		Type:   pluginv2.StoredObjectEvent_UPDATED,
		Object: []byte(`{"metadata": {"name": "one"}}`),
	}

	sdkEvent := FromProto().StoredObjectEvent(protoEvent)
	require.Equal(t, int64(3), sdkEvent.PluginContext.OrgID)
	require.Equal(t, "my-app", sdkEvent.PluginContext.PluginID)
	require.Equal(t, "org-3", sdkEvent.PluginContext.Namespace)
	require.Equal(t, "Watchlist", sdkEvent.Kind)
	require.Equal(t, StoredObjectEventUpdated, sdkEvent.Type)
	require.Equal(t, protoEvent.Object, sdkEvent.ObjectBytes)

	roundTripped := ToProto().StoredObjectEvent(sdkEvent)
	require.Equal(t, protoEvent.PluginContext.OrgId, roundTripped.PluginContext.OrgId)
	require.Equal(t, protoEvent.PluginContext.PluginId, roundTripped.PluginContext.PluginId)
	require.Equal(t, protoEvent.PluginContext.Namespace, roundTripped.PluginContext.Namespace)
	require.Equal(t, protoEvent.Kind, roundTripped.Kind)
	require.Equal(t, protoEvent.Type, roundTripped.Type)
	require.Equal(t, protoEvent.Object, roundTripped.Object)
}

func TestStoredObjectEventProtoRoundTrip(t *testing.T) {
	src := &pluginv2.StoredObjectEvent{
		PluginContext: &pluginv2.PluginContext{PluginId: "my-app", Namespace: "default"},
		Kind:          "Watchlist",
		Type:          pluginv2.StoredObjectEvent_DELETED,
		Object:        []byte(`{"metadata": {"name": "gone"}}`),
	}

	raw, err := proto.Marshal(src)
	require.NoError(t, err)

	dst := &pluginv2.StoredObjectEvent{}
	require.NoError(t, proto.Unmarshal(raw, dst))
	require.True(t, proto.Equal(src, dst))
}

func TestStoredObjectEventTypeString(t *testing.T) {
	require.Equal(t, "UNKNOWN", StoredObjectEventUnknown.String())
	require.Equal(t, "CREATED", StoredObjectEventCreated.String())
	require.Equal(t, "UPDATED", StoredObjectEventUpdated.String())
	require.Equal(t, "DELETED", StoredObjectEventDeleted.String())
}

// fakeEventStream implements the generated client-streaming server interface
// so the adapter's receive loop can be driven without a real gRPC connection.
type fakeEventStream struct {
	grpc.ServerStream
	ctx    context.Context
	events chan *pluginv2.StoredObjectEvent
	closed bool
}

func (s *fakeEventStream) Context() context.Context { return s.ctx }

func (s *fakeEventStream) Recv() (*pluginv2.StoredObjectEvent, error) {
	select {
	case ev, ok := <-s.events:
		if !ok {
			return nil, io.EOF
		}
		return ev, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

func (s *fakeEventStream) SendAndClose(*pluginv2.StoredObjectEventsResponse) error {
	s.closed = true
	return nil
}

type capturingEventHandler struct {
	events []*StoredObjectEvent
	err    error
}

func (h *capturingEventHandler) HandleStoredObjectEvent(_ context.Context, event *StoredObjectEvent) error {
	h.events = append(h.events, event)
	return h.err
}

func TestStoredObjectEventsAdapterDispatchesUntilEOF(t *testing.T) {
	handler := &capturingEventHandler{}
	adapter := newStoredObjectEventsSDKAdapter(handler)

	stream := &fakeEventStream{
		ctx:    context.Background(),
		events: make(chan *pluginv2.StoredObjectEvent, 2),
	}
	stream.events <- &pluginv2.StoredObjectEvent{
		PluginContext: &pluginv2.PluginContext{PluginId: "my-app", Namespace: "default"},
		Kind:          "Watchlist",
		Type:          pluginv2.StoredObjectEvent_CREATED,
		Object:        []byte(`{"metadata": {"name": "one"}}`),
	}
	stream.events <- &pluginv2.StoredObjectEvent{
		PluginContext: &pluginv2.PluginContext{PluginId: "my-app", Namespace: "default"},
		Kind:          "Watchlist",
		Type:          pluginv2.StoredObjectEvent_DELETED,
		Object:        []byte(`{"metadata": {"name": "one"}}`),
	}
	close(stream.events)

	require.NoError(t, adapter.StreamStoredObjectEvents(stream))
	require.True(t, stream.closed)
	require.Len(t, handler.events, 2)
	require.Equal(t, StoredObjectEventCreated, handler.events[0].Type)
	require.Equal(t, "Watchlist", handler.events[0].Kind)
	require.Equal(t, "default", handler.events[0].PluginContext.Namespace)
	require.Equal(t, StoredObjectEventDeleted, handler.events[1].Type)
}

func TestStoredObjectEventsAdapterPropagatesCancellation(t *testing.T) {
	handler := &capturingEventHandler{}
	adapter := newStoredObjectEventsSDKAdapter(handler)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stream := &fakeEventStream{
		ctx:    ctx,
		events: make(chan *pluginv2.StoredObjectEvent),
	}

	err := adapter.StreamStoredObjectEvents(stream)
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, stream.closed)
	require.Empty(t, handler.events)
}

func TestStoredObjectEventsAdapterStopsOnHandlerError(t *testing.T) {
	handler := &capturingEventHandler{err: io.ErrUnexpectedEOF}
	adapter := newStoredObjectEventsSDKAdapter(handler)

	stream := &fakeEventStream{
		ctx:    context.Background(),
		events: make(chan *pluginv2.StoredObjectEvent, 1),
	}
	stream.events <- &pluginv2.StoredObjectEvent{
		PluginContext: &pluginv2.PluginContext{PluginId: "my-app"},
		Kind:          "Watchlist",
		Type:          pluginv2.StoredObjectEvent_CREATED,
	}

	err := adapter.StreamStoredObjectEvents(stream)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	require.False(t, stream.closed)
}
