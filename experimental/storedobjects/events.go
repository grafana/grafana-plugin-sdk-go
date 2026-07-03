package storedobjects

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// EventType describes the kind of change a stored object event carries.
type EventType string

const (
	// EventCreated means the item was created.
	EventCreated EventType = "created"
	// EventUpdated means the item's spec or status changed.
	EventUpdated EventType = "updated"
	// EventDeleted means the item was deleted. The event's item carries the
	// last-known state.
	EventDeleted EventType = "deleted"
)

// Event is a single change to an item in a collection.
type Event[S, T any] struct {
	// Type is the kind of change.
	Type EventType
	// Item is the item after the change (the last-known state for deletes).
	Item Item[S, T]
}

// Watch returns a channel of change events for the collection's object type
// in the client's org namespace, provided the plugin's schema declares Events
// for the type. Only changes that happen after Grafana connects the event
// stream are delivered; existing items are not replayed, so callers that need
// current state should List first and then apply events.
//
// The channel is closed when ctx is canceled. A consumer that falls behind
// has its oldest pending events dropped rather than stalling delivery to
// other consumers, so treat events as change notifications, not a complete
// history.
func (c *Collection[S, T]) Watch(ctx context.Context) (<-chan Event[S, T], error) {
	sub := defaultBroker.subscribe(c.client.orgNamespace, c.name)
	out := make(chan Event[S, T])
	go func() {
		defer close(out)
		defer defaultBroker.unsubscribe(sub)
		for {
			select {
			case <-ctx.Done():
				return
			case raw := <-sub.ch:
				var env objectEnvelope
				if err := json.Unmarshal(raw.object, &env); err != nil {
					log.DefaultLogger.Warn("storedobjects: dropping undecodable event", "type", c.name, "error", err)
					continue
				}
				item, err := itemFromEnvelope[S, T](env)
				if err != nil {
					log.DefaultLogger.Warn("storedobjects: dropping undecodable event", "type", c.name, "error", err)
					continue
				}
				select {
				case out <- Event[S, T]{Type: raw.eventType, Item: item}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

// PublishEvent delivers a change event to every Watch subscription matching
// the namespace and object type name. It is the entry point the SDK's serve
// layer feeds with events received from Grafana; plugin code normally never
// calls it. objectJSON is the full JSON envelope of the object after the
// change (the last-known state for deletes).
func PublishEvent(namespace, name string, evtType EventType, objectJSON []byte) {
	defaultBroker.publish(namespace, name, evtType, objectJSON)
}

// rawEvent is an event before it is decoded into a subscriber's typed shape.
// Each subscriber decodes its own copy so subscribers with different S/T
// types can share one broker.
type rawEvent struct {
	eventType EventType
	object    []byte
}

// subscription is one Watch call's registration with the broker.
type subscription struct {
	namespace string
	name      string
	ch        chan rawEvent
}

// subscriberBuffer is the number of undelivered events a subscription can
// hold before the broker starts dropping its oldest events.
const subscriberBuffer = 16

// broker fans change events out to Watch subscriptions in-process.
type broker struct {
	mu   sync.Mutex
	subs map[*subscription]struct{}
}

var defaultBroker = &broker{subs: map[*subscription]struct{}{}}

func (b *broker) subscribe(namespace, name string) *subscription {
	sub := &subscription{
		namespace: namespace,
		name:      name,
		ch:        make(chan rawEvent, subscriberBuffer),
	}
	b.mu.Lock()
	b.subs[sub] = struct{}{}
	b.mu.Unlock()
	return sub
}

func (b *broker) unsubscribe(sub *subscription) {
	b.mu.Lock()
	delete(b.subs, sub)
	b.mu.Unlock()
}

func (b *broker) publish(namespace, name string, evtType EventType, objectJSON []byte) {
	ev := rawEvent{eventType: evtType, object: objectJSON}
	b.mu.Lock()
	defer b.mu.Unlock()
	for sub := range b.subs {
		if sub.namespace != namespace || sub.name != name {
			continue
		}
		// A slow subscriber must not block the publisher (it is dispatching
		// events from the gRPC stream) or starve other subscribers. Drop the
		// subscriber's oldest pending event rather than the new one: the
		// newest event is the closest to current state, which is what a
		// consumer that fell behind needs most.
		select {
		case sub.ch <- ev:
		default:
			select {
			case <-sub.ch:
				log.DefaultLogger.Warn("storedobjects: slow event consumer, dropping oldest pending event", "type", name, "namespace", namespace)
			default:
			}
			select {
			case sub.ch <- ev:
			default:
			}
		}
	}
}
