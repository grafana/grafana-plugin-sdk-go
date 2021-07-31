package experimental

import (
	"io/ioutil"
	"path"
	"path/filepath"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func TestJSONDocToFrame(t *testing.T) {
	names := []string{"doc-simple", "doc-complex"}
	for _, name := range names {
		b, err := ioutil.ReadFile(path.Join("testdata", name+".json"))
		require.NoError(t, err)

		f, err := JSONDocToFrame(b)
		require.NoError(t, err)

		dr := &backend.DataResponse{}
		dr.Frames = data.Frames{f}
		goldenFile := filepath.Join("testdata", name+".golden.txt")

		err = CheckGoldenDataResponse(goldenFile, dr, true)
		require.NoError(t, err)
	}
}
