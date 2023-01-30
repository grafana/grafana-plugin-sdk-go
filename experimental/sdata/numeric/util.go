package numeric

import "github.com/grafana/grafana-plugin-sdk-go/data"

func emptyFrameWithTypeMD(t data.FrameType, v data.FrameTypeVersion) *data.Frame {
	return data.NewFrame("").SetMeta(&data.FrameMeta{Type: t, TypeVersion: &v})
}
