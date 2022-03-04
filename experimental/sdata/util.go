package sdata

import "github.com/grafana/grafana-plugin-sdk-go/data"

func frameHasMetaType(f *data.Frame, t data.FrameType) bool {
	return f != nil && f.Meta != nil && f.Meta.Type == t
}
