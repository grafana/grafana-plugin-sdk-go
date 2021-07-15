package experimental

import (
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var update = flag.Bool("update", true, "update.golden.data files")

func TestGoldenResponseChecker(t *testing.T) {
	dr := &backend.DataResponse{}

	a := "A"

	//	frame := data.GoldenDF() ????
	dr.Frames = data.Frames{
		data.NewFrame("Frame One",
			data.NewField("Single float64", nil, []float64{
				8.26, 8.7, 14.82,
			}).SetConfig(&data.FieldConfig{Unit: "Percent"}),
			data.NewField("strval", nil, []string{
				"a", "b", "c",
			}),
			data.NewField("nillstrval", nil, []*string{
				&a, nil, &a,
			}),
			data.NewField("time", nil, []time.Time{
				time.Date(2000, 0, 0, 0, 0, 0, 0, time.UTC),
				time.Date(2001, 0, 0, 0, 0, 0, 0, time.UTC),
				time.Date(2002, 0, 0, 0, 0, 0, 0, time.UTC),
			}),
		),
		data.NewFrame("Frame Two",
			data.NewField("single string", data.Labels{"a": "b"}, []string{
				"a", "b", "c",
			}).SetConfig(&data.FieldConfig{DisplayName: "123"}),
		),
	}
	dr.Frames[0].Meta = &data.FrameMeta{
		ExecutedQueryString: "SELECT * FROM X",
		Notices: []data.Notice{
			{Severity: data.NoticeSeverityInfo, Text: "hello"},
		},
	}

	goldenFile := filepath.Join("testdata", "sample.golden.txt")

	err := CheckGoldenDataResponse(goldenFile, dr, *update)
	if err != nil {
		t.Error(err)
	}
}
