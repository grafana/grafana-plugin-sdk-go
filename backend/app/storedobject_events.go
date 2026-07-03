package app

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/storedobjects"
)

func anyStoredObjectDeclaresEvents(stored []schemabuilder.StoredObjectInfo) bool {
	for _, s := range stored {
		if s.Events {
			return true
		}
	}
	return false
}

// brokerStoredObjectEventHandler is the default StoredObjectEventHandler:
// it feeds every event Grafana pushes over the plugin protocol into the
// experimental/storedobjects broker, where Collection.Watch subscriptions
// pick them up. Kept private for the same reason as the derived admission
// handler; the entry point is declaring Events on Schema.StoredObjects.
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
		// Mirrors the on-prem org-to-namespace derivation the
		// storedobjects client falls back to, so events land on the same
		// namespace a Watch subscribed with.
		if event.PluginContext.OrgID == 1 {
			namespace = "default"
		} else {
			namespace = fmt.Sprintf("org-%d", event.PluginContext.OrgID)
		}
	}
	storedobjects.PublishEvent(namespace, event.Kind, evtType, event.ObjectBytes)
	return nil
}
