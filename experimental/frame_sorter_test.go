package experimental

import (
	"sort"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestFrameSorter(t *testing.T) {

	field := data.NewField("Single float64", nil, []float64{
		8.6, 8.7, 14.82, 10.07, 8.52,
	}).SetConfig(&data.FieldConfig{Unit: "Percent"})

	frame := data.NewFrame("Frame One",
		field,
	)

	sorter := NewFrameSorter(frame, field)

	sort.Sort(sorter)

	val, err := frame.Fields[0].FloatAt(0)

	if err != nil {
		t.Error(err)
	}
	want := float64(8.52)
	if val != want {
		t.Errorf("Want %f Got %f", want, val)
	}
}
