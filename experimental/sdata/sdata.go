package main

import (
	"fmt"
	tpkg "time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/davecgh/go-spew/spew"

)

type number interface {
	float64 | *float64 | int64 // ....
}

type numbers[T number] []T

type time interface {
	tpkg.Time | *tpkg.Time
}

type times[T time] []T

type TimeSeriesMany data.Frames

// "methods cannot have type parameters" :-(
// For above, TODO: Look at https://stackoverflow.com/questions/64189810/function-type-cannot-have-type-parameters
func AddSeriesToTimeSeriesMany[T time, N number](tsm *TimeSeriesMany, metricName string, labels data.Labels, t []T, n []N) error {
	if len(t) != len(n) {
		return fmt.Errorf("time and values must be of the same length")
	}
	frame := data.NewFrame("",
		data.NewField("time", nil, t), // time is just for convience, not a requirement of the schema.
		data.NewField(metricName, labels, n),
	)

	*tsm = append(*tsm, frame)

	return nil
}

func TSMAddSeries[T time, N number](tsm *TimeSeriesMany, s Series[T, N]) error {
	if len(s.T) != len(s.N) {
		return fmt.Errorf("time and values must be of the same length")
	}
	frame := data.NewFrame("",
		data.NewField("time", nil, s.T), // time is just for convience, not a requirement of the schema.
		data.NewField(s.Name, s.Labels, s.N),
	)

	*tsm = append(*tsm, frame)

	return nil
}

// func AddSeriesToTimeSeriesMany[T time, N number, S Series[T, N]](tsm *TimeSeriesMany, s S) error {
// 	return nil
// }

type Series[T time, N number] struct {
	Name string
	Labels data.Labels
	
	T []T
	N []N
}

// type Series[T time, N number] interface {
// 	SetSeries(metricName string, labels data.Labels, t []T, n []N) error
// 	GetSeries() Series[T, N]
// }

// type Series[T time, N number] interface {
// 	[]T
// 	[]N
// 	SetValues([]T, []N)
// 	// GetName() string
// 	// SetName(string)
	
// 	// GetLabels(data.Labels)
// 	// SetLabels(data.Labels)
	
// }

//type SeriesSet[T time, N number] []Series[T, N] // This would imply all series in the set have the same type of number
// while maybe often the case not a restriction that I think we want.

// func CreateSeries[T time, N number](metricName string, labels data.Labels, t []T, n []N) (Series[T, N], error) {
// 	return nil, nil
// }

func main() {
	s := Series[tpkg.Time, float64]{
		Name: "cpu",
		Labels: data.Labels{"host": "web1"},
		T: []tpkg.Time{tpkg.Now()},
		N: []float64{3.2},
	}

	tsm := &TimeSeriesMany{}
	TSMAddSeries(tsm, s)
	// AddSeriesToTimeSeriesMany(tsm, "cpu", data.Labels{"host": "a"}, []tpkg.Time{tpkg.Now()}, []float64{1.1})
	spew.Dump(tsm)
}

