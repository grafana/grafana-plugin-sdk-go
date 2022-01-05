package sdata_test

import (
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
)

func TestPlayGround(t *testing.T) {
	var mfs sdata.MultiFrameSeries
	mfs.AddMetric("os.cpu", data.Labels{"host": "a"}, []time.Time{time.Unix(1234567890, 0)}, []float64{3})
	spew.Dump(mfs)
}
