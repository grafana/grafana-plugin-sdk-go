package app

import (
	"context"
	"io"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/storedobjects"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

func TestDeclaredStoredObjectKinds(t *testing.T) {
	require.Empty(t, declaredStoredObjectKinds(nil))
	require.Empty(t, declaredStoredObjectKinds(&Schema{}))
	require.Equal(t, []string{"Watchlist", "ClusterRule"}, declaredStoredObjectKinds(&Schema{
		StoredObjects: []schemabuilder.StoredObjectInfo{
			{Name: "Watchlist"},
			{Name: "ClusterRule"},
		},
	}))
}

// fakeEventStream implements the generated bidi stream server interface so
// the full plugin-side path (gRPC adapter -> default handler -> broker ->
// Collection.Watch, plus Watch-driven subscription updates) can be exercised
// in-process.
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

func (s *fakeEventStream) subscriptions() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]string, 0, len(s.sent))
	for _, sub := range s.sent {
		out = append(out, sub.Kinds)
	}
	return out
}

func TestStoredObjectEventsEndToEnd(t *testing.T) {
	type watchlistSpec struct {
		Title string `json:"title"`
	}
	type watchlistStatus struct {
		State string `json:"state"`
	}

	// The Watch broker is process-wide; wait out watcher teardown from any
	// earlier test so the kind set starts empty.
	kindSub := storedobjects.DefaultKindSubscription()
	require.Eventually(t, func() bool {
		return len(kindSub.Kinds()) == 0
	}, 5*time.Second, 10*time.Millisecond)

	// Server side: the same adapter wiring backend.Manage builds from
	// ServeOpts, using the broker-backed handler and subscription Manage
	// wires when the schema declares stored objects.
	pluginOpts, err := backend.GRPCServeOpts(backend.ServeOpts{
		StoredObjectEventHandler:      brokerStoredObjectEventHandler{},
		StoredObjectEventSubscription: kindSub,
	})
	require.NoError(t, err)
	require.NotNil(t, pluginOpts.StoredObjectEventsServer)

	stream := &fakeEventStream{
		ctx:    context.Background(),
		events: make(chan *pluginv2.StoredObjectEvent),
	}
	adapterDone := make(chan error, 1)
	go func() { adapterDone <- pluginOpts.StoredObjectEventsServer.StreamStoredObjectEvents(stream) }()

	// No Watch exists yet, so the plugin must stay silent on the stream.
	time.Sleep(100 * time.Millisecond)
	require.Empty(t, stream.subscriptions())

	// Consumer side: typed Watch subscriptions like a plugin instance would
	// hold. The namespace matches the one carried by the pushed events.
	client, err := storedobjects.NewClient(storedobjects.ClientOpts{
		AppURL:       "http://grafana:3000",
		Token:        "t",
		Group:        "my-app",
		OrgNamespace: "default",
	})
	require.NoError(t, err)
	coll := storedobjects.NewCollection[watchlistSpec, watchlistStatus](client, "Watchlist")

	// First watcher for the kind: a subscription update must be sent.
	watchCtx1, cancelWatch1 := context.WithCancel(context.Background())
	defer cancelWatch1()
	events1, err := coll.Watch(watchCtx1)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		subs := stream.subscriptions()
		return len(subs) == 1 && slices.Equal(subs[0], []string{"Watchlist"})
	}, 5*time.Second, 10*time.Millisecond)

	// Second watcher for the same kind: the desired set is unchanged, so no
	// duplicate update may be sent.
	watchCtx2, cancelWatch2 := context.WithCancel(context.Background())
	defer cancelWatch2()
	events2, err := coll.Watch(watchCtx2)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	require.Len(t, stream.subscriptions(), 1)

	// Events pushed over the stream reach every watcher through the broker.
	stream.events <- &pluginv2.StoredObjectEvent{
		PluginContext: &pluginv2.PluginContext{PluginId: "my-app", Namespace: "default"},
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

	recv := func(events <-chan storedobjects.Event[watchlistSpec, watchlistStatus]) storedobjects.Event[watchlistSpec, watchlistStatus] {
		select {
		case ev, ok := <-events:
			require.True(t, ok, "watch channel closed unexpectedly")
			return ev
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for event")
			panic("unreachable")
		}
	}

	for _, events := range []<-chan storedobjects.Event[watchlistSpec, watchlistStatus]{events1, events2} {
		created := recv(events)
		require.Equal(t, storedobjects.EventCreated, created.Type)
		require.Equal(t, "one", created.Item.Name)
		require.Equal(t, "One", created.Item.Spec.Title)
		require.Equal(t, "ok", created.Item.Status.State)

		deleted := recv(events)
		require.Equal(t, storedobjects.EventDeleted, deleted.Type)
		require.Equal(t, "one", deleted.Item.Name)
	}

	// Dropping one of two watchers leaves the kind wanted: no update.
	cancelWatch2()
	select {
	case _, open := <-events2:
		require.False(t, open, "watch channel should close after cancel")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for watch channel close")
	}
	time.Sleep(100 * time.Millisecond)
	require.Len(t, stream.subscriptions(), 1)

	// Dropping the last watcher sends the empty set, pausing pushes without
	// closing the stream.
	cancelWatch1()
	require.Eventually(t, func() bool {
		subs := stream.subscriptions()
		return len(subs) == 2 && len(subs[1]) == 0
	}, 5*time.Second, 10*time.Millisecond)

	close(stream.events)
	require.NoError(t, <-adapterDone)
}

func TestBrokerHandlerRejectsUnknownEventType(t *testing.T) {
	err := brokerStoredObjectEventHandler{}.HandleStoredObjectEvent(context.Background(), &backend.StoredObjectEvent{
		Kind: "Watchlist",
		Type: backend.StoredObjectEventUnknown,
	})
	require.Error(t, err)
}
