package storedobjects

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// newWatchCollection returns a collection whose client never touches HTTP;
// Watch only needs the group/namespace identity.
func newWatchCollection(t *testing.T, namespace string) *Collection[watchlistSpec, watchlistStatus] {
	t.Helper()
	c, err := NewClient(ClientOpts{AppURL: "http://grafana:3000", Token: "t", Group: "my-app", OrgNamespace: namespace})
	require.NoError(t, err)
	return NewCollection[watchlistSpec, watchlistStatus](c, "Watchlist")
}

// recvEvent reads one event with a deadline so a broken broker fails the test
// instead of hanging it.
func recvEvent[S, T any](t *testing.T, ch <-chan Event[S, T]) Event[S, T] {
	t.Helper()
	select {
	case ev, ok := <-ch:
		require.True(t, ok, "event channel closed unexpectedly")
		return ev
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for event")
		return Event[S, T]{}
	}
}

func TestWatchReceivesMatchingEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	coll := newWatchCollection(t, "watch-match")
	events, err := coll.Watch(ctx)
	require.NoError(t, err)

	// Non-matching namespace and non-matching type name are enqueued first:
	// if the broker delivered them, they would arrive before the matching
	// event since per-subscription order is preserved.
	PublishEvent("other-namespace", "Watchlist", EventCreated, []byte(`{"metadata": {"name": "wrong-ns"}, "spec": {"title": "x"}}`))
	PublishEvent("watch-match", "ClusterRule", EventCreated, []byte(`{"metadata": {"name": "wrong-kind"}, "spec": {"title": "x"}}`))
	PublishEvent("watch-match", "Watchlist", EventCreated, []byte(`{"metadata": {"name": "mine", "labels": {"team": "a"}}, "spec": {"title": "Mine"}, "status": {"state": "ok"}}`))

	ev := recvEvent(t, events)
	require.Equal(t, EventCreated, ev.Type)
	require.Equal(t, "mine", ev.Item.Name)
	require.Equal(t, map[string]string{"team": "a"}, ev.Item.Labels)
	require.Equal(t, "Mine", ev.Item.Spec.Title)
	require.Equal(t, "ok", ev.Item.Status.State)

	PublishEvent("watch-match", "Watchlist", EventDeleted, []byte(`{"metadata": {"name": "mine"}, "spec": {"title": "Mine"}}`))
	ev = recvEvent(t, events)
	require.Equal(t, EventDeleted, ev.Type)
	require.Equal(t, "mine", ev.Item.Name)
}

func TestWatchSkipsUndecodableEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	coll := newWatchCollection(t, "watch-decode")
	events, err := coll.Watch(ctx)
	require.NoError(t, err)

	PublishEvent("watch-decode", "Watchlist", EventCreated, []byte(`not json`))
	PublishEvent("watch-decode", "Watchlist", EventUpdated, []byte(`{"metadata": {"name": "good"}, "spec": {"title": "Good"}}`))

	ev := recvEvent(t, events)
	require.Equal(t, EventUpdated, ev.Type)
	require.Equal(t, "good", ev.Item.Name)
}

func TestWatchClosesOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	coll := newWatchCollection(t, "watch-cancel")
	events, err := coll.Watch(ctx)
	require.NoError(t, err)

	cancel()
	select {
	case _, ok := <-events:
		require.False(t, ok, "channel should be closed after cancel")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}

	// The subscription must be gone from the broker so later publishes don't
	// pile up in a dead channel.
	require.Eventually(t, func() bool {
		defaultBroker.mu.Lock()
		defer defaultBroker.mu.Unlock()
		for sub := range defaultBroker.subs {
			if sub.namespace == "watch-cancel" {
				return false
			}
		}
		return true
	}, 5*time.Second, 10*time.Millisecond)
}

func TestBrokerDropsOldestOnOverflow(t *testing.T) {
	sub := defaultBroker.subscribe("overflow", "Watchlist")
	defer defaultBroker.unsubscribe(sub)

	// One more event than the buffer holds: the first published event is the
	// one that must be dropped.
	for i := 0; i <= subscriberBuffer; i++ {
		defaultBroker.publish("overflow", "Watchlist", EventUpdated, []byte(fmt.Sprintf(`{"metadata": {"name": "item-%d"}}`, i)))
	}

	require.Len(t, sub.ch, subscriberBuffer)
	first := <-sub.ch
	require.JSONEq(t, `{"metadata": {"name": "item-1"}}`, string(first.object))

	// Drain and confirm the newest event survived.
	var last rawEvent
	for len(sub.ch) > 0 {
		last = <-sub.ch
	}
	require.JSONEq(t, fmt.Sprintf(`{"metadata": {"name": "item-%d"}}`, subscriberBuffer), string(last.object))
}
