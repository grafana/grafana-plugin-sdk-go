package grafanatest

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/stretchr/testify/require"
)

func TestClientRejectsWrongValues(t *testing.T) {
	for _, tc := range []struct{ kind, value string }{
		{"int8", "127.000000000000001"}, {"int8", "1e-400"}, {"int8", "1.5"}, {"int8", "128"},
		{"int32", "1e2"}, {"uint8", "-1"}, {"uint64", "18446744073709551616"}, {"int64", "9007199254740993.5"},
		{"int8", `"257"`}, {"uint8", `"-1"`}, {"int64", `"1"`}, {"enum", "65536"}, {"time", "1.5"}, {"time", `"1000"`},
		{"string", "123"}, {"string", "true"}, {"bool", "1"}, {"bool", `"true"`},
		{"float32", "1e39"}, {"float32", `"1"`}, {"float64", "1e400"}, {"float64", `"NaN"`},
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
		{"uint64", "18446744073709551615"}, {"int16", "null"}, {"enum", "65535"}, {"string", `"123"`},
		{"bool", "false"}, {"float32", "3.4028234e38"}, {"float64", "1e-400"}, {"json", `"123"`},
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

func TestClientKeepsTimesOutsideNanosecondRange(t *testing.T) {
	for _, want := range []time.Time{
		time.Date(2500, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2500, 1, 1, 0, 0, 0, 123456789, time.UTC),
		time.Date(1600, 6, 1, 12, 0, 0, 5, time.UTC),
		time.Date(2026, 1, 2, 3, 4, 5, 6, time.UTC),
	} {
		for _, nullable := range []bool{false, true} {
			for _, position := range []int{0, 1} {
				t.Run(fmt.Sprintf("%s/nullable=%t/column=%d", want.Format(time.RFC3339Nano), nullable, position), func(t *testing.T) {
					field := data.NewField("event_time", nil, []time.Time{want})
					if nullable {
						field = data.NewField("event_time", nil, []*time.Time{&want, nil})
					}
					fields := []*data.Field{}
					if position > 0 {
						prefix := []string{"x"}
						if nullable {
							prefix = append(prefix, "y")
						}
						fields = append(fields, data.NewField("prefix", nil, prefix))
					}
					frame := data.NewFrame("", append(fields, field)...)
					server := serveResult(t, http.StatusOK, responseJSON(t, map[string]QueryResult{"B": {Frames: data.Frames{frame}}}))
					result, err := (&Client{URL: server.URL}).Query(t.Context(), testRequest())
					require.NoError(t, err)
					got, ok := result.Results["B"].Frames[0].Fields[position].ConcreteAt(0)
					require.True(t, ok)
					require.True(t, want.Equal(got.(time.Time)), "want %s, got %s", want, got)
					if nullable {
						_, ok := result.Results["B"].Frames[0].Fields[position].ConcreteAt(1)
						require.False(t, ok)
					}
					require.Empty(t, Check(result, map[string]Expectation{"B": {Frames: data.Frames{frame}}}))
				})
			}
		}
	}
}
