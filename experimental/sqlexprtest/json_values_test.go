package sqlexprtest

import (
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONNumbers(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		code Code
	}{
		{"positive overflow", `{"n":1e400}`, CodeJSONNumberOutOfRange},
		{"negative overflow", `{"n":-1e400}`, CodeJSONNumberOutOfRange},
		{"nested overflow", `{"outer":[1,{"n":1e400}]}`, CodeJSONNumberOutOfRange},
		{"huge positive exponent", `1e9999999999999999999999999999999`, CodeJSONNumberOutOfRange},
		{"large integer", `{"n":9007199254740993}`, CodeJSONIntegerAsFloat},
		{"negative integer", `-9007199254740993`, CodeJSONIntegerAsFloat},
		{"integer above float32 and int64", `123456789012345678901234567890`, CodeJSONIntegerAsFloat},
		{"zero exponent is a float", `9007199254740993e0`, ""},
		{"positive exponent is a float", `900719925474099.3e1`, ""},
		{"negative exponent is a float", `90071992547409930e-1`, ""},
		{"decimal integral value is a float", `9007199254740993.000`, ""},
		{"scientific constant", `6.02214076e23`, ""},
		{"large scientific value", `1e300`, ""},
		{"nested integer", `{"outer":[1,{"n":9007199254740993}]}`, CodeJSONIntegerAsFloat},
		{"integer limit", `9007199254740992`, ""},
		{"large exact even integer", `9007199254740994`, ""},
		{"exact large exponent", `1e20`, ""},
		{"inexact large exponent", `1e23`, ""},
		{"exact large integer", `100000000000000000000`, ""},
		{"fractional decimal", `0.1`, ""},
		{"large fractional decimal", `9007199254740993.1`, ""},
		{"underflow", `1e-9999999999999999999999999999999`, ""},
		{"zero with huge exponent", `0e9999999999999999999999999999999`, ""},
		{"number in string", `{"n":"9007199254740993"}`, ""},
		{"JSON null", `null`, ""},
		{"empty object", `{}`, ""},
		{"overwritten integer", `{"n":9007199254740993,"n":1}`, ""},
		{"overwritten overflow still fails", `{"n":1e400,"n":1}`, CodeJSONNumberOutOfRange},
		{"invalid syntax remains warning", `{"n":1e400`, CodeInvalidJSON},
	} {
		for _, nullable := range []bool{false, true} {
			t.Run(tc.name+nullableSuffix(nullable), func(t *testing.T) {
				raw := json.RawMessage(tc.raw)
				field := data.NewField("doc", nil, []json.RawMessage{raw})
				if nullable {
					field = data.NewField("doc", nil, []*json.RawMessage{&raw})
				}
				r := Check(data.Frames{typed(data.FrameTypeTable, data.NewFrame("", field))})
				assert.Equal(t, ModeTable, r.Mode)
				assert.Equal(t, 1, r.Rows)
				require.Len(t, r.Columns, 1)
				assert.Equal(t, "JSON", r.Columns[0].SQLType)
				if tc.code == "" {
					require.Empty(t, r.Findings)
					return
				}
				require.Len(t, r.Findings, 1)
				finding := r.Findings[0]
				assert.Equal(t, tc.code, finding.Code)
				assert.Equal(t, 0, finding.Frame)
				assert.Equal(t, "doc", finding.Field)
				severity, owner := SeverityWarning, OwnerGrafana
				if tc.code == CodeJSONNumberOutOfRange {
					severity = SeverityError
				}
				if tc.code == CodeInvalidJSON {
					owner = OwnerPlugin
				}
				assert.Equal(t, severity, finding.Severity)
				assert.Equal(t, owner, finding.Owner)
			})
		}
	}
}

func nullableSuffix(nullable bool) string {
	if nullable {
		return "/nullable"
	}
	return "/nonnullable"
}

func TestJSONFindingsInspectOnlyConsumedValues(t *testing.T) {
	for _, raw := range []json.RawMessage{json.RawMessage(`{"n":1e400}`), json.RawMessage(`{"n":9007199254740993}`)} {
		frame := typed(data.FrameTypeTable, data.NewFrame("",
			data.NewField("row_count", nil, []string{"one"}),
			data.NewField("doc", nil, []json.RawMessage{json.RawMessage(`{"n":1}`), raw}),
		))
		r := Check(data.Frames{frame})
		require.Equal(t, []Code{CodeUnequalFieldLengths}, codes(r))
	}
	for _, values := range []any{[]json.RawMessage{nil}, []*json.RawMessage{nil}} {
		r := Check(data.Frames{typed(data.FrameTypeTable, data.NewFrame("", data.NewField("doc", nil, values)))})
		require.Empty(t, r.Findings)
	}
}

func TestJSONOutOfRangeMessageCountsDocuments(t *testing.T) {
	frame := typed(data.FrameTypeTable, data.NewFrame("", data.NewField("doc", nil, []json.RawMessage{
		json.RawMessage(`[1e400,2e400]`),
		json.RawMessage(`{"n":3e400}`),
	})))
	r := Check(data.Frames{frame})
	require.Len(t, r.Findings, 1)
	assert.Contains(t, r.Findings[0].Message, "2 JSON values in field 0 contain a number outside float64")
}
