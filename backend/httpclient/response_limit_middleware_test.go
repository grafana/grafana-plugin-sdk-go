package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseLimitMiddleware(t *testing.T) {
	tcs := []struct {
		limit              int64
		expectedBodyLength int
		expectedBody       string
		err                error
	}{
		{limit: 1, expectedBodyLength: 1, expectedBody: "d", err: errors.New("error: http: response body too large, response limit is set to: 1")},
		{limit: 1000000, expectedBodyLength: 5, expectedBody: "dummy", err: nil},
		{limit: 0, expectedBodyLength: 5, expectedBody: "dummy", err: nil},
	}
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("Test ResponseLimitMiddleware with limit: %d", tc.limit), func(t *testing.T) {
			finalRoundTripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Request: req, Body: io.NopCloser(strings.NewReader("dummy"))}, nil
			})

			mw := ResponseLimitMiddleware(tc.limit)
			rt := mw.CreateMiddleware(Options{}, finalRoundTripper)
			require.NotNil(t, rt)
			middlewareName, ok := mw.(MiddlewareName)
			require.True(t, ok)
			require.Equal(t, ResponseLimitMiddlewareName, middlewareName.MiddlewareName())

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://test.com/query", nil)
			require.NoError(t, err)
			res, err := rt.RoundTrip(req)
			require.NoError(t, err)
			require.NotNil(t, res)
			require.NotNil(t, res.Body)

			bodyBytes, err := io.ReadAll(res.Body)
			if err != nil {
				require.EqualError(t, tc.err, err.Error())
			} else {
				require.NoError(t, tc.err)
			}
			require.NoError(t, res.Body.Close())

			require.Len(t, bodyBytes, tc.expectedBodyLength)
			require.Equal(t, string(bodyBytes), tc.expectedBody)
		})
	}
}
