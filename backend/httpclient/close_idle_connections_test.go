package httpclient

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClientCloseIdleConnections(t *testing.T) {
	tests := []struct {
		name      string
		newClient func() (*http.Client, error)
	}{
		{name: "New with default middlewares", newClient: func() (*http.Client, error) { return New() }},
		{name: "Provider.New with default middlewares", newClient: func() (*http.Client, error) { return NewProvider().New() }},
		{name: "Provider.New with ConfigureTransport", newClient: func() (*http.Client, error) {
			return NewProvider().New(Options{ConfigureTransport: func(_ Options, transport *http.Transport) {
				transport.MaxIdleConnsPerHost = 5
			}})
		}},
		{name: "New without middlewares", newClient: func() (*http.Client, error) {
			return New(Options{Middlewares: []Middleware{}})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			closed := make(chan struct{}, 1)
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
				if state == http.StateClosed {
					select {
					case closed <- struct{}{}:
					default:
					}
				}
			}
			srv.Start()
			t.Cleanup(srv.Close)

			client, err := tc.newClient()
			require.NoError(t, err)
			resp, err := client.Get(srv.URL)
			require.NoError(t, err)
			_, err = io.Copy(io.Discard, resp.Body)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())

			client.CloseIdleConnections()
			select {
			case <-closed:
			case <-time.After(5 * time.Second):
				t.Fatal("connection still open 5s after CloseIdleConnections")
			}
		})
	}
}
