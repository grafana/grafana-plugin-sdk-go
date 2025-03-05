package status_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"syscall"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
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
			err:        backend.NewErrorWithSource(errors.New("test"), backend.ErrorSourceDownstream),
			skipJoined: true,
			expected:   true,
		},
		{
			name:       "experimental Error with plugin source and status",
			err:        backend.NewErrorWithSource(errors.New("test"), backend.ErrorSourcePlugin),
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

func TestIsPluginError(t *testing.T) {
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
			name:     "plugin error",
			err:      backend.NewErrorWithSource(nil, backend.ErrorSourcePlugin),
			expected: true,
		},
		{
			name:     "downstream error",
			err:      backend.NewErrorWithSource(nil, backend.ErrorSourceDownstream),
			expected: false,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("other error"),
			expected: false,
		},
		{
			name:     "network error",
			err:      newFakeNetworkError(true, true),
			expected: false,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			wrappedErr := fmt.Errorf("error: %w", tc.err)
			assert.Equalf(t, tc.expected, status.IsPluginError(tc.err), "IsPluginError(%v)", tc.err)
			assert.Equalf(t, tc.expected, status.IsPluginError(wrappedErr), "wrapped IsPluginError(%v)", wrappedErr)
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
			err:        backend.NewErrorWithSource(errors.New("test"), backend.ErrorSourceDownstream),
			skipJoined: true,
			expected:   true,
		},
		{
			name:       "experimental Error with plugin source and status",
			err:        backend.NewErrorWithSource(errors.New("test"), backend.ErrorSourcePlugin),
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
			name:     "host unreachable error",
			err:      &net.OpError{Err: &os.SyscallError{Err: syscall.EHOSTUNREACH}},
			expected: true,
		},
		{
			name:     "network unreachable error",
			err:      &net.OpError{Err: &os.SyscallError{Err: syscall.ENETUNREACH}},
			expected: true,
		},
		{
			name:     "DNS not found error",
			err:      &net.DNSError{IsNotFound: true},
			expected: true,
		},
		{
			name:     "wrapped *url.Error with UnknownAuthorityError",
			err:      &url.Error{Op: "Get", URL: "https://example.com", Err: &tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}}},
			expected: true,
		},
		{
			name:     "wrapped *url.Error with unrelated error",
			err:      &url.Error{Op: "Get", URL: "https://example.com", Err: fmt.Errorf("some unrelated error")},
			expected: false,
		},
		{
			name:     "direct CertificateInvalidError",
			err:      x509.CertificateInvalidError{Reason: x509.Expired, Cert: nil},
			expected: true,
		},
		{
			name:     "direct UnknownAuthorityError",
			err:      x509.UnknownAuthorityError{},
			expected: true,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "io.EOF error",
			err:      io.EOF,
			expected: false,
		},
		{
			name:     "url io.EOF error",
			err:      &url.Error{Op: "Get", URL: "https://example.com", Err: io.EOF},
			expected: true,
		},
		{
			name:     "net op io.EOF error",
			err:      &net.OpError{Err: io.EOF},
			expected: true,
		},
		{
			name:     "wrapped url io.EOF error",
			err:      fmt.Errorf("wrapped: %w", &url.Error{Op: "Get", URL: "https://example.com", Err: io.EOF}),
			expected: true,
		},
		{
			name:     "joined error with io.EOF",
			err:      errors.Join(io.EOF, &url.Error{Op: "Get", URL: "https://example.com", Err: io.EOF}),
			expected: true,
		},
		{
			name: "TLS hostname verification error",
			err: &url.Error{
				Op:  "Get",
				URL: "https://example.com",
				Err: &tls.CertificateVerificationError{
					Err: x509.HostnameError{
						Host:        "example.com",
						Certificate: &x509.Certificate{},
					},
				},
			},
			expected: true,
		},
		{
			name: "TLS certificate expired",
			err: &url.Error{
				Op:  "Get",
				URL: "https://example.com",
				Err: &tls.CertificateVerificationError{
					Err: x509.CertificateInvalidError{
						Reason: x509.Expired,
					},
				},
			},
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
