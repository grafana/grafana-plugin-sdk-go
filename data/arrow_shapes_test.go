package data_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type arrowTestFrame struct {
	name string
	// sha256 of the frame's Arrow encoding. all_types equals testdata/all_types.golden.arrow.
	sha256 string
	frame  *data.Frame
}

// arrowTestFrames returns the frame shapes whose Arrow encoding is pinned by digest.
func arrowTestFrames() []arrowTestFrame {
	enumCfg := &data.FieldConfig{TypeConfig: &data.FieldTypeConfig{Enum: &data.EnumFieldConfig{
		Text:  []string{"low", "medium", "high"},
		Color: []string{"green", "yellow", "red"},
	}}}
	return []arrowTestFrame{
		{name: "all_types", sha256: "16fee727ac7b6960df80138da3889211bff26936c5156eb4531f7fa39af680b1", frame: goldenDF()},
		{name: "no_fields", sha256: "75b46f6298705c38694375457da27cfcb8945fcb1b845061eb653d95c8472d17", frame: data.NewFrame("empty")},
		{name: "zero_rows", sha256: "337ef6df96b8ac3196b5c5815ea54fa0065ccae295c9cf735b6f40947de13ae8", frame: data.NewFrame("zero_rows",
			data.NewField("time", nil, []time.Time{}),
			data.NewField("value", data.Labels{"a": "b"}, []float64{}),
			data.NewField("ns", nil, []*string{}),
			data.NewField("j", nil, []json.RawMessage{}),
			data.NewField("e", nil, []data.EnumItemIndex{}).SetConfig(enumCfg),
			data.NewField("nb", nil, []*bool{}),
		)},
		{name: "all_null", sha256: "39fdf5ddfde74da032ee98aeec140b120609ec0e532c73c6a9c9613692541097", frame: data.NewFrame("all_null",
			data.NewField("ns", nil, []*string{nil, nil, nil}),
			data.NewField("nj", nil, []*json.RawMessage{nil, nil, nil}),
			data.NewField("nf", nil, []*float64{nil, nil, nil}),
			data.NewField("nt", nil, []*time.Time{nil, nil, nil}),
			data.NewField("ne", nil, []*data.EnumItemIndex{nil, nil, nil}).SetConfig(enumCfg),
		)},
		{name: "enum", sha256: "71c8f125076c239d2ceae945c815e21663bc5ff81d4a295323e3be9d8c89a2ea", frame: data.NewFrame("enum",
			data.NewField("e", nil, []data.EnumItemIndex{0, 1, 2, 1}).SetConfig(enumCfg),
			data.NewField("ne", nil, []*data.EnumItemIndex{new(data.EnumItemIndex(2)), nil, new(data.EnumItemIndex(0)), nil}).SetConfig(enumCfg),
		)},
		{name: "nested_json", sha256: "29b1546c23acd346a08fe01d570fe7d7d8ab06e3cfe3c178f15d247c3f6d8d0c", frame: nestedJSONFrame()},
		{name: "wide_labels", sha256: "fd112926e66525bfe33c2d95f2e2e1552f1ec378eaba28a293c061f1f727b699", frame: timeSeriesFrame(1, 30)},
		{name: "range_240", sha256: "2edc8b085c4a88b1303f54749519fa027bed1ae3ed76030e1801f360dcac44e9", frame: timeSeriesFrame(240, 10)},
		{name: "big", sha256: "1464e95044a83deb907905dc1a21c751bca8fb5fad9b75ebab10a1831c5cdd24", frame: timeSeriesFrame(100_000, 1)},
	}
}

func nestedJSONFrame() *data.Frame {
	raw := json.RawMessage(`{"a":{"b":[1,2,{"c":"d"}]},"e":null}`)
	nullRaw := json.RawMessage(`null`)
	return data.NewFrame("nested_json",
		data.NewField("j", nil, []json.RawMessage{raw, json.RawMessage(`[]`), json.RawMessage(`""`)}),
		data.NewField("nj", nil, []*json.RawMessage{&raw, nil, &nullRaw}),
	)
}

// timeSeriesFrame is one series of a Prometheus-style response: a time field and a labeled value field.
func timeSeriesFrame(rows, labelCount int) *data.Frame {
	labels := make(data.Labels, labelCount)
	for i := range labelCount {
		labels[fmt.Sprintf("label_%02d", i)] = fmt.Sprintf("value_%02d_%s", i, strings.Repeat("x", 20))
	}
	times := make([]time.Time, rows)
	values := make([]float64, rows)
	for i := range rows {
		times[i] = time.Unix(1700000000+int64(i)*15, 0).UTC()
		values[i] = float64(i) / 7
	}
	f := data.NewFrame("",
		data.NewField("Time", nil, times),
		data.NewField("Value", labels, values).SetConfig(&data.FieldConfig{DisplayNameFromDS: "series"}),
	)
	f.RefID = "A"
	f.Meta = &data.FrameMeta{
		Type:                data.FrameTypeTimeSeriesMulti,
		TypeVersion:         data.FrameTypeVersion{0, 1},
		Custom:              map[string]any{"resultType": "matrix"},
		ExecutedQueryString: "Expr: up",
	}
	return f
}
