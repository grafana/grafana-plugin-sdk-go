package httpclient

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/errorsource"
	"github.com/stretchr/testify/assert"
)

func TestShouldErrorDownstream(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fail()
	}
	assert.NotNil(t, c)
	req := http.Request{
		URL: &url.URL{
			Scheme: "http",
			Host:   "localhost",
		},
		Header: http.Header{},
		Body:   io.NopCloser(&io.PipeReader{}),
	}
	defer req.Body.Close()
	_, err = c.Transport.RoundTrip(&req)
	assert.NotNil(t, err)

	var e errorsource.Error
	errors.As(err, &e)

	assert.Equal(t, backend.ErrorSourceDownstream, e.Source())
}
