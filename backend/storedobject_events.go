package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// StoredObjectEventHandler is an EXPERIMENTAL handler for change events that
// Grafana pushes for the stored object kinds the plugin has subscribed to
// over the StoredObjectEvents stream. Only new events are delivered; existing
// objects are not replayed.
type StoredObjectEventHandler interface {
	// HandleStoredObjectEvent handles a single change event.
	HandleStoredObjectEvent(ctx context.Context, event *StoredObjectEvent) error
}

// StoredObjectEventSubscription is an EXPERIMENTAL source of the stored
// object kinds the plugin currently wants change events for. The SDK reads
// it while the StoredObjectEvents stream is open and sends Grafana a full
// replacement kind set whenever the desired set changes.
//
// The protocol contract: the plugin stays silent until it wants at least one
// kind (Grafana pushes nothing before the first non-empty subscription), and
// after that every change is sent, including a transition to the empty set,
// which pauses pushes without closing the stream.
type StoredObjectEventSubscription interface {
	// Kinds returns the kinds the plugin currently wants events for. The
	// returned slice must be safe for the caller to sort and retain.
	Kinds() []string

	// Changes registers for change notifications: the returned channel is
	// signaled whenever the desired kind set may have changed (receivers
	// re-read Kinds), and the returned func unregisters the channel.
	Changes() (<-chan struct{}, func())
}

// NewStaticStoredObjectEventSubscription returns a subscription whose desired
// kind set never changes. Used when the desired kinds are known up front,
// e.g. an explicit handler that wants every declared kind.
func NewStaticStoredObjectEventSubscription(kinds ...string) StoredObjectEventSubscription {
	return staticStoredObjectEventSubscription{kinds: kinds}
}

type staticStoredObjectEventSubscription struct {
	kinds []string
}

func (s staticStoredObjectEventSubscription) Kinds() []string {
	out := make([]string, len(s.kinds))
	copy(out, s.kinds)
	return out
}

func (s staticStoredObjectEventSubscription) Changes() (<-chan struct{}, func()) {
	// The set is fixed, so the channel never fires; the adapter still sends
	// the initial set when the stream opens.
	return make(chan struct{}), func() {}
}

// StoredObjectEventType is the kind of change a stored object event carries.
type StoredObjectEventType int32

const (
	StoredObjectEventUnknown StoredObjectEventType = 0
	StoredObjectEventCreated StoredObjectEventType = 1
	StoredObjectEventUpdated StoredObjectEventType = 2
	StoredObjectEventDeleted StoredObjectEventType = 3
)

// String textual representation of the event type.
func (t StoredObjectEventType) String() string {
	return pluginv2.StoredObjectEvent_EventType(t).String()
}

// StoredObjectEvent describes a single change to a stored object.
type StoredObjectEvent struct {
	// NOTE: this may not include populated instance settings depending on the request
	PluginContext PluginContext `json:"pluginContext,omitempty"`
	// Kind is the declared object type name, e.g. "Watchlist"
	Kind string `json:"kind,omitempty"`
	// Type is the kind of change that occurred
	Type StoredObjectEventType `json:"type,omitempty"`
	// ObjectBytes is the full JSON envelope after the change (the last-known state for deletes)
	ObjectBytes []byte `json:"object_bytes,omitempty"`
}
