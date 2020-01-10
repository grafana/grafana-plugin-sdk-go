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
			"Grafana",
			"❤️",
			"Transforms",
		}).SetConfig(&dataframe.FieldConfig{}),
		dataframe.NewField("nullable_string_values", dataframe.Labels{"aLabelKey": "aLabelValue", "bLabelKey": "bLabelValue"}, []*string{
			stringPtr("🦥"),
			nil,
			stringPtr("update your unicode/font if no sloth, is 2019."),
		}).SetConfig(nullableStringValuesFieldConfig),
		dataframe.NewField("int_values", nil, []int64{
			math.MinInt64,
			1,
			math.MaxInt64,
		}).SetConfig((&dataframe.FieldConfig{}).SetMin(0).SetMax(1)),
		dataframe.NewField("nullable_int_values", nil, []*int64{
			intPtr(math.MinInt64),
			nil,
			intPtr(math.MaxInt64),
		}),
		dataframe.NewField("uint_values", nil, []uint64{
			0,
			1,
			math.MaxUint64,
		}),
		dataframe.NewField("nullable_uint_values", nil, []*uint64{
			uintPtr(0),
			nil,
			uintPtr(math.MaxUint64),
		}),
		dataframe.NewField("float_values", nil, []float64{
			0.0,
			1.0,
			2.0,
		}),
		dataframe.NewField("nullable_float_values", nil, []*float64{
			floatPtr(0.0),
			nil,
			floatPtr(2.0),
		}),
		dataframe.NewField("bool_values", nil, []bool{
			true,
			true,
			false,
		}),
		dataframe.NewField("nullable_bool_values", nil, []*bool{
			boolPtr(true),
			nil,
			boolPtr(false),
		}),
		dataframe.NewField("timestamps", nil, []time.Time{
			time.Unix(1568039445, 0),
			time.Unix(1568039450, 0),
			time.Unix(1568039455, 0),
		}),
		dataframe.NewField("nullable_timestamps", nil, []*time.Time{
			timePtr(time.Unix(1568039445, 0)),
			nil,
			timePtr(time.Unix(1568039455, 0)),
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
