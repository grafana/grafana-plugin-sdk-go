package data_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var update = flag.Bool("update", true, "update .golden.arrow files")

const maxEcma6Int = 1<<53 - 1
const minEcma6Int = -maxEcma6Int

func goldenDF() *data.Frame {
	nullableStringValuesFieldConfig := (&data.FieldConfig{
		DisplayName: "Grafana ❤️ (Previous should be heart emoji) 🦥 (Previous should be sloth emoji)",
		Links: []data.DataLink{
			{
				Title:       "Donate - The Sloth Conservation Foundation",
				TargetBlank: true,
				URL:         "https://slothconservation.com/how-to-help/donate/",
			},
		},
		NoValue: "😤",
		// math.NaN() and math.Infs become null when encoded to json
	}).SetDecimals(2).SetMax(math.Inf(1)).SetMin(math.NaN()).SetFilterable(false)

	df := data.NewFrame("many_types",
		data.NewField("string_values", data.Labels{"aLabelKey": "aLabelValue"}, []string{
			"Go Min",
			"JS Min (for >= 64)",
			"0 / nil / misc",
			"JS Max (for >= 64)",
			"Go Max",
		}).SetConfig(&data.FieldConfig{}),
		data.NewField("nullable_string_values", data.Labels{"aLabelKey": "aLabelValue", "bLabelKey": "bLabelValue"}, []*string{
			new("Grafana"),
			new("❤️"),
			nil,
			new("🦥"),
			new("update your unicode/font if no sloth, is 2019."),
		}).SetConfig(nullableStringValuesFieldConfig),
		data.NewField("int8_values", nil, []int8{
			math.MinInt8,
			math.MinInt8,
			0,
			math.MaxInt8,
			math.MaxInt8,
		}).SetConfig((&data.FieldConfig{}).SetMin(0).SetMax(1)),
		data.NewField("nullable_int8_values", nil, []*int8{
			new(int8(math.MinInt8)),
			new(int8(math.MinInt8)),
			nil,
			new(int8(math.MaxInt8)),
			new(int8(math.MaxInt8)),
		}),
		data.NewField("int16_values", nil, []int16{
			math.MinInt16,
			math.MinInt16,
			0,
			math.MaxInt16,
			math.MaxInt16,
		}),
		data.NewField("nullable_int16_values", nil, []*int16{
			new(int16(math.MinInt16)),
			new(int16(math.MinInt16)),
			nil,
			new(int16(math.MaxInt16)),
			new(int16(math.MaxInt16)),
		}),
		data.NewField("int32_values", nil, []int32{
			math.MinInt32,
			math.MinInt32,
			1,
			math.MaxInt32,
			math.MaxInt32,
		}),
		data.NewField("nullable_int32_values", nil, []*int32{
			new(int32(math.MinInt32)),
			new(int32(math.MinInt32)),
			nil,
			new(int32(math.MaxInt32)),
			new(int32(math.MaxInt32)),
		}),
		data.NewField("int64_values", nil, []int64{
			math.MinInt64,
			minEcma6Int,
			1,
			maxEcma6Int,
			math.MaxInt64,
		}),
		data.NewField("nullable_int64_values", nil, []*int64{
			new(int64(math.MinInt64)),
			new(int64(minEcma6Int)),
			nil,
			new(int64(maxEcma6Int)),
			new(int64(math.MaxInt64)),
		}),
		data.NewField("uint8_values", nil, []uint8{
			0,
			0,
			1,
			math.MaxUint8,
			math.MaxUint8,
		}),
		data.NewField("nullable_uint8_values", nil, []*uint8{
			new(uint8(0)),
			new(uint8(0)),
			nil,
			new(uint8(math.MaxUint8)),
			new(uint8(math.MaxUint8)),
		}),
		data.NewField("uint16_values", nil, []uint16{
			0,
			0,
			1,
			math.MaxUint16,
			math.MaxUint16,
		}),
		data.NewField("nullable_uint16_values", nil, []*uint16{
			new(uint16(0)),
			new(uint16(0)),
			nil,
			new(uint16(math.MaxUint16)),
			new(uint16(math.MaxUint16)),
		}),
		data.NewField("uint32_values", nil, []uint32{
			0,
			0,
			1,
			math.MaxUint32,
			math.MaxUint32,
		}),
		data.NewField("nullable_uint32_values", nil, []*uint32{
			new(uint32(0)),
			new(uint32(0)),
			nil,
			new(uint32(math.MaxUint32)),
			new(uint32(math.MaxUint32)),
		}),
		data.NewField("uint64_values", nil, []uint64{
			0,
			0,
			1,
			uint64(maxEcma6Int),
			math.MaxUint64,
		}),
		data.NewField("nullable_uint64_values", nil, []*uint64{
			new(uint64(0)),
			new(uint64(0)),
			nil,
			new(uint64(maxEcma6Int)),
			new(uint64(math.MaxUint64)),
		}),
		data.NewField("float32_values", nil, []float32{
			math.SmallestNonzeroFloat32,
			math.SmallestNonzeroFloat32,
			1.0,
			math.MaxFloat32,
			math.MaxFloat32,
		}),
		data.NewField("nullable_float32_values", nil, []*float32{
			new(float32(math.SmallestNonzeroFloat32)),
			new(float32(math.SmallestNonzeroFloat32)),
			nil,
			new(float32(math.MaxFloat32)),
			new(float32(math.MaxFloat32)),
		}),
		data.NewField("float32_values_nans", nil, []float32{
			float32(math.NaN()),
			float32(math.Inf(1)),
			1.0,
			float32(math.Inf(-1)),
			0,
		}),
		data.NewField("nullable_float32_values_nans", nil, []*float32{
			new(float32(math.Inf(1))),
			new(float32(math.NaN())),
			nil,
			new(float32(math.Inf(-1))),
			new(float32(0)),
		}),
		data.NewField("float64_values", nil, []float64{
			math.SmallestNonzeroFloat64,
			float64(minEcma6Int),
			1.0,
			float64(maxEcma6Int),
			math.MaxFloat64,
		}),
		data.NewField("float64_nans", nil, []float64{
			math.Inf(-1),
			math.NaN(),
			0,
			math.NaN(),
			math.Inf(1),
		}),
		data.NewField("nullable_float64_values", nil, []*float64{
			new(math.SmallestNonzeroFloat64),
			new(float64(minEcma6Int)),
			nil,
			new(math.MaxFloat64),
			new(float64(maxEcma6Int)),
		}),
		data.NewField("nullable_float64_values_nans", nil, []*float64{
			new(math.Inf(-1)),
			new(float64(0)),
			nil,
			new(math.NaN()),
			new(math.Inf(1)),
		}),
		data.NewField("bool_values", nil, []bool{
			true,
			false,
			true,
			true,
			false,
		}),
		data.NewField("nullable_bool_values", nil, []*bool{
			new(true),
			new(false),
			nil,
			new(true),
			new(false),
		}),

		data.NewField("timestamps", nil, []time.Time{
			time.Unix(0, 0),
			time.Unix(1568039445, 0),
			time.Unix(1568039450, 0),
			time.Unix(0, maxEcma6Int),
			time.Unix(0, math.MaxInt64),
		}).SetConfig(&data.FieldConfig{
			Interval: 1000,
		}),
		// Note: This is intentionally repeated to create a duplicate field.
		data.NewField("timestamps", nil, []time.Time{
			time.Unix(0, 0),
			time.Unix(1568039445, 0),
			time.Unix(1568039450, 0),
			time.Unix(0, maxEcma6Int),
			time.Unix(0, math.MaxInt64),
		}),
		data.NewField("nullable_timestamps", nil, []*time.Time{
			new(time.Unix(0, 0)),
			new(time.Unix(1568039445, 0)),
			nil,
			new(time.Unix(0, maxEcma6Int)),
			new(time.Unix(0, math.MaxInt64)),
		}),
		data.NewField("json", nil, []json.RawMessage{
			json.RawMessage("{\"a\":1}"),
			json.RawMessage("[1,2,3]"),
			json.RawMessage("{\"b\":2}"),
			json.RawMessage("[{\"c\":3},{\"d\":4}]"),
			json.RawMessage("{\"e\":{\"f\":5}}"),
		}),
		data.NewField("nullable_json", nil, []*json.RawMessage{
			new(json.RawMessage("{\"a\":1}")),
			new(json.RawMessage("[1,2,3]")),
			nil,
			new(json.RawMessage("[{\"c\":3},{\"d\":4}]")),
			new(json.RawMessage("{\"e\":{\"f\":5}}")),
		}),
		data.NewField("enum", nil, []data.EnumItemIndex{
			1, 2, 2, 1, 1,
		}).SetConfig(&data.FieldConfig{
			TypeConfig: &data.FieldTypeConfig{
				Enum: &data.EnumFieldConfig{
					Text: []string{
						"", "ONE", "TWO", "THREE",
					},
				},
			},
		}),
		data.NewField("nullable_enum", nil, []*data.EnumItemIndex{
			(*data.EnumItemIndex)(new(uint16(1))),
			(*data.EnumItemIndex)(new(uint16(2))),
			nil,
			(*data.EnumItemIndex)(new(uint16(3))),
			(*data.EnumItemIndex)(new(uint16(0))),
		}),
	).SetMeta(&data.FrameMeta{
		Custom:              map[string]any{"Hi": "there"},
		ExecutedQueryString: "SELECT * FROM table",
		Channel:             "sample/channel/name",
		Stats: []data.QueryStat{
			{
				FieldConfig: data.FieldConfig{
					DisplayName: "sample",
				},
				Value: 1.234,
			},
		},
	})

	df.RefID = "A"
	return df
}

func newField[V any](name string, ftype data.FieldType, vals []V) *data.Field {
	field := data.NewFieldFromFieldType(ftype, len(vals))
	field.Name = name
	for i, v := range vals {
		field.Set(i, v)
	}
	return field
}

func TestEncode(t *testing.T) {
	df := goldenDF()
	b, err := df.MarshalArrow()
	if err != nil {
		t.Fatal(err)
	}

	goldenFile := filepath.Join("testdata", "all_types.golden.arrow")

	if *update {
		if err := os.WriteFile(goldenFile, b, 0600); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(goldenFile) // #nosec G304 -- Test file is read from testdata directory
	if err != nil {
		t.Fatal(err)
	}

	// Check for the same exact file after encode
	if !bytes.Equal(b, want) {
		// check if the file still represents the same frame or not
		newDf, err := data.UnmarshalArrowFrame(want)
		if err != nil {
			t.Fatal("unable to create frame from encoded arrow file")
		}

		if diff := cmp.Diff(df, newDf, data.FrameTestCompareOptions()...); diff != "" {
			t.Errorf("Arrow frame result mismatch (-want +got):\n%s", diff)
		}

		t.Fatalf("arrow file doesn't match golden file (new version?)")
	}
}

// protip: `go get github.com/apache/arrow/go/arrow/ipc/cmd/arrow-cat` (in GOPATH to install cmd).
// Then in shell: `arrow-cat data/testdata/all_types.golden.arrow`
// also: `go get github.com/apache/arrow/go/arrow/ipc/cmd/arrow-ls` to see metadata

func TestDecode(t *testing.T) {
	goldenFile := filepath.Join("testdata", "all_types.golden.arrow")
	b, err := os.ReadFile(goldenFile) // #nosec G304 -- Test file is read from testdata directory
	if err != nil {
		t.Fatal(err)
	}

	newDf, err := data.UnmarshalArrowFrame(b)
	if err != nil {
		t.Fatal(err)
	}

	df := goldenDF()

	if diff := cmp.Diff(df, newDf, data.FrameTestCompareOptions()...); diff != "" {
		t.Errorf("Result mismatch (-want +got):\n%s", diff)
	}
}

func TestEncodeAndDecodeDuplicateFieldNames(t *testing.T) {
	frame := data.NewFrame("frame_dup_field_names",
		data.NewField("Duplicate", nil, []bool{true, false}),
		data.NewField("Duplicate", nil, []bool{false, true}),
	)

	encoded, err := frame.MarshalArrow()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := data.UnmarshalArrowFrame(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(frame, decoded, data.FrameTestCompareOptions()...); diff != "" {
		t.Errorf("Result mismatch (-want +got):\n%s", diff)
	}
}

func TestFrameMarshalArrowRowLenError(t *testing.T) {
	f := data.NewFrame("unequal length fields",
		data.NewField("1", nil, []string{}),
		data.NewField("1", nil, []string{"a"}),
	)
	_, err := f.MarshalArrow()
	require.Error(t, err)
}

func TestFrameMarshalArrowNoFields(t *testing.T) {
	f := data.NewFrame("no fields")
	_, err := f.MarshalArrow()
	require.NoError(t, err)
}

func TestFromRecord(t *testing.T) {
	df := goldenDF()
	b, err := df.MarshalArrow()
	if err != nil {
		t.Fatal(err)
	}

	// Write golden data frame to file so we can read it back in via Record reader
	fd, err := os.CreateTemp("", "data-test-from-record")
	require.NoError(t, err)
	name := fd.Name()
	defer func() { _ = os.Remove(name) }()
	n, err := fd.Write(b)
	require.NoError(t, err)
	require.Equal(t, len(b), n)

	// Read serialised data frame back into Arrow Record.
	r, err := ipc.NewFileReader(fd)
	require.NoError(t, err)
	record, err := r.Read()
	require.NoError(t, err)

	// Convert Arrow record to data frame.
	got, err := data.FromArrowRecord(record)
	require.NoError(t, err)

	if diff := cmp.Diff(df, got, data.FrameTestCompareOptions()...); diff != "" {
		t.Errorf("Result mismatch (-want +got):\n%s", diff)
	}
}

func TestFromRecordStringView(t *testing.T) {
	pool := memory.NewGoAllocator()
	require.NotNil(t, pool)
	schema := arrow.NewSchema([]arrow.Field{
		arrow.Field{Name: "sv", Type: &arrow.StringViewType{}, Nullable: false},
		arrow.Field{Name: "svn", Type: &arrow.StringViewType{}, Nullable: true},
	}, nil)
	require.NotNil(t, schema)
	b := array.NewRecordBuilder(pool, schema)
	defer b.Release()

	testStrings := []string{"foo", "", "", "🦥", "bar"}
	notNull := []bool{true, true, false, true, true}
	b.Field(0).(*array.StringViewBuilder).AppendValues(testStrings, nil)
	b.Field(1).(*array.StringViewBuilder).AppendValues(testStrings, notNull)
	record := b.NewRecord() //nolint:staticcheck // SA1019: Using deprecated NewRecord() for test backwards compatibility
	defer record.Release()

	got, err := data.FromArrowRecord(record)
	require.NoError(t, err)

	want := data.NewFrame("",
		data.NewField("sv", data.Labels{}, testStrings),
		data.NewField("svn", data.Labels{}, []*string{
			new("foo"),
			new(""),
			nil,
			new("🦥"),
			new("bar"),
		}),
	)
	if diff := cmp.Diff(want, got, data.FrameTestCompareOptions()...); diff != "" {
		t.Errorf("Result mismatch (-want +got):\n%s", diff)
	}
}
