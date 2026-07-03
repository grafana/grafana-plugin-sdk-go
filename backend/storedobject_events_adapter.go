package backend

import (
	"errors"
	"io"

	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// storedObjectEventsSDKAdapter adapter between low level plugin protocol and
// SDK interfaces.
type storedObjectEventsSDKAdapter struct {
	handler StoredObjectEventHandler
}

func newStoredObjectEventsSDKAdapter(handler StoredObjectEventHandler) *storedObjectEventsSDKAdapter {
	return &storedObjectEventsSDKAdapter{
		handler: handler,
	}
}

// StreamStoredObjectEvents receives change events pushed by Grafana and
// dispatches each to the handler. Grafana keeps the stream open for the life
// of the plugin process and closes it on shutdown, which surfaces here as
// io.EOF (clean close) or a stream context error (cancellation).
func (a *storedObjectEventsSDKAdapter) StreamStoredObjectEvents(stream grpc.ClientStreamingServer[pluginv2.StoredObjectEvent, pluginv2.StoredObjectEventsResponse]) error {
	for {
		ev, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return stream.SendAndClose(&pluginv2.StoredObjectEventsResponse{})
		}
		if err != nil {
			// Prefer the context error so callers see a plain cancellation
			// instead of a wrapped transport error when Grafana tears the
			// stream down.
			if ctxErr := stream.Context().Err(); ctxErr != nil {
				return ctxErr
			}
			return err
		}
		parsedEvent := FromProto().StoredObjectEvent(ev)
		if err := a.handler.HandleStoredObjectEvent(stream.Context(), parsedEvent); err != nil {
			return err
		}
	}
}
