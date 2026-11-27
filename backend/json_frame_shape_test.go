package backend

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

func numericFrame(rows int) *data.Frame {
	ts := make([]time.Time, rows)
	vs := make([]float64, rows)
	for i := range rows {
		ts[i] = time.Unix(1700000000+int64(i)*15, 0)
		vs[i] = float64(i) * 0.5
	}
	return data.NewFrame("", data.NewField("Time", nil, ts), data.NewField("Value", data.Labels{"job": "api"}, vs))
}

func stringFrame(rows, strLen int) *data.Frame {
	vs := make([]string, rows)
	for i := range vs {
		vs[i] = strings.Repeat("x", strLen)
	}
	return data.NewFrame("", data.NewField("Line", nil, vs))
}

func TestJSONSuitsFrames(t *testing.T) {
	s := "x"
	j := json.RawMessage(`"x"`)
	withField := func(field *data.Field) *data.Frame {
		f := numericFrame(1)
		f.Fields = append(f.Fields, field)
		return f
	}
	tests := []struct {
		name   string
		frames data.Frames
		want   bool
	}{
		{name: "no frames", want: true},
		{name: "empty frame", frames: data.Frames{data.NewFrame("")}, want: true},
		{name: "one row", frames: data.Frames{numericFrame(1)}, want: true},
		{name: "row limit", frames: data.Frames{numericFrame(jsonMaxFrameRows)}, want: true},
		{name: "over row limit", frames: data.Frames{numericFrame(jsonMaxFrameRows + 1)}, want: false},
		{name: "one long frame among short frames", frames: data.Frames{numericFrame(1), numericFrame(jsonMaxFrameRows + 1), numericFrame(1)}, want: false},
		{name: "int, bool and nullable float fields", frames: data.Frames{withField(data.NewField("i", nil, []int64{1})), withField(data.NewField("b", nil, []bool{true})), withField(data.NewField("f", nil, []*float64{nil}))}, want: true},
		{name: "string field", frames: data.Frames{withField(data.NewField("s", nil, []string{s}))}, want: false},
		{name: "nullable string field", frames: data.Frames{withField(data.NewField("s", nil, []*string{&s}))}, want: false},
		{name: "JSON field", frames: data.Frames{withField(data.NewField("j", nil, []json.RawMessage{j}))}, want: false},
		{name: "nullable JSON field", frames: data.Frames{withField(data.NewField("j", nil, []*json.RawMessage{&j}))}, want: false},
		{name: "string field in one frame of many", frames: data.Frames{numericFrame(1), stringFrame(1, 1)}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := jsonSuitsFrames(tc.frames); got != tc.want {
				t.Errorf("jsonSuitsFrames() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestQueryDataResponseFormatByFrameShape(t *testing.T) {
	tests := []struct {
		name      string
		requested DataFrameFormat
		responses map[string]data.Frames
		want      map[string]pluginv2.DataFrameFormat
	}{
		{
			name:      "JSON requested, short frames",
			requested: DataFrameFormat_JSON,
			responses: map[string]data.Frames{"A": {numericFrame(1), numericFrame(4)}},
			want:      map[string]pluginv2.DataFrameFormat{"A": pluginv2.DataFrameFormat_JSON},
		},
		{
			name:      "JSON requested, each response by its own frames",
			requested: DataFrameFormat_JSON,
			responses: map[string]data.Frames{"A": {numericFrame(1)}, "B": {numericFrame(1), numericFrame(240)}, "C": {stringFrame(1, 8)}},
			want:      map[string]pluginv2.DataFrameFormat{"A": pluginv2.DataFrameFormat_JSON, "B": pluginv2.DataFrameFormat_ARROW, "C": pluginv2.DataFrameFormat_ARROW},
		},
		{
			name:      "Arrow requested, short frames",
			requested: DataFrameFormat_ARROW,
			responses: map[string]data.Frames{"A": {numericFrame(1)}},
			want:      map[string]pluginv2.DataFrameFormat{"A": pluginv2.DataFrameFormat_ARROW},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := &QueryDataResponse{Responses: Responses{}}
			for refID, frames := range tc.responses {
				res.Responses[refID] = DataResponse{Frames: frames}
			}
			p, err := ToProto().QueryDataResponse(tc.requested, res)
			if err != nil {
				t.Fatalf("QueryDataResponse() error = %v", err)
			}
			got, err := FromProto().QueryDataResponse(p)
			if err != nil {
				t.Fatalf("FromProto().QueryDataResponse() error = %v", err)
			}
			for refID, want := range tc.want {
				if f := p.Responses[refID].Format; f != want {
					t.Errorf("response %s format = %v, want %v", refID, f, want)
				}
				if diff := cmp.Diff(tc.responses[refID], got.Responses[refID].Frames, data.FrameTestCompareOptions()...); diff != "" {
					t.Errorf("response %s frames differ after decode (-want +got):\n%s", refID, diff)
				}
			}
		})
	}
}

func TestJSONSuitsFramesAllocs(t *testing.T) {
	frames := data.Frames{numericFrame(jsonMaxFrameRows), stringFrame(1, 8)}
	if allocs := testing.AllocsPerRun(100, func() { jsonSuitsFrames(frames) }); allocs != 0 {
		t.Errorf("jsonSuitsFrames() allocs = %v, want 0", allocs)
	}
}

func TestQueryDataResponseJSONLongFrameAllocBytes(t *testing.T) {
	encode := func(format DataFrameFormat) uint64 {
		res := &QueryDataResponse{Responses: Responses{"A": {Frames: data.Frames{stringFrame(2000, 64)}}}}
		defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
		var m0, m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m0)
		if _, err := ToProto().QueryDataResponse(format, res); err != nil {
			t.Fatalf("QueryDataResponse() error = %v", err)
		}
		runtime.ReadMemStats(&m1)
		return m1.TotalAlloc - m0.TotalAlloc
	}
	arrow, jsonRequested := encode(DataFrameFormat_ARROW), encode(DataFrameFormat_JSON)
	if jsonRequested > arrow*11/10 {
		t.Errorf("JSON-requested encode of a long frame allocates %d bytes, want at most 1.1x the Arrow encode (%d bytes)", jsonRequested, arrow)
	}
}

var benchQueryDataResponse *pluginv2.QueryDataResponse

func BenchmarkQueryDataResponseByFrameShape(b *testing.B) {
	labeled := func(rows int) *data.Frame {
		f := numericFrame(rows)
		f.Fields[1].Labels = data.Labels{"__name__": "up", "instance": "10.0.0.1:9100", "job": "node", "region": "eu-west-1", "zone": "z1", "cluster": "c1", "namespace": "default", "pod": "node-exporter-abcde", "service": "node", "team": "infra"}
		return f
	}
	shapes := []struct {
		name   string
		frames int
		frame  func() *data.Frame
	}{
		{name: "2000x1_row", frames: 2000, frame: func() *data.Frame { return labeled(1) }},
		{name: "2000x16_rows", frames: 2000, frame: func() *data.Frame { return labeled(16) }},
		{name: "200x240_rows", frames: 200, frame: func() *data.Frame { return labeled(240) }},
		{name: "2000x1_row_string", frames: 2000, frame: func() *data.Frame { return stringFrame(1, 32) }},
	}
	for _, s := range shapes {
		frames := make(data.Frames, s.frames)
		for i := range frames {
			frames[i] = s.frame()
		}
		res := &QueryDataResponse{Responses: Responses{"A": {Frames: frames}}}
		for _, f := range []struct {
			name   string
			format DataFrameFormat
		}{{"arrow", DataFrameFormat_ARROW}, {"json", DataFrameFormat_JSON}} {
			b.Run(s.name+"/"+f.name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					p, err := ToProto().QueryDataResponse(f.format, res)
					if err != nil {
						b.Fatal(err)
					}
					benchQueryDataResponse = p
				}
			})
		}
	}
}
