package data_test

import (
	"bytes"
	"flag"
	"io/ioutil"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update .golden.arrow files")

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
		NoValue:       "😤",
		NullValueMode: data.NullValueModeNull,
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
			stringPtr("Grafana"),
			stringPtr("❤️"),
			nil,
			stringPtr("🦥"),
			stringPtr("update your unicode/font if no sloth, is 2019."),
		}).SetConfig(nullableStringValuesFieldConfig),
		data.NewField("int8_values", nil, []int8{
			math.MinInt8,
			math.MinInt8,
			0,
			math.MaxInt8,
			math.MaxInt8,
		}).SetConfig((&data.FieldConfig{}).SetMin(0).SetMax(1)),
		data.NewField("nullable_int8_values", nil, []*int8{
			int8Ptr(math.MinInt8),
			int8Ptr(math.MinInt8),
			nil,
			int8Ptr(math.MaxInt8),
			int8Ptr(math.MaxInt8),
		}),
		data.NewField("int16_values", nil, []int16{
			math.MinInt16,
			math.MinInt16,
			0,
			math.MaxInt16,
			math.MaxInt16,
		}),
		data.NewField("nullable_int16_values", nil, []*int16{
			int16Ptr(math.MinInt16),
			int16Ptr(math.MinInt16),
			nil,
			int16Ptr(math.MaxInt16),
			int16Ptr(math.MaxInt16),
		}),
		data.NewField("int32_values", nil, []int32{
			math.MinInt32,
			math.MinInt32,
			1,
			math.MaxInt32,
			math.MaxInt32,
		}),
		data.NewField("nullable_int32_values", nil, []*int32{
			int32Ptr(math.MinInt32),
			int32Ptr(math.MinInt32),
			nil,
			int32Ptr(math.MaxInt32),
			int32Ptr(math.MaxInt32),
		}),
		data.NewField("int64_values", nil, []int64{
			math.MinInt64,
			minEcma6Int,
			1,
			maxEcma6Int,
			math.MaxInt64,
		}),
		data.NewField("nullable_int64_values", nil, []*int64{
			int64Ptr(math.MinInt64),
			int64Ptr(minEcma6Int),
			nil,
			int64Ptr(maxEcma6Int),
			int64Ptr(math.MaxInt64),
		}),
		data.NewField("uint8_values", nil, []uint8{
			0,
			0,
			1,
			math.MaxUint8,
			math.MaxUint8,
		}),
		data.NewField("nullable_uint8_values", nil, []*uint8{
			uint8Ptr(0),
			uint8Ptr(0),
			nil,
			uint8Ptr(math.MaxUint8),
			uint8Ptr(math.MaxUint8),
		}),
		data.NewField("uint16_values", nil, []uint16{
			0,
			0,
			1,
			math.MaxUint16,
			math.MaxUint16,
		}),
		data.NewField("nullable_uint16_values", nil, []*uint16{
			uint16Ptr(0),
			uint16Ptr(0),
			nil,
			uint16Ptr(math.MaxUint16),
			uint16Ptr(math.MaxUint16),
		}),
		data.NewField("uint32_values", nil, []uint32{
			0,
			0,
			1,
			math.MaxUint32,
			math.MaxUint32,
		}),
		data.NewField("nullable_uint32_values", nil, []*uint32{
			uint32Ptr(0),
			uint32Ptr(0),
			nil,
			uint32Ptr(math.MaxUint32),
			uint32Ptr(math.MaxUint32),
		}),

		data.NewField("uint64_values", nil, []uint64{
			0,
			0,
			1,
			uint64(maxEcma6Int),
			math.MaxUint64,
		}),
		data.NewField("nullable_uint64_values", nil, []*uint64{
			uint64Ptr(0),
			uint64Ptr(0),
			nil,
			uint64Ptr(uint64(maxEcma6Int)),
			uint64Ptr(math.MaxUint64),
		}),
		data.NewField("float32_values", nil, []float32{
			math.SmallestNonzeroFloat32,
			math.SmallestNonzeroFloat32,
			1.0,
			math.MaxFloat32,
			math.MaxFloat32,
		}),
		data.NewField("nullable_float32_values", nil, []*float32{
			float32Ptr(math.SmallestNonzeroFloat32),
			float32Ptr(math.SmallestNonzeroFloat32),
			nil,
			float32Ptr(math.MaxFloat32),
			float32Ptr(math.MaxFloat32),
		}),
		data.NewField("float64_values", nil, []float64{
			math.SmallestNonzeroFloat64,
			float64(minEcma6Int),
			1.0,
			float64(maxEcma6Int),
			math.MaxFloat64,
		}),
		data.NewField("nullable_float64_values", nil, []*float64{
			float64Ptr(math.SmallestNonzeroFloat64),
			float64Ptr(float64(minEcma6Int)),
			nil,
			float64Ptr(math.MaxFloat64),
			float64Ptr(float64(maxEcma6Int)),
		}),
		data.NewField("bool_values", nil, []bool{
			true,
			false,
			true,
			true,
			false,
		}),
		data.NewField("nullable_bool_values", nil, []*bool{
			boolPtr(true),
			boolPtr(false),
			nil,
			boolPtr(true),
			boolPtr(false),
		}),
		data.NewField("timestamps", nil, []time.Time{
			time.Unix(0, 0),
			time.Unix(1568039445, 0),
			time.Unix(1568039450, 0),
			time.Unix(0, maxEcma6Int),
			time.Unix(0, math.MaxInt64),
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
			timePtr(time.Unix(0, 0)),
			timePtr(time.Unix(1568039445, 0)),
			nil,
			timePtr(time.Unix(0, maxEcma6Int)),
			timePtr(time.Unix(0, math.MaxInt64)),
		}),
	).SetMeta(&data.FrameMeta{
		Custom: map[string]interface{}{"Hi": "there"},
	})

	df.RefID = "A"
	return df
}

func TestEncode(t *testing.T) {
	df := goldenDF()
	b, err := df.MarshalArrow()
	if err != nil {
		t.Fatal(err)
	}

	goldenFile := filepath.Join("testdata", "all_types.golden.arrow")

	if *update {
		if err := ioutil.WriteFile(goldenFile, b, 0600); err != nil {
			t.Fatal(err)
		}
	}

	want, err := ioutil.ReadFile(goldenFile)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b, want) {
		t.Fatalf("data frame doesn't match golden file")
	}
}

// protip: `go get github.com/apache/arrow/go/arrow/ipc/cmd/arrow-cat` (in GOPATH to install cmd).
// Then in shell: `arrow-cat data/testdata/all_types.golden.arrow`
// also: `go get github.com/apache/arrow/go/arrow/ipc/cmd/arrow-ls` to see metadata

func TestDecode(t *testing.T) {
	goldenFile := filepath.Join("testdata", "all_types.golden.arrow")
	b, err := ioutil.ReadFile(goldenFile)
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
