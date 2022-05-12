package experimental

import (
	"encoding/json"
	"flag"
	"path/filepath"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

// some sample custom meta
type SomeCustomMeta struct {
	SomeValue string `json:"someValue,omitempty"`
}

var update = flag.Bool("update", false, "update.golden.data files")

func TestGoldenResponseChecker(t *testing.T) {
	dr := &backend.DataResponse{}

	dr.Frames = data.Frames{
		data.NewFrame("Frame One",
			data.NewField("Single float64", nil, []float64{
				8.26, 8.7, 14.82, 10.07, 8.52,
			}).SetConfig(&data.FieldConfig{Unit: "Percent"}),
		),
		data.NewFrame("Frame Two",
			data.NewField("single string", data.Labels{"a": "b"}, []string{
				"a", "b", "c",
			}).SetConfig(&data.FieldConfig{DisplayName: "123"}),
		),
	}

	t.Run("create data frames with no meta", func(t *testing.T) {
		goldenFile := filepath.Join("testdata", "frame-no-meta.golden")
		checkGoldenFiles(t, goldenFile, dr)
	})

	t.Run("create data frames with some non-custom meta", func(t *testing.T) {
		dr.Frames[0].Meta = &data.FrameMeta{
			ExecutedQueryString: "SELECT * FROM X",
			Notices: []data.Notice{
				{Severity: data.NoticeSeverityInfo, Text: "hello"},
			},
		}

		goldenFile := filepath.Join("testdata", "frame-non-custom-meta.golden")
		checkGoldenFiles(t, goldenFile, dr)
	})

	t.Run("create data frames with some empty custom meta", func(t *testing.T) {
		dr.Frames[0].Meta = &data.FrameMeta{
			Custom: SomeCustomMeta{},
		}

		goldenFile := filepath.Join("testdata", "frame-empty-custom-meta.golden")
		checkGoldenFiles(t, goldenFile, dr)
	})

	t.Run("create data frames with some custom meta", func(t *testing.T) {
		dr.Frames[0].Meta = &data.FrameMeta{
			Custom: SomeCustomMeta{
				SomeValue: "value",
			},
		}

		goldenFile := filepath.Join("testdata", "frame-custom-meta.golden")
		checkGoldenFiles(t, goldenFile, dr)
	})

	t.Run("should render string for JSON fields", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2}
		b, err := json.Marshal(m)
		require.NoError(t, err)
		r := json.RawMessage(b)
		res := &backend.DataResponse{
			Frames: data.Frames{
				data.NewFrame("JSON frame",
					data.NewField("json.RawMessage", nil, []json.RawMessage{r}),
					data.NewField("*json.RawMessage", nil, []*json.RawMessage{&r}),
				),
			}}
		goldenFile := filepath.Join("testdata", "frame-json")
		checkGoldenFiles(t, goldenFile, res)
	})
}

func TestReadGoldenFile(t *testing.T) {
	t.Run("read a large golden file", func(t *testing.T) {
		goldenFile := filepath.Join("testdata", "large.golden.txt")
		dr, err := readGoldenFile(goldenFile)
		require.NotEmpty(t, dr)
		require.NoError(t, err)
	})
}

func checkGoldenFiles(t *testing.T, goldenFile string, dr *backend.DataResponse) {
	err := CheckGoldenDataResponse(goldenFile+".txt", dr, *update)
	require.NoError(t, err)

	err = CheckGoldenJSON(goldenFile+".json", dr, *update)
	require.NoError(t, err)
}
