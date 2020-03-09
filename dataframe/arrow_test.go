package dataframe_test

import (
	"bytes"
	"flag"
	"io/ioutil"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
)

var update = flag.Bool("update", false, "update .golden.arrow files")

var MAX_ECMA6_INT = int64(1<<53 - 1)
var MIN_ECMA6_INT = int64(-MAX_ECMA6_INT)

func goldenDF() *dataframe.Frame {
	nullableStringValuesFieldConfig := (&dataframe.FieldConfig{
		Title: "Grafana ❤️ (Previous should be heart emoji) 🦥 (Previous should be sloth emoji)",
		Links: []dataframe.DataLink{
			dataframe.DataLink{
				Title:       "Donate - The Sloth Conservation Foundation",
				TargetBlank: true,
				URL:         "https://slothconservation.com/how-to-help/donate/",
			},
		},
		NoValue:       "😤",
		NullValueMode: dataframe.NullValueModeNull,
		// math.NaN() and math.Infs become null when encoded to json
	}).SetDecimals(2).SetMax(math.Inf(1)).SetMin(math.NaN()).SetFilterable(false)

	df := dataframe.New("many_types",
		dataframe.NewField("string_values", dataframe.Labels{"aLabelKey": "aLabelValue"}, []string{
			"Go Min",
			"JS Min (for >= 64)",
			"0 / nil / misc",
			"JS Max (for >= 64)",
			"Go Max",
		}).SetConfig(&dataframe.FieldConfig{}),
		dataframe.NewField("nullable_string_values", dataframe.Labels{"aLabelKey": "aLabelValue", "bLabelKey": "bLabelValue"}, []*string{
			stringPtr("Grafana"),
			stringPtr("❤️"),
			nil,
			stringPtr("🦥"),
			stringPtr("update your unicode/font if no sloth, is 2019."),
		}).SetConfig(nullableStringValuesFieldConfig),
		dataframe.NewField("int8_values", nil, []int8{
			math.MinInt8,
			math.MinInt8,
			0,
			math.MaxInt8,
			math.MaxInt8,
		}).SetConfig((&dataframe.FieldConfig{}).SetMin(0).SetMax(1)),
		dataframe.NewField("nullable_int8_values", nil, []*int8{
			int8Ptr(math.MinInt8),
			int8Ptr(math.MinInt8),
			nil,
			int8Ptr(math.MaxInt8),
			int8Ptr(math.MaxInt8),
		}),
		dataframe.NewField("int16_values", nil, []int16{
			math.MinInt16,
			math.MinInt16,
			0,
			math.MaxInt16,
			math.MaxInt16,
		}),
		dataframe.NewField("nullable_int16_values", nil, []*int16{
			int16Ptr(math.MinInt16),
			int16Ptr(math.MinInt16),
			nil,
			int16Ptr(math.MaxInt16),
			int16Ptr(math.MaxInt16),
		}),
		dataframe.NewField("int32_values", nil, []int32{
			math.MinInt32,
			math.MinInt32,
			1,
			math.MaxInt32,
			math.MaxInt32,
		}),
		dataframe.NewField("nullable_int32_values", nil, []*int32{
			int32Ptr(math.MinInt32),
			int32Ptr(math.MinInt32),
			nil,
			int32Ptr(math.MaxInt32),
			int32Ptr(math.MaxInt32),
		}),
		dataframe.NewField("int64_values", nil, []int64{
			math.MinInt64,
			MIN_ECMA6_INT,
			1,
			MAX_ECMA6_INT,
			math.MaxInt64,
		}),
		dataframe.NewField("nullable_int64_values", nil, []*int64{
			int64Ptr(math.MinInt64),
			int64Ptr(MIN_ECMA6_INT),
			nil,
			int64Ptr(MAX_ECMA6_INT),
			int64Ptr(math.MaxInt64),
		}),
		dataframe.NewField("uint8_values", nil, []uint8{
			0,
			0,
			1,
			math.MaxUint8,
			math.MaxUint8,
		}),
		dataframe.NewField("nullable_uint8_values", nil, []*uint8{
			uint8Ptr(0),
			uint8Ptr(0),
			nil,
			uint8Ptr(math.MaxUint8),
			uint8Ptr(math.MaxUint8),
		}),
		dataframe.NewField("uint16_values", nil, []uint16{
			0,
			0,
			1,
			math.MaxUint16,
			math.MaxUint16,
		}),
		dataframe.NewField("nullable_uint16_values", nil, []*uint16{
			uint16Ptr(0),
			uint16Ptr(0),
			nil,
			uint16Ptr(math.MaxUint16),
			uint16Ptr(math.MaxUint16),
		}),
		dataframe.NewField("uint32_values", nil, []uint32{
			0,
			0,
			1,
			math.MaxUint32,
			math.MaxUint32,
		}),
		dataframe.NewField("nullable_uint32_values", nil, []*uint32{
			uint32Ptr(0),
			uint32Ptr(0),
			nil,
			uint32Ptr(math.MaxUint32),
			uint32Ptr(math.MaxUint32),
		}),

		dataframe.NewField("uint64_values", nil, []uint64{
			0,
			0,
			1,
			uint64(MAX_ECMA6_INT),
			math.MaxUint64,
		}),
		dataframe.NewField("nullable_uint64_values", nil, []*uint64{
			uint64Ptr(0),
			uint64Ptr(0),
			nil,
			uint64Ptr(uint64(MAX_ECMA6_INT)),
			uint64Ptr(math.MaxUint64),
		}),
		dataframe.NewField("float32_values", nil, []float32{
			math.SmallestNonzeroFloat32,
			math.SmallestNonzeroFloat32,
			1.0,
			math.MaxFloat32,
			math.MaxFloat32,
		}),
		dataframe.NewField("nullable_float32_values", nil, []*float32{
			float32Ptr(math.SmallestNonzeroFloat32),
			float32Ptr(math.SmallestNonzeroFloat32),
			nil,
			float32Ptr(math.MaxFloat32),
			float32Ptr(math.MaxFloat32),
		}),
		dataframe.NewField("float64_values", nil, []float64{
			math.SmallestNonzeroFloat64,
			float64(MIN_ECMA6_INT),
			1.0,
			float64(MAX_ECMA6_INT),
			math.MaxFloat64,
		}),
		dataframe.NewField("nullable_float64_values", nil, []*float64{
			float64Ptr(math.SmallestNonzeroFloat64),
			float64Ptr(float64(MIN_ECMA6_INT)),
			nil,
			float64Ptr(math.MaxFloat64),
			float64Ptr(float64(MAX_ECMA6_INT)),
		}),
		dataframe.NewField("bool_values", nil, []bool{
			true,
			false,
			true,
			true,
			false,
		}),
		dataframe.NewField("nullable_bool_values", nil, []*bool{
			boolPtr(true),
			boolPtr(false),
			nil,
			boolPtr(true),
			boolPtr(false),
		}),
		dataframe.NewField("timestamps", nil, []time.Time{
			time.Unix(0, 0),
			time.Unix(1568039445, 0),
			time.Unix(1568039450, 0),
			time.Unix(0, MAX_ECMA6_INT),
			time.Unix(0, math.MaxInt64),
		}),
		dataframe.NewField("nullable_timestamps", nil, []*time.Time{
			timePtr(time.Unix(0, 0)),
			timePtr(time.Unix(1568039445, 0)),
			nil,
			timePtr(time.Unix(0, MAX_ECMA6_INT)),
			timePtr(time.Unix(0, math.MaxInt64)),
		}),
	)

	df.RefID = "A"
	df.Meta = &dataframe.QueryResultMeta{
		SearchWords: []string{"Grafana", "❤️", " 🦥 ", "test"},
		Limit:       4242,
	}
	return df
}

func TestEncode(t *testing.T) {
	df := goldenDF()
	b, err := dataframe.MarshalArrow(df)
	if err != nil {
		t.Fatal(err)
	}

	goldenFile := filepath.Join("testdata", "all_types.golden.arrow")

	if *update {
		if err := ioutil.WriteFile(goldenFile, b, 0644); err != nil {
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
// Then in shell: `arrow-cat dataframe/testdata/all_types.golden.arrow`
// also: `go get github.com/apache/arrow/go/arrow/ipc/cmd/arrow-ls` to see metadata

func TestDecode(t *testing.T) {
	goldenFile := filepath.Join("testdata", "all_types.golden.arrow")
	b, err := ioutil.ReadFile(goldenFile)
	if err != nil {
		t.Fatal(err)
	}

	newDf, err := dataframe.UnmarshalArrow(b)
	if err != nil {
		t.Fatal(err)
	}

	df := goldenDF()

	opt := cmp.Comparer(func(x, y *dataframe.ConfFloat64) bool {
		if x == nil && y == nil {
			return true
		}
		if y == nil {
			if math.IsNaN(float64(*x)) {
				return true
			}
			if math.IsInf(float64(*x), 1) {
				return true
			}
			if math.IsInf(float64(*x), -1) {
				return true
			}
		}
		if x == nil {
			if math.IsNaN(float64(*y)) {
				return true
			}
			if math.IsInf(float64(*y), 1) {
				return true
			}
			if math.IsInf(float64(*y), -1) {
				return true
			}
		}
		return *x == *y
	})

	if diff := cmp.Diff(df, newDf, opt); diff != "" {
		t.Errorf("Result mismatch (-want +got):\n%s", diff)
	}

}
