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

func main() {
	tsm := &TimeSeriesMany{}
	AddSeriesToTimeSeriesMany(tsm, "cpu", data.Labels{"host": "a"}, []tpkg.Time{tpkg.Now()}, []float64{1.1})
	spew.Dump(tsm)
}