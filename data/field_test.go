package data_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestField(t *testing.T) {
	f := data.NewField("value", nil, []float64{1.0, 2.0, 3.0})

	if f.Len() != 3 {
		t.Fatal("unexpected length")
	}
}

func TestField_Float64(t *testing.T) {
	field := data.NewField("value", nil, make([]*float64, 3))

	want := 2.0
	field.Set(1, &want)

	if field.Len() != 3 {
		t.Fatal("unexpected length")
	}

	got := field.At(1).(*float64)

	if *got != want {
		t.Errorf("%+v", *got)
	}
}

func TestField_String(t *testing.T) {
	field := data.NewField("value", nil, make([]*string, 3))

	want := "foo"
	field.Set(1, &want)

	if field.Len() != 3 {
		t.Fatal("unexpected length")
	}

	got := field.At(1).(*string)

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

	for i := range tests {
		tt := tests[i]
		t.Run("", func(t *testing.T) {
			f := data.NewField(t.Name(), nil, tt.Values)

			if f.Len() != len(tt.Values) {
				t.Error(f.Len())
			}

			for i := 0; i < f.Len(); i++ {
				got := reflect.ValueOf(f.At(i))
				want := reflect.ValueOf(tt.Values[i])

				if got != want {
					t.Error(got, want)
				}
			}
		})
	}
}
