package app

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/storedobjects"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

func TestAnyStoredObjectDeclaresEvents(t *testing.T) {
	require.False(t, anyStoredObjectDeclaresEvents(nil))
	require.False(t, anyStoredObjectDeclaresEvents([]schemabuilder.StoredObjectInfo{{Name: "Watchlist"}}))
	require.True(t, anyStoredObjectDeclaresEvents([]schemabuilder.StoredObjectInfo{
		{Name: "Watchlist"},
		{Name: "ClusterRule", Events: true},
	}))
}

// fakeEventStream implements the generated client-streaming server interface
// so the full plugin-side path (gRPC adapter -> default handler -> broker ->
// Collection.Watch) can be exercised in-process.
type fakeEventStream struct {
	grpc.ServerStream
	ctx    context.Context
	events chan *pluginv2.StoredObjectEvent
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

func (s *fakeEventStream) SendAndClose(*pluginv2.StoredObjectEventsResponse) error { return nil }

func TestStoredObjectEventsEndToEnd(t *testing.T) {
	type watchlistSpec struct {
		Title string `json:"title"`
	}
	type watchlistStatus struct {
		State string `json:"state"`
	}

	// Server side: the same adapter wiring backend.Manage builds from
	// ServeOpts, using the default broker-backed handler Manage wires when a
	// declared stored object opts into events.
	pluginOpts, err := backend.GRPCServeOpts(backend.ServeOpts{
		StoredObjectEventHandler: brokerStoredObjectEventHandler{},
	})
	require.NoError(t, err)
	require.NotNil(t, pluginOpts.StoredObjectEventsServer)

	// Consumer side: a typed Watch subscription like a plugin instance would
	// hold. The namespace matches what the handler derives from OrgID 1.
	client, err := storedobjects.NewClient(storedobjects.ClientOpts{
		AppURL:       "http://grafana:3000",
		Token:        "t",
		Group:        "my-app",
		OrgNamespace: "default",
	})
	require.NoError(t, err)
	coll := storedobjects.NewCollection[watchlistSpec, watchlistStatus](client, "Watchlist")

	watchCtx, cancelWatch := context.WithCancel(context.Background())
	defer cancelWatch()
	events, err := coll.Watch(watchCtx)
	require.NoError(t, err)

	stream := &fakeEventStream{
		ctx:    context.Background(),
		events: make(chan *pluginv2.StoredObjectEvent, 2),
	}
	stream.events <- &pluginv2.StoredObjectEvent{
		PluginContext: &pluginv2.PluginContext{PluginId: "my-app", OrgId: 1},
		Kind:          "Watchlist",
		Type:          pluginv2.StoredObjectEvent_CREATED,
		Object:        []byte(`{"metadata": {"name": "one"}, "spec": {"title": "One"}, "status": {"state": "ok"}}`),
	}
	stream.events <- &pluginv2.StoredObjectEvent{
		PluginContext: &pluginv2.PluginContext{PluginId: "my-app", Namespace: "default"},
		Kind:          "Watchlist",
		Type:          pluginv2.StoredObjectEvent_DELETED,
		Object:        []byte(`{"metadata": {"name": "one"}, "spec": {"title": "One"}}`),
	}
	close(stream.events)

	require.NoError(t, pluginOpts.StoredObjectEventsServer.StreamStoredObjectEvents(stream))

	recv := func() storedobjects.Event[watchlistSpec, watchlistStatus] {
		select {
		case ev, ok := <-events:
			require.True(t, ok, "watch channel closed unexpectedly")
			return ev
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for event")
			panic("unreachable")
		}
	}

	created := recv()
	require.Equal(t, storedobjects.EventCreated, created.Type)
	require.Equal(t, "one", created.Item.Name)
	require.Equal(t, "One", created.Item.Spec.Title)
	require.Equal(t, "ok", created.Item.Status.State)

	deleted := recv()
	require.Equal(t, storedobjects.EventDeleted, deleted.Type)
	require.Equal(t, "one", deleted.Item.Name)
}

func TestBrokerHandlerRejectsUnknownEventType(t *testing.T) {
	err := brokerStoredObjectEventHandler{}.HandleStoredObjectEvent(context.Background(), &backend.StoredObjectEvent{
		Kind: "Watchlist",
		Type: backend.StoredObjectEventUnknown,
	})
	require.Error(t, err)
}
