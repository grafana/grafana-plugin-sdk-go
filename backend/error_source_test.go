package backend

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrorSource(t *testing.T) {
	var es ErrorSource
	require.False(t, es.IsValid())
	require.True(t, ErrorSourceDownstream.IsValid())
	require.True(t, ErrorSourcePlugin.IsValid())
}

func TestIsDownstreamError(t *testing.T) {
	tcs := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil",
			err:      nil,
			expected: false,
		},
		{
			name:     "downstream error",
			err:      DownstreamError(nil),
			expected: true,
		},
		{
			name:     "timeout network error",
			err:      newFakeNetworkError(true, false),
			expected: true,
		},
		{
			name:     "wrapped timeout network error",
			err:      fmt.Errorf("oh no. err %w", newFakeNetworkError(true, false)),
			expected: true,
		},
		{
			name:     "temporary timeout network error",
			err:      newFakeNetworkError(true, true),
			expected: true,
		},
		{
			name:     "non-timeout network error",
			err:      newFakeNetworkError(false, false),
			expected: false,
		},
		{
			name:     "os.ErrDeadlineExceeded",
			err:      os.ErrDeadlineExceeded,
			expected: true,
		},
		{
			name:     "os.ErrDeadlineExceeded",
			err:      fmt.Errorf("error: %w", os.ErrDeadlineExceeded),
			expected: true,
		},
		{
			name:     "wrapped os.ErrDeadlineExceeded",
			err:      errors.Join(fmt.Errorf("oh no"), os.ErrDeadlineExceeded),
			expected: true,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("other error"),
			expected: false,
		},
		{
			name:     "context.Canceled",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "wrapped context.Canceled",
			err:      fmt.Errorf("error: %w", context.Canceled),
			expected: true,
		},
		{
			name:     "joined context.Canceled",
			err:      errors.Join(fmt.Errorf("oh no"), context.Canceled),
			expected: true,
		},
		{
			name:     "gRPC canceled error",
			err:      status.Error(codes.Canceled, "canceled"),
			expected: true,
		},
		{
			name:     "wrapped gRPC canceled error",
			err:      fmt.Errorf("error: %w", status.Error(codes.Canceled, "canceled")),
			expected: true,
		},
		{
			name:     "joined gRPC canceled error",
			err:      errors.Join(fmt.Errorf("oh no"), status.Error(codes.Canceled, "canceled")),
			expected: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equalf(t, tc.expected, IsDownstreamError(tc.err), "IsDownstreamError(%v)", tc.err)
		})
	}
}

var _ net.Error = &fakeNetworkError{}

type fakeNetworkError struct {
	timeout   bool
	temporary bool
}

func newFakeNetworkError(timeout, temporary bool) *fakeNetworkError {
	return &fakeNetworkError{
		timeout:   timeout,
		temporary: temporary,
	}
}

func (d *fakeNetworkError) Error() string {
	return "dummy timeout error"
}

func (d *fakeNetworkError) Timeout() bool {
	return d.timeout
}

func (d *fakeNetworkError) Temporary() bool {
	return d.temporary
}
