package dataframe_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
)

func TestDataFrame(t *testing.T) {
	df := dataframe.New("http_requests_total",
		dataframe.NewField("timestamp", nil, []time.Time{time.Now(), time.Now(), time.Now()}),
		dataframe.NewField("value", dataframe.Labels{"service": "auth"}, []float64{1.0, 2.0, 3.0}),
		dataframe.NewField("category", dataframe.Labels{"service": "auth"}, []string{"foo", "bar", "test"}),
		dataframe.NewField("valid", dataframe.Labels{"service": "auth"}, []bool{true, false, true}),
	)

	if df.Rows() != 3 {
		t.Fatal("unexpected length")
	}
}

func TestField(t *testing.T) {
	f := dataframe.NewField("value", nil, []float64{1.0, 2.0, 3.0})

	if f.Len() != 3 {
		t.Fatal("unexpected length")
	}
}

func TestField_Float64(t *testing.T) {
	f := dataframe.NewField("value", nil, make([]*float64, 3))

	want := 2.0
	f.Vector.Set(1, &want)

	if f.Len() != 3 {
		t.Fatal("unexpected length")
	}

	got := f.Vector.At(1).(*float64)

	if *got != want {
		t.Errorf("%+v", *got)
	}
}

func TestField_String(t *testing.T) {
	f := dataframe.NewField("value", nil, make([]*string, 3))

	want := "foo"
	f.Vector.Set(1, &want)

	if f.Len() != 3 {
		t.Fatal("unexpected length")
	}

	got := f.Vector.At(1).(*string)

	if *got != want {
		t.Errorf("%+v", *got)
	}
}

func TestTimeField(t *testing.T) {
	tests := []struct {
		Values []*time.Time
	}{
		{
			Values: []*time.Time{timePtr(time.Unix(111, 0))},
		},
		{
			Values: []*time.Time{nil, timePtr(time.Unix(111, 0))},
		},
		{
			Values: []*time.Time{nil, timePtr(time.Unix(111, 0)), nil},
		},
		{
			Values: make([]*time.Time, 10),
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			f := dataframe.NewField(t.Name(), nil, tt.Values)

			if f.Len() != len(tt.Values) {
				t.Error(f.Len())
			}

			for i := 0; i < f.Len(); i++ {
				got := reflect.ValueOf(f.Vector.At(i))
				want := reflect.ValueOf(tt.Values[i])

				if got != want {
					t.Error(got, want)
				}
			}

		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func float64Ptr(f float64) *float64 {
	return &f
}

func int8Ptr(i int8) *int8 {
	return &i
}

func int16Ptr(i int16) *int16 {
	return &i
}

func int32Ptr(i int32) *int32 {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}

func uint8Ptr(ui uint8) *uint8 {
	return &ui
}

func uint16Ptr(ui uint16) *uint16 {
	return &ui
}

func uint32Ptr(ui uint32) *uint32 {
	return &ui
}

func uint64Ptr(ui uint64) *uint64 {
	return &ui
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
