package slo_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/slo"
	"github.com/stretchr/testify/assert"
)

func TestAddDuration(t *testing.T) {
	//nolint:bodyclose
	next := MockRoundTripper{assert: assertDuration(t, 0)}
	fn := slo.RoundTripper(httpclient.Options{}, next)
	res, err := fn.RoundTrip(&http.Request{})
	assert.Equal(t, err, nil)

	err = res.Body.Close()
	assert.Equal(t, err, nil)
}

func TestAddDurationExists(t *testing.T) {
	duration := slo.NewDuration(1)
	//nolint:bodyclose
	next := MockRoundTripper{assert: assertDuration(t, duration.Value())}
	fn := slo.RoundTripper(httpclient.Options{}, next)

	req := &http.Request{}
	ctx := context.Background()
	ctx = context.WithValue(ctx, slo.DurationKey{}, duration)
	*req = *req.WithContext(ctx)

	res, err := fn.RoundTrip(req)
	assert.Equal(t, err, nil)

	err = res.Body.Close()
	assert.Equal(t, err, nil)

	assert.True(t, duration.Value() > 1)
}

type MockRoundTripper struct {
	assert func(*http.Request) (*http.Response, error)
}

func (m MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.assert(req)
}

func assertDuration(t *testing.T, want float64) func(req *http.Request) (*http.Response, error) {
	t.Helper()
	return func(req *http.Request) (*http.Response, error) {
		ctx := req.Context()
		val := ctx.Value(slo.DurationKey{})
		assert.NotNil(t, val)
		assert.Equal(t, want, val.(*slo.Duration).Value())

		res := &http.Response{Body: http.NoBody}
		return res, nil
	}
}
