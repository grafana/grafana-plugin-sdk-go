package numeric

import "github.com/grafana/grafana-plugin-sdk-go/data"

func emptyFrameWithTypeMD(t data.FrameType) *data.Frame {
	return data.NewFrame("").SetMeta(&data.FrameMeta{Type: t})
}
