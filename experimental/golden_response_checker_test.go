package experimental

import (
	"flag"
	"path/filepath"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var update = flag.Bool("update", false, "update.golden.data files")

func TestGoldenResponseChecker(t *testing.T) {
	dr := &backend.DataResponse{}

	//	frame := data.GoldenDF() ????
	dr.Frames = data.Frames{
		data.NewFrame("Frame One",
			data.NewField("Percentage CPU", nil, []float64{
				8.26, 8.7, 14.82, 10.07, 8.52,
			}).SetConfig(&data.FieldConfig{Unit: "Percent"}),
		),
		data.NewFrame("Frame Two",
			data.NewField("Percentage CPU", nil, []string{
				"a", "b", "c",
			}).SetConfig(&data.FieldConfig{DisplayName: "123"}),
		),
	}

	goldenFile := filepath.Join("testdata", "sample.golden.txt")

	CheckGoldenDataResponse(goldenFile, dr, t, *update)
}
