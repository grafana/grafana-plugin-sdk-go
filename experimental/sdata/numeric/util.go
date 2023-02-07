package numeric

import "github.com/grafana/grafana-plugin-sdk-go/data"

func emptyFrameWithTypeMD(t data.FrameType, v data.FrameTypeVersion) *data.Frame {
	return data.NewFrame("").SetMeta(&data.FrameMeta{Type: t, TypeVersion: v})
}

func frameHasType(f *data.Frame, t data.FrameType) bool {
	return f != nil && f.Meta != nil && f.Meta.Type == t
}
