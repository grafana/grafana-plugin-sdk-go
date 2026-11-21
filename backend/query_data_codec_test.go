package backend

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/mem"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

func codecSeries(n, rows int) data.Frames {
	frames := make(data.Frames, n)
	for s := range n {
		ts := make([]time.Time, rows)
		vals := make([]float64, rows)
		for i := range rows {
			ts[i] = time.UnixMilli(1_700_000_000_000 + int64(i)*15_000)
			vals[i] = float64(s*rows + i)
		}
		labels := data.Labels{"__name__": "up", "job": "api", "instance": fmt.Sprintf("host-%d", s)}
		frames[s] = data.NewFrame("", data.NewField("time", nil, ts), data.NewField("value", labels, vals))
	}
	return frames
}

func codecResponses(t *testing.T) map[string]*pluginv2.QueryDataResponse {
	t.Helper()
	toProto := func(format DataFrameFormat, rsp *QueryDataResponse) *pluginv2.QueryDataResponse {
		t.Helper()
		m, err := ToProto().QueryDataResponse(format, rsp)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	return map[string]*pluginv2.QueryDataResponse{
		"instant vector":  toProto(DataFrameFormat_ARROW, &QueryDataResponse{Responses: Responses{"A": {Frames: codecSeries(500, 1)}}}),
		"range":           toProto(DataFrameFormat_ARROW, &QueryDataResponse{Responses: Responses{"A": {Frames: codecSeries(50, 240)}}}),
		"json frames":     toProto(DataFrameFormat_JSON, &QueryDataResponse{Responses: Responses{"A": {Frames: codecSeries(3, 10)}}}),
		"several refIDs":  toProto(DataFrameFormat_ARROW, &QueryDataResponse{Responses: Responses{"B": {Frames: codecSeries(2, 5)}, "": {Frames: codecSeries(1, 1)}, "A": {Status: StatusOK}}}),
		"error responses": toProto(DataFrameFormat_ARROW, &QueryDataResponse{Responses: Responses{"A": ErrDataResponseWithSource(StatusBadRequest, ErrorSourceDownstream, "bad query"), "B": {Frames: codecSeries(1, 3)}}}),
		"no responses":    {Responses: map[string]*pluginv2.DataResponse{}},
		"nil responses":   {},
		"nil frames":      {Responses: map[string]*pluginv2.DataResponse{"A": {Status: 200, JsonMeta: []byte(`{"a":1}`)}}},
		"nil response":    {Responses: map[string]*pluginv2.DataResponse{"A": nil, "B": {Frames: [][]byte{{}}, Status: 200}}},
	}
}

func TestQueryDataCodecMatchesProto(t *testing.T) {
	codec := newQueryDataCodec()
	for name, m := range codecResponses(t) {
		t.Run(name, func(t *testing.T) {
			want, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			out, err := codec.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			if got := out.Materialize(); !bytes.Equal(got, want) {
				t.Fatalf("codec bytes differ from a deterministic proto.Marshal: got %d bytes, want %d", len(got), len(want))
			}
			var back pluginv2.QueryDataResponse
			if err := encoding.GetCodecV2("proto").Unmarshal(out, &back); err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(&back, m) {
				t.Error("the standard codec decodes a different message")
			}
		})
	}
}

func TestQueryDataCodecDoesNotCopyFrames(t *testing.T) {
	frame := bytes.Repeat([]byte{7}, 1<<20)
	m := &pluginv2.QueryDataResponse{Responses: map[string]*pluginv2.DataResponse{"A": {Frames: [][]byte{frame}}}}
	out, err := newQueryDataCodec().Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, buf := range out {
		if b := buf.ReadOnlyData(); len(b) == len(frame) && &b[0] == &frame[0] {
			return
		}
	}
	t.Error("the marshalled message does not reference the frame bytes, so they were copied")
}

func TestQueryDataCodecOtherMessages(t *testing.T) {
	codec := newQueryDataCodec()
	if codec.Name() != "proto" {
		t.Errorf("Name() = %q, want proto", codec.Name())
	}
	m := &pluginv2.CheckHealthResponse{Status: pluginv2.CheckHealthResponse_OK, Message: "ok", JsonDetails: []byte(`{"a":1}`)}
	want, err := encoding.GetCodecV2("proto").Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	got, err := codec.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Materialize(), want.Materialize()) {
		t.Error("codec bytes for another message differ from the proto codec")
	}
}

type fixedDataServer struct {
	pluginv2.UnimplementedDataServer
	rsp *pluginv2.QueryDataResponse
}

func (s fixedDataServer) QueryData(context.Context, *pluginv2.QueryDataRequest) (*pluginv2.QueryDataResponse, error) {
	return s.rsp, nil
}

func TestQueryDataCodecOverGRPC(t *testing.T) {
	want := codecResponses(t)["range"]
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer(queryDataCodecOption())
	pluginv2.RegisterDataServer(srv, fixedDataServer{rsp: want})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(64<<20)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	got, err := pluginv2.NewDataClient(conn).QueryData(context.Background(), &pluginv2.QueryDataRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(got, want) {
		t.Error("the client received a different response")
	}
}

func BenchmarkQueryDataCodecMarshal(b *testing.B) {
	for _, n := range []int{1_000, 25_000} {
		m, err := ToProto().QueryDataResponse(DataFrameFormat_ARROW, &QueryDataResponse{Responses: Responses{"A": {Frames: codecSeries(n, 1)}}})
		if err != nil {
			b.Fatal(err)
		}
		for _, c := range []struct {
			name  string
			codec encoding.CodecV2
		}{{"proto", encoding.GetCodecV2("proto")}, {"queryData", newQueryDataCodec()}} {
			b.Run(fmt.Sprintf("series=%d/%s", n, c.name), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					out, err := c.codec.Marshal(m)
					if err != nil {
						b.Fatal(err)
					}
					out.Free()
				}
			})
		}
	}
}

// ownedBytes returns the bytes in out that are not the frames' own bytes, which is what marshalling adds to the
// heap while grpc-go holds the message until SendMsg returns.
func ownedBytes(out mem.BufferSlice, m *pluginv2.QueryDataResponse) int {
	frames := map[*byte]bool{}
	for _, dr := range m.Responses {
		for _, f := range dr.GetFrames() {
			if len(f) > 0 {
				frames[&f[0]] = true
			}
		}
	}
	n := 0
	for _, buf := range out {
		if b := buf.ReadOnlyData(); len(b) > 0 && !frames[&b[0]] {
			n += len(b)
		}
	}
	return n
}

func TestQueryDataCodecCopiesOnlyHeaders(t *testing.T) {
	m, err := ToProto().QueryDataResponse(DataFrameFormat_ARROW, &QueryDataResponse{Responses: Responses{"A": {Frames: codecSeries(25_000, 1)}}})
	if err != nil {
		t.Fatal(err)
	}
	payload := proto.Size(m)
	for _, c := range []struct {
		name  string
		codec encoding.CodecV2
		limit int
	}{{"proto", encoding.GetCodecV2("proto"), payload}, {"queryData", newQueryDataCodec(), payload / 100}} {
		out, err := c.codec.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		owned := ownedBytes(out, m)
		out.Free()
		t.Logf("%s codec: %d of %d payload bytes are a copy", c.name, owned, payload)
		if owned > c.limit {
			t.Errorf("%s codec copies %d bytes of a %d byte payload, want at most %d", c.name, owned, payload, c.limit)
		}
	}
}
