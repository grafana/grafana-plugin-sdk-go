package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/status"
)

// DialBreakerMiddlewareName is the middleware name used by DialBreakerMiddleware.
const DialBreakerMiddlewareName = "DialBreaker"

// DefaultDialBreakerOpenFor is the open window that DialBreakerMiddleware uses when DialBreakerOptions.OpenFor is 0.
const DefaultDialBreakerOpenFor = 10 * time.Second

// ErrDialBreakerOpen is the error that DialBreakerMiddleware returns, as a downstream error, while it fails requests without a dial.
var ErrDialBreakerOpen = errors.New("upstream unreachable: dial failed repeatedly")

var dialBreakerTransitions = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "plugins",
	Name:      "datasource_dial_breaker_transitions_total",
	Help:      "Number of dial breaker state changes, by transition: open, probe or close.",
}, []string{"transition"})

// DialBreakerOptions configures DialBreakerMiddleware.
type DialBreakerOptions struct {
	// Failures is the number of dial timeouts in a row that opens the breaker. 0 or less disables the breaker.
	Failures int
	// OpenFor is how long an open breaker fails requests before it lets one probe through. 0 means DefaultDialBreakerOpenFor.
	OpenFor time.Duration
}

// DialBreakerMiddleware fails requests at once after Failures dial timeouts in a row, then lets one probe through after OpenFor.
// Each client gets its own breaker. Only dial timeouts count, and any other result of a request that was not cancelled closes the breaker.
func DialBreakerMiddleware(opts DialBreakerOptions) Middleware {
	return NamedMiddlewareFunc(DialBreakerMiddlewareName, func(_ Options, next http.RoundTripper) http.RoundTripper {
		if opts.Failures <= 0 {
			return next
		}
		return newDialBreaker(opts, time.Now).roundTripper(next)
	})
}

type dialBreaker struct {
	failures int
	openFor  time.Duration
	now      func() time.Time

	mu          sync.Mutex
	consecutive int
	openUntil   time.Time
	probing     bool
	lastSuccess time.Time
}

func newDialBreaker(opts DialBreakerOptions, now func() time.Time) *dialBreaker {
	openFor := opts.OpenFor
	if openFor <= 0 {
		openFor = DefaultDialBreakerOpenFor
	}
	return &dialBreaker{failures: opts.Failures, openFor: openFor, now: now}
}

func (b *dialBreaker) roundTripper(next http.RoundTripper) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		start, probe, err := b.admit(req.Context())
		if err != nil {
			return nil, err
		}
		res, err := next.RoundTrip(req)
		b.record(req.Context(), start, probe, err)
		return res, err
	})
}

// admit returns the start time of an admitted request and whether it is the probe, or the error for a request it fails.
func (b *dialBreaker) admit(ctx context.Context) (time.Time, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	if b.openUntil.IsZero() {
		return now, false, nil
	}
	if wait := b.openUntil.Sub(now); wait > 0 {
		return now, false, status.DownstreamError(fmt.Errorf("%w, next dial attempt in %s", ErrDialBreakerOpen, wait.Round(100*time.Millisecond)))
	}
	if b.probing {
		return now, false, status.DownstreamError(fmt.Errorf("%w, a dial attempt is in progress", ErrDialBreakerOpen))
	}
	b.probing = true
	dialBreakerTransitions.WithLabelValues("probe").Inc()
	log.DefaultLogger.FromContext(ctx).Info("dial breaker lets one probe through")
	return now, true, nil
}

func (b *dialBreaker) record(ctx context.Context, start time.Time, probe bool, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if probe {
		b.probing = false
	}
	if err != nil && ctx.Err() != nil {
		return
	}
	now := b.now()
	if !isDialTimeout(err) {
		if !b.openUntil.IsZero() {
			dialBreakerTransitions.WithLabelValues("close").Inc()
			log.DefaultLogger.FromContext(ctx).Info("dial breaker closed")
		}
		b.consecutive = 0
		b.openUntil = time.Time{}
		b.lastSuccess = now
		return
	}
	if probe {
		b.openUntil = now.Add(b.openFor)
		dialBreakerTransitions.WithLabelValues("open").Inc()
		log.DefaultLogger.FromContext(ctx).Warn("dial breaker opened again after a failed probe", "openFor", b.openFor)
		return
	}
	if !start.After(b.lastSuccess) {
		return
	}
	b.consecutive++
	if b.openUntil.IsZero() && b.consecutive >= b.failures {
		b.openUntil = now.Add(b.openFor)
		dialBreakerTransitions.WithLabelValues("open").Inc()
		log.DefaultLogger.FromContext(ctx).Warn("dial breaker opened", "dialTimeouts", b.consecutive, "openFor", b.openFor)
	}
}

func isDialTimeout(err error) bool {
	opErr, ok := errors.AsType[*net.OpError](err)
	return ok && opErr.Op == "dial" && opErr.Timeout()
}
