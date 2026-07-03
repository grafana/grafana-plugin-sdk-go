package backend

import (
	"context"
	"errors"
	"io"
	"slices"
	"sort"

	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// storedObjectEventsSDKAdapter adapter between low level plugin protocol and
// SDK interfaces.
type storedObjectEventsSDKAdapter struct {
	handler      StoredObjectEventHandler
	subscription StoredObjectEventSubscription
}

func newStoredObjectEventsSDKAdapter(handler StoredObjectEventHandler, subscription StoredObjectEventSubscription) *storedObjectEventsSDKAdapter {
	return &storedObjectEventsSDKAdapter{
		handler:      handler,
		subscription: subscription,
	}
}

// StreamStoredObjectEvents services one Grafana-opened event stream: events
// Grafana pushes are dispatched to the handler, and subscription updates
// (the full replacement kind set the plugin wants events for) are sent
// upstream whenever the desired set changes. Grafana keeps the stream open
// for the life of the plugin process and closes it on shutdown, which
// surfaces here as io.EOF (clean close) or a stream context error.
func (a *storedObjectEventsSDKAdapter) StreamStoredObjectEvents(stream grpc.BidiStreamingServer[pluginv2.StoredObjectEvent, pluginv2.StoredObjectEventsSubscription]) error {
	// Receiving and sending run concurrently (one goroutine each, which is
	// the concurrency gRPC streams permit). The derived context stops the
	// surviving loop once the other finishes.
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	errs := make(chan error, 2)
	go func() {
		errs <- a.recvLoop(ctx, stream)
	}()
	go func() {
		errs <- a.sendLoop(ctx, stream)
	}()

	// The first loop to finish decides the outcome. Waiting for the second
	// guarantees no Send or Recv happens after this handler returns, which
	// gRPC forbids.
	err := <-errs
	cancel()
	if second := <-errs; err == nil {
		err = second
	}
	return err
}

// recvLoop dispatches events pushed by Grafana to the handler until the
// stream ends or the handler fails.
func (a *storedObjectEventsSDKAdapter) recvLoop(ctx context.Context, stream grpc.BidiStreamingServer[pluginv2.StoredObjectEvent, pluginv2.StoredObjectEventsSubscription]) error {
	for {
		ev, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			// Prefer the stream's context error so callers see a plain
			// cancellation instead of a wrapped transport error when Grafana
			// tears the stream down.
			if ctxErr := stream.Context().Err(); ctxErr != nil {
				return ctxErr
			}
			return err
		}
		parsedEvent := FromProto().StoredObjectEvent(ev)
		if err := a.handler.HandleStoredObjectEvent(ctx, parsedEvent); err != nil {
			return err
		}
	}
}

// sendLoop keeps Grafana's view of the desired kind set current. Protocol
// contract: nothing is sent until the plugin wants at least one kind (so
// Grafana pushes no events before the first non-empty subscription); after
// that every change to the set is sent, including a transition to empty,
// which pauses pushes without closing the stream.
func (a *storedObjectEventsSDKAdapter) sendLoop(ctx context.Context, stream grpc.BidiStreamingServer[pluginv2.StoredObjectEvent, pluginv2.StoredObjectEventsSubscription]) error {
	if a.subscription == nil {
		// No subscription source means the plugin never subscribes over this
		// stream, so Grafana never pushes events on it.
		<-ctx.Done()
		return nil
	}

	changes, stop := a.subscription.Changes()
	defer stop()

	var sentAny bool
	var lastSent []string
	maybeSend := func() error {
		kinds := a.subscription.Kinds()
		sort.Strings(kinds)
		if !sentAny && len(kinds) == 0 {
			return nil
		}
		// The change channel coalesces notifications, so the set may be
		// unchanged by the time it is re-read; skip no-op updates.
		if sentAny && slices.Equal(kinds, lastSent) {
			return nil
		}
		if err := stream.Send(&pluginv2.StoredObjectEventsSubscription{Kinds: kinds}); err != nil {
			return err
		}
		sentAny = true
		lastSent = kinds
		return nil
	}

	// Send the set desired at stream open right away (if non-empty) so kinds
	// already being watched don't wait for the next change.
	if err := maybeSend(); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-changes:
			if err := maybeSend(); err != nil {
				return err
			}
		}
	}
}
