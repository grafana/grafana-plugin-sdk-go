package parsers

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPromFrames(t *testing.T) {
	files := []string{
		"simple-labels",
	}

	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			f, err := os.Open(path.Join("testdata", name+".json"))
			require.NoError(t, err)

			iter := jsoniter.Parse(jsoniter.ConfigDefault, f, 1024)
			rsp := ReadPrometheusResult(iter)

			out, err := jsoniter.MarshalIndent(rsp, "", "  ")
			require.NoError(t, err)

			save := false
			fpath := path.Join("testdata", name+".out.json")
			current, err := ioutil.ReadFile(fpath)
			if err == nil {
				same := assert.JSONEq(t, string(out), string(current))
				if !same {
					save = true
				}
			} else {
				assert.Fail(t, "missing file: %s", fpath)
				save = true
			}

			if save {
				err = os.WriteFile(fpath, out, 0600)
				require.NoError(t, err)
			}
		})
	}
}
