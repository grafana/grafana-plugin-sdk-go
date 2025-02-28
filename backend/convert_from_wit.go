package backend

import (
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genwit/grafana/plugins/types"
)

type ConvertFromWit struct{}

func FromWit() ConvertFromWit {
	return ConvertFromWit{}
}

func (f ConvertFromWit) PluginContext(wit types.PluginContext) PluginContext {
	ctx := PluginContext{
		OrgID: wit.OrgID,
		// PluginID:                   wit.PluginId,
		// PluginVersion:              wit.PluginVersion,
		// APIVersion:                 wit.ApiVersion,
		// User:                       c.User(wit.User),
		// AppInstanceSettings:        c.AppInstanceSettings(wit.AppInstanceSettings),
		// DataSourceInstanceSettings: c.DataSourceInstanceSettings(wit.DataSourceInstanceSettings),
		// GrafanaConfig:              c.GrafanaConfig(wit.GrafanaConfig),
		// UserAgent:                  c.UserAgent(wit.UserAgent),
	}
	return ctx
}

func (f ConvertFromWit) QueryDataResponse(wit types.QueryDataResponse) (*QueryDataResponse, error) {
	qdr := &QueryDataResponse{
		Responses: make(Responses, wit.Responses.Len()),
	}
	for _, r := range wit.Responses.Slice() {
		refID := r.F0
		res := r.F1

		frameBytes := make([][]byte, res.Frames.Len())
		for i, f := range res.Frames.Slice() {
			frameBytes[i] = f.Slice()
		}

		frames, err := data.UnmarshalArrowFrames(frameBytes)
		if err != nil {
			return nil, err
		}

		// TODO: add status
		// status := Status(res.Status)
		// if !status.IsValid() {
		// 	status = StatusUnknown
		// }

		dr := DataResponse{
			Frames: frames,
			// Status: status,
		}
		// TODO: add error
		// if res.Error != "" {
		// 	dr.Error = err
		// 	dr.ErrorSource = ErrorSource(res.ErrorSource)
		// }
		qdr.Responses[refID] = dr

	}
	return qdr, nil
}

func (f ConvertFromWit) QueryDataRequest(wit types.QueryDataRequest) QueryDataRequest {
	return QueryDataRequest{
		PluginContext: f.PluginContext(wit.PluginContext),
	}
}
