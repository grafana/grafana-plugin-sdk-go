package log

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextualLogger(t *testing.T) {
	t.Run("FromContext", func(t *testing.T) {
		logger := New()
		ctx := WithContext(context.Background(), logger)
		ctxLogger := FromContext(ctx)
		require.Equal(t, logger, ctxLogger)
		require.NotEqual(t, DefaultLogger, ctxLogger)
	})

	t.Run("FromContext on empty context should return default logger", func(t *testing.T) {
		ctxLogger := FromContext(context.Background())
		require.Equal(t, DefaultLogger, ctxLogger)
	})
}
