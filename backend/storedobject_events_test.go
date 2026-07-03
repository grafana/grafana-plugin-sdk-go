package backend

import (
	"context"
	"io"
	"slices"
	"sync"
	"testing"
	"time"

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

func TestStoredObjectEventsSubscriptionProtoRoundTrip(t *testing.T) {
	src := &pluginv2.StoredObjectEventsSubscription{
		Kinds: []string{"ClusterRule", "Watchlist"},
	}

	raw, err := proto.Marshal(src)
	require.NoError(t, err)

	dst := &pluginv2.StoredObjectEventsSubscription{}
	require.NoError(t, proto.Unmarshal(raw, dst))
	require.True(t, proto.Equal(src, dst))
}

func TestStoredObjectEventTypeString(t *testing.T) {
	require.Equal(t, "UNKNOWN", StoredObjectEventUnknown.String())
	require.Equal(t, "CREATED", StoredObjectEventCreated.String())
	require.Equal(t, "UPDATED", StoredObjectEventUpdated.String())
	require.Equal(t, "DELETED", StoredObjectEventDeleted.String())
}

// fakeEventStream implements the generated bidi stream server interface so
// the adapter's receive and subscription-send loops can be driven without a
// real gRPC connection.
type fakeEventStream struct {
	grpc.ServerStream
	ctx    context.Context
	events chan *pluginv2.StoredObjectEvent

	mu   sync.Mutex
	sent []*pluginv2.StoredObjectEventsSubscription
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

func (s *fakeEventStream) Send(sub *pluginv2.StoredObjectEventsSubscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, sub)
	return nil
}

// subscriptions returns the kind sets sent so far, in send order.
func (s *fakeEventStream) subscriptions() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]string, 0, len(s.sent))
	for _, sub := range s.sent {
		out = append(out, sub.Kinds)
	}
	return out
}

type capturingEventHandler struct {
	events []*StoredObjectEvent
	err    error
}

func (h *capturingEventHandler) HandleStoredObjectEvent(_ context.Context, event *StoredObjectEvent) error {
	h.events = append(h.events, event)
	return h.err
}

// fakeSubscription is a mutable StoredObjectEventSubscription for driving the
// adapter's send loop in tests.
type fakeSubscription struct {
	mu      sync.Mutex
	kinds   []string
	changes chan struct{}
}

func newFakeSubscription(kinds ...string) *fakeSubscription {
	return &fakeSubscription{kinds: kinds, changes: make(chan struct{}, 1)}
}

func (f *fakeSubscription) Kinds() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.kinds))
	copy(out, f.kinds)
	return out
}

func (f *fakeSubscription) Changes() (<-chan struct{}, func()) {
	return f.changes, func() {}
}

func (f *fakeSubscription) set(kinds ...string) {
	f.mu.Lock()
	f.kinds = kinds
	f.mu.Unlock()
	select {
	case f.changes <- struct{}{}:
	default:
	}
}

func TestStoredObjectEventsAdapterDispatchesUntilEOF(t *testing.T) {
	handler := &capturingEventHandler{}
	adapter := newStoredObjectEventsSDKAdapter(handler, nil)

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
	require.Len(t, handler.events, 2)
	require.Equal(t, StoredObjectEventCreated, handler.events[0].Type)
	require.Equal(t, "Watchlist", handler.events[0].Kind)
	require.Equal(t, "default", handler.events[0].PluginContext.Namespace)
	require.Equal(t, StoredObjectEventDeleted, handler.events[1].Type)
	// No subscription source: the plugin must stay silent on the stream.
	require.Empty(t, stream.subscriptions())
}

func TestStoredObjectEventsAdapterPropagatesCancellation(t *testing.T) {
	handler := &capturingEventHandler{}
	adapter := newStoredObjectEventsSDKAdapter(handler, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stream := &fakeEventStream{
		ctx:    ctx,
		events: make(chan *pluginv2.StoredObjectEvent),
	}

	err := adapter.StreamStoredObjectEvents(stream)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, handler.events)
}

func TestStoredObjectEventsAdapterStopsOnHandlerError(t *testing.T) {
	handler := &capturingEventHandler{err: io.ErrUnexpectedEOF}
	adapter := newStoredObjectEventsSDKAdapter(handler, nil)

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
}

func TestStoredObjectEventsAdapterSendsInitialNonEmptySet(t *testing.T) {
	handler := &capturingEventHandler{}
	sub := newFakeSubscription("Watchlist", "ClusterRule")
	adapter := newStoredObjectEventsSDKAdapter(handler, sub)

	stream := &fakeEventStream{
		ctx:    context.Background(),
		events: make(chan *pluginv2.StoredObjectEvent),
	}

	done := make(chan error, 1)
	go func() { done <- adapter.StreamStoredObjectEvents(stream) }()

	// The set desired at stream open is sent right away, sorted.
	require.Eventually(t, func() bool {
		subs := stream.subscriptions()
		return len(subs) == 1 && slices.Equal(subs[0], []string{"ClusterRule", "Watchlist"})
	}, 5*time.Second, 10*time.Millisecond)

	close(stream.events)
	require.NoError(t, <-done)
}

func TestStoredObjectEventsAdapterSubscriptionFlow(t *testing.T) {
	handler := &capturingEventHandler{}
	sub := newFakeSubscription()
	adapter := newStoredObjectEventsSDKAdapter(handler, sub)

	stream := &fakeEventStream{
		ctx:    context.Background(),
		events: make(chan *pluginv2.StoredObjectEvent),
	}

	done := make(chan error, 1)
	go func() { done <- adapter.StreamStoredObjectEvents(stream) }()

	// The desired set is empty at stream open: nothing may be sent until at
	// least one kind is wanted.
	time.Sleep(100 * time.Millisecond)
	require.Empty(t, stream.subscriptions())

	sub.set("Watchlist")
	require.Eventually(t, func() bool {
		subs := stream.subscriptions()
		return len(subs) == 1 && slices.Equal(subs[0], []string{"Watchlist"})
	}, 5*time.Second, 10*time.Millisecond)

	// A change signal that leaves the set identical must not produce a
	// duplicate update.
	sub.set("Watchlist")
	time.Sleep(100 * time.Millisecond)
	require.Len(t, stream.subscriptions(), 1)

	// After the first send, a transition to the empty set is sent: it pauses
	// pushes without closing the stream.
	sub.set()
	require.Eventually(t, func() bool {
		subs := stream.subscriptions()
		return len(subs) == 2 && len(subs[1]) == 0
	}, 5*time.Second, 10*time.Millisecond)

	close(stream.events)
	require.NoError(t, <-done)
}
