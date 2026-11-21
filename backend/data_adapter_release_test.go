package backend

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type keptResponseHandler struct {
	rsp *QueryDataResponse
}

func (h *keptResponseHandler) QueryData(context.Context, *QueryDataRequest) (*QueryDataResponse, error) {
	h.rsp = &QueryDataResponse{Responses: Responses{"A": {Frames: codecSeries(3, 2)}}}
	return h.rsp, nil
}

func TestDataSDKAdapterReleaseFrames(t *testing.T) {
	want, err := ToProto().QueryDataResponse(DataFrameFormat_ARROW, &QueryDataResponse{Responses: Responses{"A": {Frames: codecSeries(3, 2)}}})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		opts     ServeOpts
		released bool
	}{
		{name: "default keeps the plugin's frames", opts: ServeOpts{}},
		{name: "ReleaseQueryDataFrames releases them", opts: ServeOpts{ReleaseQueryDataFrames: true}, released: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &keptResponseHandler{}
			tc.opts.QueryDataHandler = h
			pluginOpts, err := GRPCServeOpts(tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			got, err := pluginOpts.DataServer.QueryData(context.Background(), &pluginv2.QueryDataRequest{PluginContext: &pluginv2.PluginContext{}})
			if err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(got, want) {
				t.Error("the encoded response differs from ToProto().QueryDataResponse")
			}
			for i, f := range h.rsp.Responses["A"].Frames {
				if released := f == nil; released != tc.released {
					t.Errorf("frame %d released = %v, want %v", i, released, tc.released)
				}
			}
		})
	}
}

func TestToProtoQueryDataResponseKeepsFrames(t *testing.T) {
	rsp := &QueryDataResponse{Responses: Responses{"A": {Frames: data.Frames{data.NewFrame("a")}}}}
	if _, err := ToProto().QueryDataResponse(DataFrameFormat_ARROW, rsp); err != nil {
		t.Fatal(err)
	}
	if rsp.Responses["A"].Frames[0] == nil {
		t.Error("ToProto().QueryDataResponse released a frame")
	}
}
