package backend

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEndpoint(t *testing.T) {
	require.True(t, Endpoint("").IsEmpty())

	ctx := context.Background()

	t.Run("Empty context should return empty endpoint", func(t *testing.T) {
		epFromCtx := EndpointFromContext(ctx)
		require.Equal(t, Endpoint(""), epFromCtx)
	})

	t.Run("Endpoint in context should be returned", func(t *testing.T) {
		ep := Endpoint("someEndpoint")
		ctx = WithEndpoint(ctx, ep)
		epFromCtx := EndpointFromContext(ctx)
		require.Equal(t, ep, epFromCtx)
	})
}
