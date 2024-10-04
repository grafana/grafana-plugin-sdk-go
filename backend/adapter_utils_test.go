package backend

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/errorsource"

	"github.com/stretchr/testify/require"
)

func TestErrorWrapper(t *testing.T) {
	t.Run("No downstream error should not set downstream error source in context", func(t *testing.T) {
		ctx := errorsource.InitContext(context.Background())

		actualErr := errors.New("BOOM")
		wrapper := errorWrapper(func(_ context.Context) (RequestStatus, error) {
			return RequestStatusError, actualErr
		})
		status, err := wrapper(ctx)
		require.ErrorIs(t, err, actualErr)
		require.Equal(t, RequestStatusError, status)
		require.Equal(t, errorsource.DefaultErrorSource, errorsource.FromContext(ctx))
	})

	t.Run("Downstream error should set downstream error source in context", func(t *testing.T) {
		ctx := errorsource.InitContext(context.Background())

		actualErr := errors.New("BOOM")
		wrapper := errorWrapper(func(_ context.Context) (RequestStatus, error) {
			return RequestStatusError, errorsource.DownstreamError(actualErr)
		})
		status, err := wrapper(ctx)
		require.ErrorIs(t, err, actualErr)
		require.Equal(t, RequestStatusError, status)
		require.Equal(t, errorsource.ErrorSourceDownstream, errorsource.FromContext(ctx))
	})
}
