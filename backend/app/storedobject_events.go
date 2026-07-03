package app

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/storedobjects"
)

// declaredStoredObjectKinds returns the kind names declared in the schema.
// Used to build the all-kinds subscription an explicit event handler gets.
func declaredStoredObjectKinds(schema *Schema) []string {
	if schema == nil {
		return nil
	}
	kinds := make([]string, 0, len(schema.StoredObjects))
	for _, s := range schema.StoredObjects {
		kinds = append(kinds, s.Name)
	}
	return kinds
}

// brokerStoredObjectEventHandler is the default StoredObjectEventHandler:
// it feeds every event Grafana pushes over the plugin protocol into the
// experimental/storedobjects broker, where Collection.Watch subscriptions
// pick them up. Kept private for the same reason as the derived admission
// handler; the entry point is calling Collection.Watch, which is also what
// subscribes the plugin to the kind's events.
type brokerStoredObjectEventHandler struct{}

func (brokerStoredObjectEventHandler) HandleStoredObjectEvent(_ context.Context, event *backend.StoredObjectEvent) error {
	var evtType storedobjects.EventType
	switch event.Type {
	case backend.StoredObjectEventCreated:
		evtType = storedobjects.EventCreated
	case backend.StoredObjectEventUpdated:
		evtType = storedobjects.EventUpdated
	case backend.StoredObjectEventDeleted:
		evtType = storedobjects.EventDeleted
	default:
		return fmt.Errorf("unknown stored object event type %q", event.Type)
	}
	namespace := event.PluginContext.Namespace
	if namespace == "" {
		// Namespace is how events are routed to Watch subscribers; Grafana
		// always sets it on pushed events, so an empty value is a protocol
		// violation rather than something to derive around.
		return fmt.Errorf("stored object event for %q has no namespace", event.Kind)
	}
	storedobjects.PublishEvent(namespace, event.Kind, evtType, event.ObjectBytes)
	return nil
}
