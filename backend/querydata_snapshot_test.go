package backend_test

// TestScenarioGoldenBytes asserts byte-for-byte equivalence between the
// current SDK encode output and committed golden fixtures, for the same
// scenarios exercised by BenchmarkQueryDataResponseMacro.
//
// Fixtures live under testdata/querydata_scenarios/ and are named:
//
//	<scenario>.json                              — full QueryDataResponse JSON
//	<scenario>__<refID>__frameNNN.arrow          — Arrow IPC bytes per frame
//
// By default the test COMPARES. Pass -update to regenerate (reviews the diff
// before committing).
//
// The macro benchmark and this test draw inputs from macroScenarios() in
// querydata_scenarios_test.go, so a change that moves a benchmark number
// without drifting bytes is a perf win; a change that drifts bytes here is a
// format regression that needs review before merge.

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var updateGoldens = flag.Bool("update", false, "rewrite testdata/querydata_scenarios/ goldens (compare-only by default)")

const goldenDir = "testdata/querydata_scenarios"

func TestScenarioGoldenBytes(t *testing.T) {
	to := backend.ConvertToProtobuf{}

	if *updateGoldens {
		if err := os.MkdirAll(goldenDir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	for _, sc := range macroScenarios() {
		t.Run(sc.name, func(t *testing.T) {
			resp := sc.build()

			// Arrow: ConvertToProtobuf with ARROW format is the exact gRPC
			// payload; walk refIDs in sorted order because the protobuf map
			// iterates randomly.
			pResp, err := to.QueryDataResponse(backend.DataFrameFormat_ARROW, resp)
			if err != nil {
				t.Fatalf("ConvertToProtobuf: %v", err)
			}
			refIDs := make([]string, 0, len(pResp.Responses))
			for k := range pResp.Responses {
				refIDs = append(refIDs, k)
			}
			sort.Strings(refIDs)
			for _, ref := range refIDs {
				for i, fb := range pResp.Responses[ref].Frames {
					assertGoldenBytes(t, goldenPath(sc.name, ref, i), fb)
				}
			}

			// JSON: encode via the same jsoniter.Encoder Grafana uses,
			// capturing to a buffer so we can diff the bytes.
			var buf bytes.Buffer
			if err := jsonCfg.NewEncoder(&buf).Encode(resp); err != nil {
				t.Fatalf("jsoniter Encode: %v", err)
			}
			assertGoldenBytes(t, filepath.Join(goldenDir, sc.name+".json"), buf.Bytes())
		})
	}
}

func goldenPath(scenario, refID string, frameIdx int) string {
	return filepath.Join(goldenDir, scenario+"__"+refID+"__frame"+padFrameIdx(frameIdx)+".arrow")
}

func padFrameIdx(i int) string {
	s := []byte("000")
	// simple 3-digit pad, supports up to frame999
	for p := 2; p >= 0 && i > 0; p-- {
		s[p] = byte('0' + i%10)
		i /= 10
	}
	return string(s)
}

func assertGoldenBytes(t *testing.T, path string, got []byte) {
	t.Helper()
	if *updateGoldens {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path) // #nosec G304 -- golden path derived from test-local inputs
	if err != nil {
		t.Fatalf("read %s: %v (run with -update to generate)", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s: bytes differ from golden (len got=%d want=%d). Inspect diff or re-run with -update if the change is intentional.", path, len(got), len(want))
	}
}

// ---------- scenarios ----------


// jsonCfg matches Grafana's /api/ds/query encoder setup:
// pkg/api/response/response.go:163 — jsoniter.ConfigCompatibleWithStandardLibrary.
var jsonCfg = jsoniter.ConfigCompatibleWithStandardLibrary

// encodeJSONLikeGrafana streams the response to io.Discard via jsoniter's
// Encoder exactly as Grafana's response.StreamingResponse.WriteTo does at
// pkg/api/response/response.go:164.
func encodeJSONLikeGrafana(resp *backend.QueryDataResponse) error {
	return jsonCfg.NewEncoder(io.Discard).Encode(resp)
}

// ---------- frame factories ----------

func macroTimeSeriesFrame(refID string, rows int, labels data.Labels) *data.Frame {
	times := make([]time.Time, rows)
	values := make([]float64, rows)
	start := time.Unix(1700000000, 0)
	for i := 0; i < rows; i++ {
		times[i] = start.Add(time.Duration(i) * time.Second)
		values[i] = math.Sin(float64(i) * 0.01)
	}
	f := data.NewFrame("",
		data.NewField("time", nil, times),
		data.NewField("value", labels, values),
	)
	f.RefID = refID
	return f
}

func macroWideFrame(refID string, rows, cols int) *data.Frame {
	fields := make([]*data.Field, cols+1)
	times := make([]time.Time, rows)
	start := time.Unix(1700000000, 0)
	for i := 0; i < rows; i++ {
		times[i] = start.Add(time.Duration(i) * time.Second)
	}
	fields[0] = data.NewField("time", nil, times)
	for c := 0; c < cols; c++ {
		vs := make([]float64, rows)
		for r := 0; r < rows; r++ {
			vs[r] = float64(r*c) * 0.1
		}
		fields[c+1] = data.NewField("col_"+strconv.Itoa(c), nil, vs)
	}
	f := data.NewFrame("", fields...)
	f.RefID = refID
	return f
}

func macroStringHeavyFrame(refID string, rows int) *data.Frame {
	times := make([]time.Time, rows)
	levels := make([]string, rows)
	msgs := make([]string, rows)
	lines := make([]*string, rows)
	start := time.Unix(1700000000, 0)
	lvlOptions := []string{"info", "warn", "error", "debug"}
	for i := 0; i < rows; i++ {
		times[i] = start.Add(time.Duration(i) * time.Millisecond)
		levels[i] = lvlOptions[i%len(lvlOptions)]
		msgs[i] = "request handled id=" + strconv.Itoa(i) + " path=/api/v1/resource"
		if i%7 != 0 {
			s := `{"trace_id":"abc` + strconv.Itoa(i) + `","span":"root"}`
			lines[i] = &s
		}
	}
	f := data.NewFrame("",
		data.NewField("time", nil, times),
		data.NewField("level", nil, levels),
		data.NewField("message", nil, msgs),
		data.NewField("attrs", nil, lines),
	)
	f.RefID = refID
	return f
}

func macroMixedFrame(refID string, rows int) *data.Frame {
	times := make([]*time.Time, rows)
	f64 := make([]*float64, rows)
	i64 := make([]*int64, rows)
	strs := make([]*string, rows)
	bools := make([]*bool, rows)
	raws := make([]*json.RawMessage, rows)
	start := time.Unix(1700000000, 0)
	for i := 0; i < rows; i++ {
		if i%10 != 0 { // 10% nulls
			t := start.Add(time.Duration(i) * time.Second)
			times[i] = &t
			v := float64(i) * 1.5
			f64[i] = &v
			n := int64(i)
			i64[i] = &n
			s := "v_" + strconv.Itoa(i%100)
			strs[i] = &s
			b := i%2 == 0
			bools[i] = &b
			r := json.RawMessage(`{"id":` + strconv.Itoa(i%50) + `}`)
			raws[i] = &r
		}
	}
	f := data.NewFrame("",
		data.NewField("time", data.Labels{"source": "macro"}, times),
		data.NewField("f64", nil, f64),
		data.NewField("i64", nil, i64),
		data.NewField("string", data.Labels{"kind": "text"}, strs),
		data.NewField("bool", nil, bools),
		data.NewField("attrs", nil, raws),
	)
	f.RefID = refID
	f.SetMeta(&data.FrameMeta{ExecutedQueryString: "SELECT * FROM t"})
	return f
}

// macroFullMetadataFrame builds a small frame whose fields together exercise
// every FieldConfig serialization path (display name, units, min/max with an
// Inf edge case, decimals, interval, value mappings of all three kinds,
// thresholds with -Inf first step, color map, DataLinks with InternalDataLink,
// TypeConfig with EnumFieldConfig, Custom map). The frame itself carries a
// fully populated FrameMeta (notices, channel, preferred viz, custom, data
// topic). Byte-level regressions in any of these codecs will surface in the
// golden snapshot.
func macroFullMetadataFrame(refID string, rows int) *data.Frame {
	times := make([]time.Time, rows)
	vals := make([]float64, rows)
	states := make([]data.EnumItemIndex, rows)
	start := time.Unix(1700000000, 0)
	for i := 0; i < rows; i++ {
		times[i] = start.Add(time.Duration(i) * time.Second)
		vals[i] = float64(i) * 0.25
		states[i] = data.EnumItemIndex(i % 3)
	}

	timeField := data.NewField("time", nil, times)
	timeField.Config = &data.FieldConfig{
		DisplayName: "Time",
		Path:        "metrics/time",
		Description: "Sample timestamp (UTC)",
		Interval:    1000,
	}

	filterable := true
	writeable := false
	decimals := uint16(3)
	valueField := data.NewField("value", data.Labels{"unit": "qps", "host": "api-01"}, vals)
	valueField.Config = (&data.FieldConfig{
		DisplayName:       "QPS",
		DisplayNameFromDS: "query.qps",
		Path:              "metrics/value",
		Description:       "Requests per second",
		Filterable:        &filterable,
		Writeable:         &writeable,
		Unit:              "reqps",
		Decimals:          &decimals,
		Mappings: data.ValueMappings{
			data.ValueMapper{
				"0": data.ValueMappingResult{Text: "idle", Color: "gray", Index: 0},
				"1": data.ValueMappingResult{Text: "active", Color: "green", Index: 1},
			},
			data.RangeValueMapper{
				From:   confFloat(0),
				To:     confFloat(100),
				Result: data.ValueMappingResult{Text: "normal", Color: "green"},
			},
			data.SpecialValueMapper{
				Match:  data.SpecialValueNaN,
				Result: data.ValueMappingResult{Text: "N/A", Color: "red"},
			},
		},
		Thresholds: &data.ThresholdsConfig{
			Mode: data.ThresholdsModeAbsolute,
			Steps: []data.Threshold{
				{Value: data.ConfFloat64(math.Inf(-1)), Color: "green"},
				{Value: 50, Color: "yellow", State: "warn"},
				{Value: 90, Color: "red", State: "crit"},
			},
		},
		Color: map[string]interface{}{
			"mode":       "thresholds",
			"fixedColor": "blue",
		},
		Links: []data.DataLink{
			{
				Title:       "Open trace",
				TargetBlank: true,
				URL:         "https://example.test/trace/${__value.raw}",
			},
			{
				Title: "Drill-in",
				URL:   "/explore?left=${__value.time}",
				Internal: &data.InternalDataLink{
					DatasourceUID:  "ds-uid-1",
					DatasourceName: "loki-prod",
					Query:          map[string]any{"expr": `{app="api"}`, "refId": "A"},
					Transformations: &[]data.LinkTransformationConfig{
						{Type: data.Regex, Field: "message", Expression: `trace=(\w+)`, MapValue: "traceID"},
					},
					Range: &data.TimeRange{From: start, To: start.Add(time.Hour)},
				},
			},
		},
		NoValue: "n/a",
		Custom: map[string]interface{}{
			"axisPlacement": "left",
			"fillOpacity":   50,
			"nested":        map[string]any{"a": 1, "b": "two"},
		},
	}).SetMin(0).SetMax(math.Inf(+1))

	stateField := data.NewField("state", nil, states)
	stateField.Config = &data.FieldConfig{
		DisplayName: "State",
		TypeConfig: &data.FieldTypeConfig{
			Enum: &data.EnumFieldConfig{
				Text:        []string{"idle", "active", "error"},
				Color:       []string{"gray", "green", "red"},
				Icon:        []string{"pause", "play", "alert"},
				Description: []string{"No activity", "Serving traffic", "Failing"},
			},
		},
	}

	f := data.NewFrame("full-metadata", timeField, valueField, stateField)
	f.RefID = refID
	f.SetMeta(&data.FrameMeta{
		Type:                           data.FrameTypeTimeSeriesMulti,
		TypeVersion:                    data.FrameTypeVersion{0, 1},
		Channel:                        "ds/foo/bar",
		PreferredVisualization:         data.VisTypeGraph,
		PreferredVisualizationPluginID: "timeseries",
		ExecutedQueryString:            "SELECT time, value, state FROM t WHERE host='api-01'",
		DataTopic:                      data.DataTopicAnnotations,
		Notices: []data.Notice{
			{Severity: data.NoticeSeverityInfo, Text: "synthetic data", Link: "https://example.test/docs"},
			{Severity: data.NoticeSeverityWarning, Text: "approximate values"},
		},
		Custom: map[string]interface{}{
			"source":  "macro-scenario",
			"version": 7,
			"tags":    []string{"alpha", "beta"},
		},
	})
	return f
}

func confFloat(v float64) *data.ConfFloat64 {
	c := data.ConfFloat64(v)
	return &c
}

// ---------- dataplane shape factories ----------
//
// One factory per data plane frame type, shape-accurate per the contract at
// https://grafana.github.io/dataplane/contract/. Small row counts — coverage
// here is about structural shape, not volume (volume lives in the macro
// factories above). timeseries-many is deprecated (replaced by -multi) and is
// intentionally omitted.

func dataplaneTimeSeriesWide(refID string, rows int) *data.Frame {
	times := make([]time.Time, rows)
	a := make([]float64, rows)
	b := make([]float64, rows)
	start := time.Unix(1700000000, 0)
	for i := 0; i < rows; i++ {
		times[i] = start.Add(time.Duration(i) * time.Second)
		a[i] = float64(i) * 0.5
		b[i] = float64(i) * -0.25
	}
	f := data.NewFrame("",
		data.NewField("time", nil, times),
		data.NewField("value", data.Labels{"host": "a", "region": "us"}, a),
		data.NewField("value", data.Labels{"host": "b", "region": "us"}, b),
	)
	f.RefID = refID
	f.SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesWide, TypeVersion: data.FrameTypeVersion{0, 1}})
	return f
}

func dataplaneTimeSeriesLong(refID string, rows int) *data.Frame {
	// time repeats; string fields carry dimensions row-by-row
	times := make([]time.Time, rows)
	hosts := make([]string, rows)
	regions := make([]string, rows)
	values := make([]float64, rows)
	start := time.Unix(1700000000, 0)
	hostOpts := []string{"a", "b"}
	for i := 0; i < rows; i++ {
		// every other row: same timestamp, different host (long format pattern)
		times[i] = start.Add(time.Duration(i/2) * time.Second)
		hosts[i] = hostOpts[i%2]
		regions[i] = "us"
		values[i] = float64(i) * 0.1
	}
	f := data.NewFrame("",
		data.NewField("time", nil, times),
		data.NewField("host", nil, hosts),
		data.NewField("region", nil, regions),
		data.NewField("value", nil, values),
	)
	f.RefID = refID
	f.SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesLong, TypeVersion: data.FrameTypeVersion{0, 1}})
	return f
}

// dataplaneTimeSeriesMultiFrames returns n frames, each a single series —
// the multi format is a collection of single-series frames under one refID.
func dataplaneTimeSeriesMultiFrames(refID string, rows, n int) []*data.Frame {
	start := time.Unix(1700000000, 0)
	out := make([]*data.Frame, n)
	for k := 0; k < n; k++ {
		times := make([]time.Time, rows)
		vals := make([]float64, rows)
		for i := 0; i < rows; i++ {
			times[i] = start.Add(time.Duration(i) * time.Second)
			vals[i] = float64(i+k*1000) * 0.1
		}
		f := data.NewFrame("",
			data.NewField("time", nil, times),
			data.NewField("value", data.Labels{"series": "s" + strconv.Itoa(k)}, vals),
		)
		f.RefID = refID
		f.SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesMulti, TypeVersion: data.FrameTypeVersion{0, 1}})
		out[k] = f
	}
	return out
}

// dataplaneNumericWide: no time, labels on numeric fields carry dimensions.
// Row count is typically 1 (instant values).
func dataplaneNumericWide(refID string) *data.Frame {
	f := data.NewFrame("",
		data.NewField("value", data.Labels{"host": "a"}, []float64{1.5}),
		data.NewField("value", data.Labels{"host": "b"}, []float64{2.75}),
		data.NewField("count", data.Labels{"host": "a"}, []int64{42}),
		data.NewField("count", data.Labels{"host": "b"}, []int64{99}),
	)
	f.RefID = refID
	f.SetMeta(&data.FrameMeta{Type: data.FrameTypeNumericWide, TypeVersion: data.FrameTypeVersion{0, 1}})
	return f
}

func dataplaneNumericLong(refID string) *data.Frame {
	// one row per (host, region) combination
	hosts := []string{"a", "a", "b", "b"}
	regions := []string{"us", "eu", "us", "eu"}
	values := []float64{1.1, 2.2, 3.3, 4.4}
	counts := []int64{10, 20, 30, 40}
	f := data.NewFrame("",
		data.NewField("host", nil, hosts),
		data.NewField("region", nil, regions),
		data.NewField("value", nil, values),
		data.NewField("count", nil, counts),
	)
	f.RefID = refID
	f.SetMeta(&data.FrameMeta{Type: data.FrameTypeNumericLong, TypeVersion: data.FrameTypeVersion{0, 1}})
	return f
}

// dataplaneNumericMultiFrames: one frame per series, each with a single
// numeric value column (no time).
func dataplaneNumericMultiFrames(refID string, n int) []*data.Frame {
	out := make([]*data.Frame, n)
	for k := 0; k < n; k++ {
		f := data.NewFrame("",
			data.NewField("value", data.Labels{"series": "s" + strconv.Itoa(k)}, []float64{float64(k) * 1.25}),
		)
		f.RefID = refID
		f.SetMeta(&data.FrameMeta{Type: data.FrameTypeNumericMulti, TypeVersion: data.FrameTypeVersion{0, 1}})
		out[k] = f
	}
	return out
}

func dataplaneLogLines(refID string, rows int) *data.Frame {
	times := make([]time.Time, rows)
	bodies := make([]string, rows)
	severities := make([]string, rows)
	ids := make([]string, rows)
	start := time.Unix(1700000000, 0)
	sevs := []string{"info", "warning", "error", "debug"}
	for i := 0; i < rows; i++ {
		times[i] = start.Add(time.Duration(i) * time.Millisecond)
		bodies[i] = "event " + strconv.Itoa(i) + ": request served"
		severities[i] = sevs[i%len(sevs)]
		ids[i] = "id-" + strconv.Itoa(i)
	}
	f := data.NewFrame("",
		data.NewField("timestamp", nil, times),
		data.NewField("body", nil, bodies),
		data.NewField("severity", nil, severities),
		data.NewField("id", nil, ids),
	)
	f.RefID = refID
	f.SetMeta(&data.FrameMeta{Type: data.FrameTypeLogLines, TypeVersion: data.FrameTypeVersion{0, 1}})
	return f
}

// dataplaneTable — no structural constraints; tag-only coverage.
func dataplaneTable(refID string, rows int) *data.Frame {
	ids := make([]int64, rows)
	names := make([]string, rows)
	active := make([]bool, rows)
	for i := 0; i < rows; i++ {
		ids[i] = int64(i)
		names[i] = "row-" + strconv.Itoa(i)
		active[i] = i%2 == 0
	}
	f := data.NewFrame("",
		data.NewField("id", nil, ids),
		data.NewField("name", nil, names),
		data.NewField("active", nil, active),
	)
	f.RefID = refID
	f.SetMeta(&data.FrameMeta{Type: data.FrameTypeTable})
	return f
}

func dataplaneDirectoryListing(refID string) *data.Frame {
	f := data.NewFrame("",
		data.NewField("name", nil, []string{"logs", "metrics", "README.md"}),
		data.NewField("media-type", nil, []string{"directory", "directory", "text/markdown"}),
	)
	f.RefID = refID
	f.SetMeta(&data.FrameMeta{Type: data.FrameTypeDirectoryListing})
	return f
}

// ---------- response builders ----------

// singleRefResponse packs frames under a single refID (one panel, one query).
func singleRefResponse(refID string, frames ...*data.Frame) *backend.QueryDataResponse {
	return &backend.QueryDataResponse{
		Responses: backend.Responses{
			refID: backend.DataResponse{Frames: frames, Status: backend.StatusOK},
		},
	}
}

// ---------- scenarios ----------

type macroScenario struct {
	name string
	// build returns a fresh response per call so StopTimer/StartTimer can isolate
	// the measured stages from construction cost, and so the snapshot test can
	// re-materialize the input for each subtest.
	build func() *backend.QueryDataResponse
}

func macroScenarios() []macroScenario {
	return []macroScenario{
		{
			name: "TimeSeries_1Frame_1kRows",
			build: func() *backend.QueryDataResponse {
				return singleRefResponse("A", macroTimeSeriesFrame("A", 1000, data.Labels{"instance": "host-1"}))
			},
		},
		{
			name: "TimeSeries_10Frames_1kRows",
			build: func() *backend.QueryDataResponse {
				frames := make([]*data.Frame, 10)
				for i := range frames {
					frames[i] = macroTimeSeriesFrame("A", 1000, data.Labels{"instance": "host-" + strconv.Itoa(i)})
				}
				return singleRefResponse("A", frames...)
			},
		},
		{
			name: "Wide_1Frame_20cols_10kRows",
			build: func() *backend.QueryDataResponse {
				return singleRefResponse("A", macroWideFrame("A", 10000, 20))
			},
		},
		{
			name: "StringHeavy_1Frame_5kRows",
			build: func() *backend.QueryDataResponse {
				return singleRefResponse("A", macroStringHeavyFrame("A", 5000))
			},
		},
		{
			name: "Mixed_5Frames_multiRefID",
			build: func() *backend.QueryDataResponse {
				resp := &backend.QueryDataResponse{Responses: backend.Responses{}}
				for i, ref := range []string{"A", "B", "C", "D", "E"} {
					resp.Responses[ref] = backend.DataResponse{
						Frames: []*data.Frame{macroMixedFrame(ref, 1000+i*250)},
						Status: backend.StatusOK,
					}
				}
				return resp
			},
		},
		{
			name: "FullMetadata_1Frame_500Rows",
			build: func() *backend.QueryDataResponse {
				return singleRefResponse("A", macroFullMetadataFrame("A", 500))
			},
		},
		// Dataplane frame-type coverage — one scenario per known FrameType so
		// the golden snapshots lock down the contract shapes. Row counts are
		// small: coverage is structural, not scale.
		{
			name: "Dataplane_TimeSeriesWide",
			build: func() *backend.QueryDataResponse {
				return singleRefResponse("A", dataplaneTimeSeriesWide("A", 32))
			},
		},
		{
			name: "Dataplane_TimeSeriesLong",
			build: func() *backend.QueryDataResponse {
				return singleRefResponse("A", dataplaneTimeSeriesLong("A", 32))
			},
		},
		{
			name: "Dataplane_TimeSeriesMulti",
			build: func() *backend.QueryDataResponse {
				return singleRefResponse("A", dataplaneTimeSeriesMultiFrames("A", 32, 3)...)
			},
		},
		{
			name: "Dataplane_NumericWide",
			build: func() *backend.QueryDataResponse {
				return singleRefResponse("A", dataplaneNumericWide("A"))
			},
		},
		{
			name: "Dataplane_NumericLong",
			build: func() *backend.QueryDataResponse {
				return singleRefResponse("A", dataplaneNumericLong("A"))
			},
		},
		{
			name: "Dataplane_NumericMulti",
			build: func() *backend.QueryDataResponse {
				return singleRefResponse("A", dataplaneNumericMultiFrames("A", 3)...)
			},
		},
		{
			name: "Dataplane_LogLines",
			build: func() *backend.QueryDataResponse {
				return singleRefResponse("A", dataplaneLogLines("A", 64))
			},
		},
		{
			name: "Dataplane_Table",
			build: func() *backend.QueryDataResponse {
				return singleRefResponse("A", dataplaneTable("A", 16))
			},
		},
		{
			name: "Dataplane_DirectoryListing",
			build: func() *backend.QueryDataResponse {
				return singleRefResponse("A", dataplaneDirectoryListing("A"))
			},
		},
	}
}
