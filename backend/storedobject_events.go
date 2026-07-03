package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// StoredObjectEventHandler is an EXPERIMENTAL handler for change events that
// Grafana pushes for the stored object kinds a plugin declares Events for in
// its schema artifact. Only new events are delivered; existing objects are
// not replayed.
type StoredObjectEventHandler interface {
	// HandleStoredObjectEvent handles a single change event.
	HandleStoredObjectEvent(ctx context.Context, event *StoredObjectEvent) error
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
