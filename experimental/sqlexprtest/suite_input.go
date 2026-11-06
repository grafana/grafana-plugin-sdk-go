package sqlexprtest

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Input returns the reports for one scenario. Pass the options it receives to
// Check, CheckResponse or CheckQueryData. A report checked with a different
// minimum Grafana version fails the scenario. A custom Input can load its
// fixtures at run time. Its errors and panics fail the scenario.
type Input func(context.Context, ...Option) (map[string]*Report, error)

// FramesInput checks frames for one RefID. The frames remain owned by the caller.
func FramesInput(refID string, frames data.Frames) Input {
	return func(_ context.Context, opts ...Option) (map[string]*Report, error) {
		r := Check(frames, opts...)
		r.RefID = refID
		return map[string]*Report{refID: r}, nil
	}
}

// ResponseInput checks a captured response against its requested RefIDs.
func ResponseInput(response *backend.QueryDataResponse, refIDs []string) Input {
	return func(_ context.Context, opts ...Option) (map[string]*Report, error) {
		return CheckResponse(response, refIDs, opts...), nil
	}
}

// QueryDataInput runs the actual plugin handler using CheckQueryData's individual
// query execution model. Supply mocks beneath the handler to test decoding and
// frame construction. Browser preparation and Grafana SQL execution are not run.
func QueryDataInput(handler backend.QueryDataHandler, request *backend.QueryDataRequest) Input {
	return func(ctx context.Context, opts ...Option) (map[string]*Report, error) {
		return CheckQueryData(ctx, handler, request, opts...)
	}
}
