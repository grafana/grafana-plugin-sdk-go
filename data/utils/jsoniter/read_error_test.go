package jsoniter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdkjsoniter "github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

type nestedItem struct {
	Name   string            `json:"name"`
	Values []float64         `json:"values"`
	Labels map[string]string `json:"labels"`
	Inner  struct {
		Tags []string `json:"tags"`
	} `json:"inner"`
}

// failingReader returns err once n bytes have been read.
type failingReader struct {
	r   io.Reader
	n   int
	err error
}

func (f *failingReader) Read(p []byte) (int, error) {
	if f.n <= 0 {
		return 0, f.err
	}
	if len(p) > f.n {
		p = p[:f.n]
	}
	k, err := f.r.Read(p)
	f.n -= k
	return k, err
}

func nestedItemsJSON(t *testing.T) []byte {
	t.Helper()
	items := make([]nestedItem, 8)
	for i := range items {
		items[i].Name = fmt.Sprintf("item-%d", i)
		items[i].Values = []float64{float64(i), 1.5, -2.25}
		items[i].Labels = map[string]string{"host": fmt.Sprintf("h%d", i)}
		items[i].Inner.Tags = []string{"a", "b"}
	}
	b, err := json.Marshal(items)
	require.NoError(t, err)
	return b
}

func TestParseKeepsReadError(t *testing.T) {
	b := nestedItemsJSON(t)
	causes := []error{
		context.Canceled,
		fmt.Errorf("read tcp: %w", context.DeadlineExceeded),
		errors.New("http: response body too large"),
	}
	for _, cause := range causes {
		t.Run(cause.Error(), func(t *testing.T) {
			for n := range len(b) {
				iter, err := sdkjsoniter.Parse(sdkjsoniter.ConfigCompatibleWithStandardLibrary, &failingReader{r: bytes.NewReader(b), n: n, err: cause}, 16)
				require.NoError(t, err)
				var out []nestedItem
				err = iter.ReadVal(&out)
				assert.ErrorIs(t, err, cause, "read fails after %d bytes", n)
			}
		})
	}
}

func TestParseReadErrorAfterDecodeError(t *testing.T) {
	cause := errors.New("read failed")
	iter, err := sdkjsoniter.Parse(sdkjsoniter.ConfigDefault, &failingReader{r: bytes.NewReader(nil), err: cause}, 16)
	require.NoError(t, err)
	iter.SetError(errors.New("decode failed"))
	var out []nestedItem
	err = iter.ReadVal(&out)
	require.ErrorContains(t, err, "decode failed")
	require.NotErrorIs(t, err, cause)
}

func TestParseReaderEOF(t *testing.T) {
	b := nestedItemsJSON(t)

	iter, err := sdkjsoniter.Parse(sdkjsoniter.ConfigDefault, bytes.NewReader(b), 16)
	require.NoError(t, err)
	var out []nestedItem
	require.NoError(t, iter.ReadVal(&out))
	require.Len(t, out, 8)

	iter, err = sdkjsoniter.Parse(sdkjsoniter.ConfigDefault, bytes.NewReader(b[:len(b)/2]), 16)
	require.NoError(t, err)
	require.Error(t, iter.ReadVal(&out))
}
