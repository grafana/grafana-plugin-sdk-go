package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// ResponseSizeStatName is the QueryStat.DisplayName ResponseSizeMiddleware writes.
const ResponseSizeStatName = "Response size"

// ResponseSizeMiddleware adds each query's Arrow-encoded response size, in bytes, as a
// data.QueryStat on its first frame. It costs one extra encode per response. A query
// whose frames fail to encode gets no stat rather than an understated one.
func ResponseSizeMiddleware() HandlerMiddleware {
	return HandlerMiddlewareFunc(func(next Handler) Handler {
		return &responseSizeHandler{BaseHandler: NewBaseHandler(next)}
	})
}

type responseSizeHandler struct {
	BaseHandler
}

func (h *responseSizeHandler) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	resp, err := h.BaseHandler.QueryData(ctx, req)
	if resp == nil {
		return resp, err
	}
	for _, dr := range resp.Responses {
		if len(dr.Frames) == 0 || dr.Frames[0] == nil {
			continue
		}
		size, ok := arrowEncodedSize(dr.Frames)
		if !ok {
			continue
		}
		first := dr.Frames[0]
		if first.Meta == nil {
			first.Meta = &data.FrameMeta{}
		}
		first.Meta.Stats = append(first.Meta.Stats, data.QueryStat{
			FieldConfig: data.FieldConfig{DisplayName: ResponseSizeStatName, Unit: "bytes"},
			Value:       float64(size),
		})
	}
	return resp, err
}

func arrowEncodedSize(frames data.Frames) (int64, bool) {
	var total int64
	for _, f := range frames {
		if f == nil {
			continue
		}
		b, err := f.MarshalArrow()
		if err != nil {
			return 0, false
		}
		total += int64(len(b))
	}
	return total, true
}
