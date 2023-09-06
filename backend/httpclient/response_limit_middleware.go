package httpclient

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ResponseLimitMiddlewareName is the middleware name used by ResponseLimitMiddleware.
const ResponseLimitMiddlewareName = "response-limit"

func ResponseLimitMiddleware(limit int64) Middleware {
	return NamedMiddlewareFunc(ResponseLimitMiddlewareName, func(opts Options, next http.RoundTripper) http.RoundTripper {
		if limit <= 0 {
			return next
		}
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			res, err := next.RoundTrip(req)
			if err != nil {
				return nil, err
			}

			if res != nil && res.StatusCode != http.StatusSwitchingProtocols {
				res.Body = MaxBytesReader(res.Body, limit)
			}

			return res, nil
		})
	})
}

// Similar implementation to http/net MaxBytesReader
// https://pkg.go.dev/net/http#MaxBytesReader
// What's happening differently here, is that the field that
// is limited is the response and not the request, thus
// the error handling/message needed to be accurate.

// ErrResponseBodyTooLarge indicates response body is too large
var ErrResponseBodyTooLarge = errors.New("http: response body too large")

// MaxBytesReader is similar to io.LimitReader but is intended for
// limiting the size of incoming request bodies. In contrast to
// io.LimitReader, MaxBytesReader's result is a ReadCloser, returns a
// non-EOF error for a Read beyond the limit, and closes the
// underlying reader when its Close method is called.
//
// MaxBytesReader prevents clients from accidentally or maliciously
// sending a large request and wasting server resources.
func MaxBytesReader(r io.ReadCloser, n int64) io.ReadCloser {
	return &maxBytesReader{r: r, n: n}
}

type maxBytesReader struct {
	r   io.ReadCloser // underlying reader
	n   int64         // max bytes remaining
	err error         // sticky error
}

func (l *maxBytesReader) Read(p []byte) (n int, err error) {
	if l.err != nil {
		return 0, l.err
	}
	if len(p) == 0 {
		return 0, nil
	}
	// If they asked for a 32KB byte read but only 5 bytes are
	// remaining, no need to read 32KB. 6 bytes will answer the
	// question of the whether we hit the limit or go past it.
	if int64(len(p)) > l.n+1 {
		p = p[:l.n+1]
	}
	n, err = l.r.Read(p)

	if int64(n) <= l.n {
		l.n -= int64(n)
		l.err = err
		return n, err
	}

	n = int(l.n)
	l.n = 0

	l.err = fmt.Errorf("error: %w, response limit is set to: %d", ErrResponseBodyTooLarge, n)
	return n, l.err
}

func (l *maxBytesReader) Close() error {
	return l.r.Close()
}
