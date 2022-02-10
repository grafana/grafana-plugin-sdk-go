package sdata_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
)

func TestSimpleMultiFrameNumeric(t *testing.T) {
	var mfn *sdata.MultiFrameNumeric
	var mfnr sdata.NumericCollectionWriter = mfn
	mfnr.AddMetric("os.cpu", data.Labels{"host": "a"}, 1)
}
