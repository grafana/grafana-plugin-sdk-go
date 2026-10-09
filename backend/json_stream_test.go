package backend_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"testing"
	"time"

	j "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

func streamTestResponses(t *testing.T) map[string]*backend.QueryDataResponse {
	t.Helper()
	golden, err := os.ReadFile("../data/testdata/all_types.golden.json")
	require.NoError(t, err)
	allTypes := &data.Frame{}
	require.NoError(t, json.Unmarshal(golden, allTypes))

	inf, one := math.Inf(1), 1.5
	labelled := data.NewFrame("cpu \"quoted\" ünïcode",
		data.NewField("time", nil, []time.Time{time.Unix(1, 0).UTC(), time.Unix(2, 0).UTC()}),
		data.NewField("value", data.Labels{"host": "a,b=c"}, []*float64{&one, &inf}),
		data.NewField("nan", nil, []float64{math.NaN(), math.Inf(-1)}),
	)
	labelled.Fields[1].Config = &data.FieldConfig{Unit: "percent", Custom: map[string]any{"a": 1.0, "b": []any{"x"}}}
	labelled.Meta = &data.FrameMeta{ExecutedQueryString: `up{job="x"}`, Custom: map[string]any{"n": 12345678901234.0}, Notices: []data.Notice{{Severity: data.NoticeSeverityWarning, Text: "careful"}}}

	rich := backend.NewQueryDataResponse()
	rich.Responses["A"] = backend.DataResponse{Frames: data.Frames{labelled, allTypes}}
	rich.Responses["B"] = backend.DataResponse{Error: errors.New(`boom "bad"`), Status: backend.StatusBadRequest, ErrorSource: backend.ErrorSourceDownstream}
	rich.Responses[""] = backend.DataResponse{Frames: data.Frames{data.NewFrame("empty")}}
	rich.Responses["C"] = backend.DataResponse{}

	out := map[string]*backend.QueryDataResponse{"rich": rich, "empty": backend.NewQueryDataResponse()}
	rng := rand.New(rand.NewPCG(7, 7)) //nolint:gosec // G404: deterministic test data
	for i := range 200 {
		out[fmt.Sprintf("random-%03d", i)] = randomResponse(rng)
	}
	return out
}

func randomResponse(rng *rand.Rand) *backend.QueryDataResponse {
	qdr := backend.NewQueryDataResponse()
	for r := range 1 + rng.IntN(3) {
		ref := fmt.Sprintf("R%d", r)
		if rng.IntN(5) == 0 {
			qdr.Responses[ref] = backend.DataResponse{Error: fmt.Errorf("e%d", rng.IntN(9)), Status: backend.Status(400 + rng.IntN(100))}
			continue
		}
		var frames data.Frames
		for range rng.IntN(4) {
			n := rng.IntN(40)
			ts := make([]time.Time, n)
			vs := make([]*float64, n)
			ss := make([]string, n)
			for i := range n {
				ts[i] = time.Unix(int64(rng.IntN(2e9)), int64(rng.IntN(1e9))).UTC()
				if rng.IntN(5) > 0 {
					v := rng.NormFloat64() * math.Pow(10, float64(rng.IntN(30)-15))
					vs[i] = &v
				}
				ss[i] = fmt.Sprintf("s%d %s", rng.IntN(100), sampleStrings[rng.IntN(len(sampleStrings))])
			}
			frames = append(frames, data.NewFrame("f", data.NewField("t", nil, ts), data.NewField("v", data.Labels{"l": fmt.Sprint(rng.IntN(9))}, vs), data.NewField("s", nil, ss)))
		}
		qdr.Responses[ref] = backend.DataResponse{Frames: frames}
	}
	return qdr
}

var sampleStrings = []string{"plain", "tab\there", `quote"`, "ünïcode", "</script>", "emoji 😀", "line\nbreak"}

func readValFrom(t *testing.T, cfg j.API, body []byte) (*backend.QueryDataResponse, error) {
	t.Helper()
	iter, err := jsoniter.Parse(cfg, bytes.NewReader(body), 64)
	require.NoError(t, err)
	qdr := &backend.QueryDataResponse{}
	return qdr, iter.ReadVal(qdr)
}

func TestQueryDataResponseReadValMatchesUnmarshalJSON(t *testing.T) {
	for name, qdr := range streamTestResponses(t) {
		t.Run(name, func(t *testing.T) {
			body, err := qdr.MarshalJSON()
			require.NoError(t, err)
			want := &backend.QueryDataResponse{}
			require.NoError(t, want.UnmarshalJSON(body))
			wantJSON, err := want.MarshalJSON()
			require.NoError(t, err)

			for _, cfg := range []j.API{jsoniter.ConfigCompatibleWithStandardLibrary, jsoniter.ConfigDefault} {
				got, err := readValFrom(t, cfg, body)
				require.NoError(t, err)
				gotJSON, err := got.MarshalJSON()
				require.NoError(t, err)
				require.JSONEq(t, string(wantJSON), string(gotJSON))
			}
		})
	}
}

func TestQueryDataResponseReadValSpecialBodies(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `{"results":null}`, `{"results":{}}`, `{"results":{"A":{"status":200}}}`} {
		t.Run(body, func(t *testing.T) {
			want := &backend.QueryDataResponse{}
			require.NoError(t, want.UnmarshalJSON([]byte(body)))
			got, err := readValFrom(t, jsoniter.ConfigCompatibleWithStandardLibrary, []byte(body))
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
	for _, body := range []string{`{"kind":"Status","status":"Failure"}`, `{"results":{"A":{"frames":[{"schema":{"name":"x`, `[1,2]`} {
		t.Run(body, func(t *testing.T) {
			want := &backend.QueryDataResponse{}
			require.Error(t, want.UnmarshalJSON([]byte(body)))
			_, err := readValFrom(t, jsoniter.ConfigCompatibleWithStandardLibrary, []byte(body))
			require.Error(t, err)
		})
	}
}

func TestQueryDataResponseReadValIntoExistingValue(t *testing.T) {
	existing := func() *backend.QueryDataResponse {
		q := backend.NewQueryDataResponse()
		q.Responses["old"] = backend.DataResponse{Status: backend.StatusOK}
		return q
	}
	for _, body := range []string{`{}`, `{"results":{"new":{"status":200}}}`} {
		t.Run(body, func(t *testing.T) {
			want := existing()
			require.NoError(t, want.UnmarshalJSON([]byte(body)))
			iter, err := jsoniter.Parse(jsoniter.ConfigCompatibleWithStandardLibrary, bytes.NewReader([]byte(body)), 64)
			require.NoError(t, err)
			got := existing()
			require.NoError(t, iter.ReadVal(got))
			require.Equal(t, want, got)
		})
	}
}

func TestQueryDataResponseReadValDoesNotCopyBody(t *testing.T) {
	body, err := streamTestResponses(t)["rich"].MarshalJSON()
	require.NoError(t, err)
	big := backend.NewQueryDataResponse()
	frames := make(data.Frames, 0, 200)
	for range 200 {
		var f data.Frame
		require.NoError(t, json.Unmarshal(mustFrameJSON(t, body), &f))
		frames = append(frames, &f)
	}
	big.Responses["A"] = backend.DataResponse{Frames: frames}
	body, err = big.MarshalJSON()
	require.NoError(t, err)

	unmarshalBytes := allocatedBytes(func() {
		q := &backend.QueryDataResponse{}
		require.NoError(t, q.UnmarshalJSON(body))
	})
	readValBytes := allocatedBytes(func() {
		_, err := readValFrom(t, jsoniter.ConfigCompatibleWithStandardLibrary, body)
		require.NoError(t, err)
	})
	require.LessOrEqual(t, readValBytes, unmarshalBytes*11/10, "ReadVal allocated %d bytes, UnmarshalJSON %d, body %d", readValBytes, unmarshalBytes, len(body))
}

func TestQueryDataResponseReadValKeepsReadError(t *testing.T) {
	body, err := streamTestResponses(t)["rich"].MarshalJSON()
	require.NoError(t, err)
	for _, cause := range []error{context.Canceled, fmt.Errorf("read tcp: %w", context.DeadlineExceeded)} {
		t.Run(cause.Error(), func(t *testing.T) {
			for n := range len(body) {
				iter, err := jsoniter.Parse(jsoniter.ConfigCompatibleWithStandardLibrary, &failAfter{r: bytes.NewReader(body), n: n, err: cause}, 64)
				require.NoError(t, err)
				qdr := &backend.QueryDataResponse{}
				require.NotPanics(t, func() { err = iter.ReadVal(qdr) }, "read fails after %d bytes", n)
				require.ErrorIs(t, err, cause, "read fails after %d bytes", n)
			}
		})
	}
}

func BenchmarkQueryDataResponseDecode(b *testing.B) {
	big := backend.NewQueryDataResponse()
	frames := make(data.Frames, 0, 2000)
	ts := make([]time.Time, 600)
	vs := make([]float64, 600)
	for i := range ts {
		ts[i] = time.Unix(int64(1_700_000_000+i), 0).UTC()
		vs[i] = float64(i) * 1.25
	}
	for i := range 2000 {
		frames = append(frames, data.NewFrame("", data.NewField("Time", nil, ts), data.NewField("Value", data.Labels{"instance": fmt.Sprintf("host-%d", i)}, vs)))
	}
	big.Responses["A"] = backend.DataResponse{Frames: frames}
	body, err := big.MarshalJSON()
	if err != nil {
		b.Fatal(err)
	}
	b.Run("UnmarshalJSON", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(body)))
		for b.Loop() {
			q := &backend.QueryDataResponse{}
			if err := q.UnmarshalJSON(body); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ReadVal from a reader", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(body)))
		for b.Loop() {
			iter, err := jsoniter.Parse(jsoniter.ConfigCompatibleWithStandardLibrary, bytes.NewReader(body), 10*1024)
			if err != nil {
				b.Fatal(err)
			}
			if err := iter.ReadVal(&backend.QueryDataResponse{}); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// failAfter returns err once n bytes have been read.
type failAfter struct {
	r   *bytes.Reader
	n   int
	err error
}

func (f *failAfter) Read(p []byte) (int, error) {
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

func mustFrameJSON(t *testing.T, qdrBody []byte) []byte {
	t.Helper()
	q := &backend.QueryDataResponse{}
	require.NoError(t, q.UnmarshalJSON(qdrBody))
	b, err := json.Marshal(q.Responses["A"].Frames[1])
	require.NoError(t, err)
	return b
}

func allocatedBytes(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}
