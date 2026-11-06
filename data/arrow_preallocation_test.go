package data_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestArrowVariableWidthColumns(t *testing.T) {
	for _, rows := range []int{0, 1, 65, 1025} {
		t.Run(fmt.Sprint(rows), func(t *testing.T) {
			strings := make([]string, rows)
			nullableStrings := make([]*string, rows)
			messages := make([]json.RawMessage, rows)
			nullableMessages := make([]*json.RawMessage, rows)
			for i := range rows {
				strings[i] = fmt.Sprintf("service-%d-❤️", i)
				messages[i] = json.RawMessage(fmt.Sprintf(`{"id":%d}`, i))
				if i%3 == 0 {
					strings[i] = ""
					messages[i] = json.RawMessage(`null`)
				} else {
					nullableStrings[i] = &strings[i]
					nullableMessages[i] = &messages[i]
				}
			}
			frame := data.NewFrame("variable-width",
				data.NewField("strings", nil, strings),
				data.NewField("nullable strings", nil, nullableStrings),
				data.NewField("json", nil, messages),
				data.NewField("nullable json", nil, nullableMessages),
			)
			encoded, err := frame.MarshalArrow()
			require.NoError(t, err)
			decoded, err := data.UnmarshalArrowFrame(encoded)
			require.NoError(t, err)
			require.Empty(t, cmp.Diff(frame, decoded, data.FrameTestCompareOptions()...))
		})
	}
}

func BenchmarkMarshalArrowColumns(b *testing.B) {
	for _, rows := range []int{1, 1000, 10000} {
		timestamps := make([]time.Time, rows)
		values := make([]float64, rows)
		strings := make([]string, rows)
		messages := make([]json.RawMessage, rows)
		for i := range rows {
			timestamps[i] = time.Unix(1700000000+int64(i), 0)
			values[i] = float64(i) / 3
			strings[i] = fmt.Sprintf("service-%d-❤️", i)
			messages[i] = json.RawMessage(fmt.Sprintf(`{"service":"api","span":%d}`, i))
		}
		for name, frame := range map[string]*data.Frame{
			"numeric":      data.NewFrame("series", data.NewField("Time", nil, timestamps), data.NewField("Value", data.Labels{"service": "api"}, values)),
			"strings-json": data.NewFrame("spans", data.NewField("Service", nil, strings), data.NewField("Tags", nil, messages)),
		} {
			b.Run(fmt.Sprintf("%s/rows=%d", name, rows), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, err := frame.MarshalArrow(); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
