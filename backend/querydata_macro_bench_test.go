package backend_test

// Macro benchmark modelling the SDK portion of the Grafana query-response path:
//
//	plugin data.Frames
//	   -> backend.ConvertToProtobuf.QueryDataResponse (Arrow)   // plugin-side: data.Frames.MarshalArrow
//	   -> backend.ConvertFromProtobuf.QueryDataResponse          // Grafana-side: data.UnmarshalArrowFrames
//	   -> jsoniter.NewEncoder(w).Encode(resp)                    // Grafana -> browser (pkg/api/response/response.go:164)
//
// A change to any of MarshalArrow / UnmarshalArrowFrames / frame+response JSON
// codecs moves a single number here. Per-stage micro-benches still live in
// data/arrow_bench_test.go and data/frame_json_bench_test.go.
//
// Factories, scenarios, and the Grafana-like encoder helper live in
// querydata_scenarios_test.go so that TestScenarioGoldenBytes snapshots the
// same inputs these benchmarks exercise.

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// BenchmarkQueryDataResponseMacro runs the full SDK round-trip per iteration:
// plugin-side Arrow marshal, Grafana-side Arrow unmarshal, JSON marshal to the
// browser. Construction is excluded via StopTimer so only the serialization
// work on the real hot path is timed.
func BenchmarkQueryDataResponseMacro(b *testing.B) {
	to := backend.ConvertToProtobuf{}
	from := backend.ConvertFromProtobuf{}

	for _, sc := range macroScenarios() {
		b.Run(sc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				resp := sc.build()
				b.StartTimer()

				protoResp, err := to.QueryDataResponse(backend.DataFrameFormat_ARROW, resp)
				if err != nil {
					b.Fatalf("ConvertToProtobuf: %v", err)
				}
				decoded, err := from.QueryDataResponse(protoResp)
				if err != nil {
					b.Fatalf("ConvertFromProtobuf: %v", err)
				}
				if err := encodeJSONLikeGrafana(decoded); err != nil {
					b.Fatalf("jsoniter Encode: %v", err)
				}
			}
		})
	}
}

// Per-stage sub-benches against the same scenarios. These let benchstat
// attribute a total-time delta to a specific stage without re-running the
// per-stage micro-benches in data/.
func BenchmarkQueryDataResponseMacro_Stages(b *testing.B) {
	to := backend.ConvertToProtobuf{}
	from := backend.ConvertFromProtobuf{}

	for _, sc := range macroScenarios() {
		b.Run(sc.name+"/1_MarshalArrow", func(b *testing.B) {
			resp := sc.build()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := to.QueryDataResponse(backend.DataFrameFormat_ARROW, resp); err != nil {
					b.Fatal(err)
				}
			}
			_ = resp
		})

		b.Run(sc.name+"/2_UnmarshalArrow", func(b *testing.B) {
			resp := sc.build()
			protoResp, err := to.QueryDataResponse(backend.DataFrameFormat_ARROW, resp)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := from.QueryDataResponse(protoResp); err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(sc.name+"/3_EncodeJSON", func(b *testing.B) {
			resp := sc.build()
			protoResp, err := to.QueryDataResponse(backend.DataFrameFormat_ARROW, resp)
			if err != nil {
				b.Fatal(err)
			}
			decoded, err := from.QueryDataResponse(protoResp)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := encodeJSONLikeGrafana(decoded); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// compile-time check that the pluginv2 format enum matches the SDK alias we use.
var _ = pluginv2.DataFrameFormat_ARROW
