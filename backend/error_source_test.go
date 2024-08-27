package backend

import (
	"errors"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			name:     "wrapped os.ErrDeadlineExceeded",
			err:      errors.Join(os.ErrDeadlineExceeded),
			expected: true,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("other error"),
			expected: false,
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
