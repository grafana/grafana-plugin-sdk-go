package data_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

var errCut = errors.New("cut")

// cutReader returns errCut after n bytes.
type cutReader struct {
	r io.Reader
	n int
}

func (c *cutReader) Read(p []byte) (int, error) {
	if c.n <= 0 {
		return 0, errCut
	}
	if len(p) > c.n {
		p = p[:c.n]
	}
	k, err := c.r.Read(p)
	c.n -= k
	return k, err
}

func TestFrameJSONCutAtEveryByte(t *testing.T) {
	b, err := json.Marshal(goldenDF())
	require.NoError(t, err)

	for i := range len(b) {
		t.Run("bytes", func(t *testing.T) {
			f := &data.Frame{}
			var err error
			require.NotPanics(t, func() { err = f.UnmarshalJSON(b[:i]) }, "cut at byte %d", i)
			require.Error(t, err, "cut at byte %d", i)
		})
		t.Run("stream", func(t *testing.T) {
			iter, err := jsoniter.Parse(jsoniter.ConfigCompatibleWithStandardLibrary, &cutReader{r: bytes.NewReader(b), n: i}, 64)
			require.NoError(t, err)
			f := &data.Frame{}
			require.NotPanics(t, func() { err = iter.ReadVal(f) }, "cut at byte %d", i)
			require.Error(t, err, "cut at byte %d", i)
		})
	}
}

func TestFrameJSONFieldWithoutType(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{name: "no type info", json: `{"schema":{"fields":[{"name":"a"}]},"data":{"values":[[1]]}}`},
		{name: "nullable without frame type", json: `{"schema":{"fields":[{"name":"a","typeInfo":{"nullable":true}}]},"data":{"values":[[1]]}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &data.Frame{}
			var err error
			require.NotPanics(t, func() { err = f.UnmarshalJSON([]byte(tt.json)) })
			require.ErrorContains(t, err, `field "a" has no type`)
		})
	}
}
