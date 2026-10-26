package grafanatest

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientRejectsInexactIntegers(t *testing.T) {
	for _, tc := range []struct{ kind, value string }{
		{"int8", "127.000000000000001"}, {"int8", "1e-400"}, {"int8", "1.5"}, {"int8", "128"},
		{"int32", "1e2"}, {"uint8", "-1"}, {"uint64", "18446744073709551616"}, {"int64", "9007199254740993.5"},
	} {
		for _, nullable := range []bool{false, true} {
			for _, position := range []int{0, 1} {
				t.Run(fmt.Sprintf("%s=%s/nullable=%t/column=%d", tc.kind, tc.value, nullable, position), func(t *testing.T) {
					wire := integerWire(tc.kind, tc.value, nullable, position)
					server := serveResult(t, http.StatusOK, []byte(wire))
					result, err := (&Client{URL: server.URL}).Query(t.Context(), testRequest())
					require.ErrorContains(t, err, "not valid typed frame JSON")
					require.Nil(t, result)
				})
			}
		}
	}
}

func TestClientAcceptsExactIntegers(t *testing.T) {
	for _, tc := range []struct{ kind, value string }{
		{"int8", "-128"}, {"int8", "127"}, {"uint8", "255"}, {"int64", "-9223372036854775808"},
		{"uint64", "18446744073709551615"}, {"int16", "null"},
	} {
		t.Run(tc.kind+"="+tc.value, func(t *testing.T) {
			server := serveResult(t, http.StatusOK, []byte(integerWire(tc.kind, tc.value, true, 1)))
			_, err := (&Client{URL: server.URL}).Query(t.Context(), testRequest())
			require.NoError(t, err)
		})
	}
}

func integerWire(kind, value string, nullable bool, position int) string {
	fields := fmt.Sprintf(`{"name":"n","type":"number","typeInfo":{"frame":%q,"nullable":%t}}`, kind, nullable)
	values := "[" + value + "]"
	if position > 0 {
		fields = `{"name":"prefix","type":"number","typeInfo":{"frame":"int64"}},` + fields
		values = "[0]," + values
	}
	return `{"results":{"B":{"frames":[{"schema":{"fields":[` + fields + `]},"data":{"values":[` + values + `]}}]}}}`
}
