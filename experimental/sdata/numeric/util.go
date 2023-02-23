package numeric

import "github.com/grafana/grafana-plugin-sdk-go/data"

func emptyFrameWithTypeMD(refID string, t data.FrameType, v data.FrameTypeVersion) *data.Frame {
	f := data.NewFrame("").SetMeta(&data.FrameMeta{Type: t, TypeVersion: v})
	f.RefID = refID
	return f
}

func frameHasType(f *data.Frame, t data.FrameType) bool {
	return f != nil && f.Meta != nil && f.Meta.Type == t
}
