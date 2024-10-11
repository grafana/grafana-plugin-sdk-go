package status_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/errorsource"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

func TestSource(t *testing.T) {
	var s status.Source
	require.False(t, s.IsValid())
	require.Equal(t, "plugin", s.String())
	require.True(t, status.SourceDownstream.IsValid())
	require.Equal(t, "downstream", status.SourceDownstream.String())
	require.True(t, status.SourcePlugin.IsValid())
	require.Equal(t, "plugin", status.SourcePlugin.String())
}

func TestIsDownstreamError(t *testing.T) {
	tcs := []struct {
		name       string
		err        error
		expected   bool
		skipJoined bool
	}{
		{
			name:     "nil",
			err:      nil,
			expected: false,
		},
		{
			name:     "downstream error",
			err:      status.DownstreamError(nil),
			expected: true,
		},
		{
			name:     "timeout network error",
			err:      newFakeNetworkError(true, false),
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
			name:     "gRPC canceled error",
			err:      grpcstatus.Error(grpccodes.Canceled, "canceled"),
			expected: true,
		},
		{
			name:       "experimental Error with downstream source and status",
			err:        errorsource.New(errors.New("test"), backend.ErrorSourceDownstream, backend.StatusUnknown),
			skipJoined: true,
			expected:   true,
		},
		{
			name:       "experimental Error with plugin source and status",
			err:        errorsource.New(errors.New("test"), backend.ErrorSourcePlugin, backend.StatusUnknown),
			skipJoined: true,
			expected:   false,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			wrappedErr := fmt.Errorf("error: %w", tc.err)
			joinedErr := errors.Join(errors.New("oh no"), tc.err)
			assert.Equalf(t, tc.expected, status.IsDownstreamError(tc.err), "IsDownstreamHTTPError(%v)", tc.err)
			assert.Equalf(t, tc.expected, status.IsDownstreamError(wrappedErr), "wrapped IsDownstreamHTTPError(%v)", wrappedErr)

			if !tc.skipJoined {
				assert.Equalf(t, tc.expected, status.IsDownstreamError(joinedErr), "joined IsDownstreamHTTPError(%v)", joinedErr)
			}
		})
	}
}

func TestIsDownstreamHTTPError(t *testing.T) {
	tcs := []struct {
		name       string
		err        error
		expected   bool
		skipJoined bool
	}{
		{
			name:     "nil",
			err:      nil,
			expected: false,
		},
		{
			name:     "downstream error",
			err:      status.DownstreamError(nil),
			expected: true,
		},
		{
			name:     "timeout network error",
			err:      newFakeNetworkError(true, false),
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
			name:     "gRPC canceled error",
			err:      grpcstatus.Error(grpccodes.Canceled, "canceled"),
			expected: true,
		},
		{
			name:       "experimental Error with downstream source and status",
			err:        errorsource.New(errors.New("test"), backend.ErrorSourceDownstream, backend.StatusUnknown),
			skipJoined: true,
			expected:   true,
		},
		{
			name:       "experimental Error with plugin source and status",
			err:        errorsource.New(errors.New("test"), backend.ErrorSourcePlugin, backend.StatusUnknown),
			skipJoined: true,
			expected:   false,
		},
		{
			name:     "connection reset error",
			err:      &net.OpError{Err: &os.SyscallError{Err: syscall.ECONNREFUSED}},
			expected: true,
		},
		{
			name:     "connection refused error",
			err:      &net.OpError{Err: &os.SyscallError{Err: syscall.ECONNREFUSED}},
			expected: true,
		},
		{
			name:     "DNS not found error",
			err:      &net.DNSError{IsNotFound: true},
			expected: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			wrappedErr := fmt.Errorf("error: %w", tc.err)
			joinedErr := errors.Join(errors.New("oh no"), tc.err)
			assert.Equalf(t, tc.expected, status.IsDownstreamHTTPError(tc.err), "IsDownstreamHTTPError(%v)", tc.err)
			assert.Equalf(t, tc.expected, status.IsDownstreamHTTPError(wrappedErr), "wrapped IsDownstreamHTTPError(%v)", wrappedErr)

			if !tc.skipJoined {
				assert.Equalf(t, tc.expected, status.IsDownstreamHTTPError(joinedErr), "joined IsDownstreamHTTPError(%v)", joinedErr)
			}
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
