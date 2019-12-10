package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend/common"
	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/grafana_plugin"
)

// DataQueryHandler handles data source queries.
type DataQueryHandler interface {
	DataQuery(ctx context.Context, pc common.PluginConfig, headers map[string]string, queries []common.DataQuery) (common.DataQueryResponse, error)
}

func (p *backendPluginWrapper) DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error) {

	pc := common.PluginConfigFromProto(req.Config)

	queries := make([]common.DataQuery, len(req.Queries))
	for i, q := range req.Queries {
		queries[i] = *common.DataQueryFromProtobuf(q)
	}

	resp, err := p.handlers.DataQuery(ctx, pc, req.Headers, queries)
	if err != nil {
		return nil, err
	}

	encodedFrames := make([][]byte, len(resp.Frames))
	for i, frame := range resp.Frames {
		encodedFrames[i], err = dataframe.MarshalArrow(frame)
		if err != nil {
			return nil, err
		}
	}

	return &bproto.DataQueryResponse{
		Frames:   encodedFrames,
		Metadata: resp.Metadata,
	}, nil
}
