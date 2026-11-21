package httpclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/status"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	errDialTimeout = &net.OpError{Op: "dial", Net: "tcp", Err: os.ErrDeadlineExceeded}
	errDialRefused = &net.OpError{Op: "dial", Net: "tcp", Err: &os.SyscallError{Syscall: "connect", Err: syscall.ECONNREFUSED}}
	errReadTimeout = &net.OpError{Op: "read", Net: "tcp", Err: os.ErrDeadlineExceeded}
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// scriptedTransport returns the next error from its script, or a response when the error is nil.
type scriptedTransport struct {
	mu    sync.Mutex
	calls int
	errs  []error
}

func (s *scriptedTransport) RoundTrip(*http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.errs[s.calls]
	s.calls++
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func newTestBreaker(t *testing.T, failures int, next http.RoundTripper) (http.RoundTripper, *fakeClock) {
	t.Helper()
	clock := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	return newDialBreaker(DialBreakerOptions{Failures: failures}, clock.now).roundTripper(next), clock
}

func roundTrip(rt http.RoundTripper, req *http.Request) error {
	res, err := rt.RoundTrip(req)
	if res != nil && res.Body != nil {
		_ = res.Body.Close()
	}
	return err
}

func doRequest(t *testing.T, rt http.RoundTripper) error {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "http://upstream.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}
	return roundTrip(rt, req)
}

func repeat(err error, n int) []error {
	errs := make([]error, n)
	for i := range errs {
		errs[i] = err
	}
	return errs
}

func TestDialBreakerOpensAfterConsecutiveDialTimeouts(t *testing.T) {
	next := &scriptedTransport{errs: repeat(errDialTimeout, 3)}
	rt, _ := newTestBreaker(t, 3, next)
	for range 3 {
		if err := doRequest(t, rt); !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("doRequest() = %v, want the dial timeout", err)
		}
	}

	err := doRequest(t, rt)
	if !errors.Is(err, ErrDialBreakerOpen) {
		t.Fatalf("doRequest() after 3 dial timeouts = %v, want ErrDialBreakerOpen", err)
	}
	if !status.IsDownstreamError(err) {
		t.Errorf("breaker error %v is not a downstream error", err)
	}
	if !strings.Contains(err.Error(), "next dial attempt in 10s") {
		t.Errorf("breaker error %q does not name the wait", err)
	}
	if next.calls != 3 {
		t.Errorf("transport got %d calls, want 3", next.calls)
	}
}

func TestDialBreakerDoesNotOpen(t *testing.T) {
	tests := []struct {
		name string
		errs []error
	}{
		{name: "refused dials", errs: repeat(errDialRefused, 5)},
		{name: "timeouts after the dial", errs: repeat(errReadTimeout, 5)},
		{name: "success between timeouts", errs: []error{errDialTimeout, errDialTimeout, nil, errDialTimeout, errDialTimeout, nil}},
		{name: "refusal between timeouts", errs: []error{errDialTimeout, errDialTimeout, errDialRefused, errDialTimeout, errDialTimeout, nil}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			next := &scriptedTransport{errs: tc.errs}
			rt, _ := newTestBreaker(t, 3, next)
			for i := range tc.errs {
				if err := doRequest(t, rt); errors.Is(err, ErrDialBreakerOpen) {
					t.Fatalf("request %d: breaker opened: %v", i, err)
				}
			}
			if next.calls != len(tc.errs) {
				t.Errorf("transport got %d calls, want %d", next.calls, len(tc.errs))
			}
		})
	}
}

func TestDialBreakerIgnoresCancelledRequests(t *testing.T) {
	next := &scriptedTransport{errs: repeat(errDialTimeout, 5)}
	rt, _ := newTestBreaker(t, 3, next)
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("client went away"))
	for i := range 5 {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://upstream.invalid", nil)
		if err := roundTrip(rt, req); errors.Is(err, ErrDialBreakerOpen) {
			t.Fatalf("request %d: a cancelled request opened the breaker", i)
		}
	}
}

func TestDialBreakerIgnoresTimeoutsThatStartedBeforeASuccess(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{}, 3)
	var mu sync.Mutex
	slow := 0
	next := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("X-Slow") != "" {
			mu.Lock()
			slow++
			mu.Unlock()
			started <- struct{}{}
			<-release
			return nil, errDialTimeout
		}
		return &http.Response{StatusCode: http.StatusOK}, nil
	})
	rt, clock := newTestBreaker(t, 3, next)

	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() {
			req, _ := http.NewRequest(http.MethodGet, "http://upstream.invalid", nil)
			req.Header.Set("X-Slow", "1")
			_ = roundTrip(rt, req)
		})
	}
	for range 3 {
		<-started
	}
	clock.advance(time.Second)
	if err := doRequest(t, rt); err != nil {
		t.Fatalf("doRequest() = %v, want success", err)
	}
	close(release)
	wg.Wait()

	if err := doRequest(t, rt); err != nil {
		t.Fatalf("doRequest() after 3 late timeouts = %v, want success: timeouts that started before the success must not count", err)
	}
}

func TestDialBreakerHalfOpen(t *testing.T) {
	t.Run("one probe, and a successful probe closes", func(t *testing.T) {
		probeRelease := make(chan struct{})
		probeStarted := make(chan struct{})
		calls := 0
		var mu sync.Mutex
		next := RoundTripperFunc(func(*http.Request) (*http.Response, error) {
			mu.Lock()
			calls++
			n := calls
			mu.Unlock()
			if n <= 2 {
				return nil, errDialTimeout
			}
			if n == 3 {
				close(probeStarted)
				<-probeRelease
			}
			return &http.Response{StatusCode: http.StatusOK}, nil
		})
		rt, clock := newTestBreaker(t, 2, next)
		_ = doRequest(t, rt)
		_ = doRequest(t, rt)
		clock.advance(10 * time.Second)

		probeErr := make(chan error, 1)
		go func() { probeErr <- doRequest(t, rt) }()
		<-probeStarted
		if err := doRequest(t, rt); !errors.Is(err, ErrDialBreakerOpen) {
			t.Fatalf("doRequest() during the probe = %v, want ErrDialBreakerOpen", err)
		}
		close(probeRelease)
		if err := <-probeErr; err != nil {
			t.Fatalf("probe = %v, want success", err)
		}
		if err := doRequest(t, rt); err != nil {
			t.Fatalf("doRequest() after a successful probe = %v, want success", err)
		}
	})

	t.Run("a failed probe opens the breaker again", func(t *testing.T) {
		next := &scriptedTransport{errs: repeat(errDialTimeout, 3)}
		rt, clock := newTestBreaker(t, 2, next)
		_ = doRequest(t, rt)
		_ = doRequest(t, rt)
		clock.advance(10 * time.Second)
		if err := doRequest(t, rt); !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("probe = %v, want the dial timeout", err)
		}
		clock.advance(9 * time.Second)
		if err := doRequest(t, rt); !errors.Is(err, ErrDialBreakerOpen) {
			t.Fatalf("doRequest() 9s after a failed probe = %v, want ErrDialBreakerOpen", err)
		}
		if next.calls != 3 {
			t.Errorf("transport got %d calls, want 3", next.calls)
		}
	})

	t.Run("a cancelled probe lets the next request probe", func(t *testing.T) {
		next := &scriptedTransport{errs: []error{errDialTimeout, errDialTimeout, context.Canceled, nil}}
		rt, clock := newTestBreaker(t, 2, next)
		_ = doRequest(t, rt)
		_ = doRequest(t, rt)
		clock.advance(10 * time.Second)
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(errors.New("client went away"))
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://upstream.invalid", nil)
		_ = roundTrip(rt, req)
		if err := doRequest(t, rt); err != nil {
			t.Fatalf("doRequest() after a cancelled probe = %v, want the next probe to go through", err)
		}
	})
}

func TestDialBreakerMiddleware(t *testing.T) {
	t.Run("disabled when Failures is 0", func(t *testing.T) {
		next := &scriptedTransport{errs: repeat(errDialTimeout, 10)}
		rt := DialBreakerMiddleware(DialBreakerOptions{}).CreateMiddleware(Options{}, next)
		for i := range 10 {
			if err := doRequest(t, rt); errors.Is(err, ErrDialBreakerOpen) {
				t.Fatalf("request %d: a disabled breaker opened", i)
			}
		}
	})

	t.Run("OpenFor 0 means 10s", func(t *testing.T) {
		b := newDialBreaker(DialBreakerOptions{Failures: 1}, time.Now)
		if b.openFor != 10*time.Second {
			t.Errorf("openFor = %s, want 10s", b.openFor)
		}
	})

	t.Run("name", func(t *testing.T) {
		mw, ok := DialBreakerMiddleware(DialBreakerOptions{Failures: 1}).(MiddlewareName)
		if !ok || mw.MiddlewareName() != DialBreakerMiddlewareName {
			t.Errorf("middleware name is not %q", DialBreakerMiddlewareName)
		}
	})

	t.Run("each client has its own state", func(t *testing.T) {
		mw := DialBreakerMiddleware(DialBreakerOptions{Failures: 1})
		a := mw.CreateMiddleware(Options{}, &scriptedTransport{errs: []error{errDialTimeout}})
		b := mw.CreateMiddleware(Options{}, &scriptedTransport{errs: []error{nil}})
		_ = doRequest(t, a)
		if err := doRequest(t, b); err != nil {
			t.Errorf("second client = %v, want success", err)
		}
	})
}

func transitionCount(t *testing.T, transition string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, mf := range families {
		if mf.GetName() != "plugins_datasource_dial_breaker_transitions_total" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "transition" && l.GetValue() == transition {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}

func TestDialBreakerTransitionsCounter(t *testing.T) {
	transitions := []string{"open", "probe", "close"}
	before := map[string]float64{}
	for _, tr := range transitions {
		before[tr] = transitionCount(t, tr)
	}

	next := &scriptedTransport{errs: []error{errDialTimeout, nil}}
	rt, clock := newTestBreaker(t, 1, next)
	_ = doRequest(t, rt)
	clock.advance(10 * time.Second)
	_ = doRequest(t, rt)

	for _, tr := range transitions {
		if got := transitionCount(t, tr) - before[tr]; got != 1 {
			t.Errorf("%s transitions = %v, want 1", tr, got)
		}
	}
}

func BenchmarkDialBreakerClosed(b *testing.B) {
	ok := RoundTripperFunc(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: http.StatusOK}, nil })
	req, err := http.NewRequest(http.MethodGet, "http://upstream.invalid", nil)
	if err != nil {
		b.Fatal(err)
	}
	for _, bc := range []struct {
		name string
		rt   http.RoundTripper
	}{
		{"without", ok},
		{"with", DialBreakerMiddleware(DialBreakerOptions{Failures: 5}).CreateMiddleware(Options{}, ok)},
	} {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := roundTrip(bc.rt, req); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
