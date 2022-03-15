package sdata

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func emptyFrameWithTypeMD(t data.FrameType) *data.Frame {
	return data.NewFrame("").SetMeta(&data.FrameMeta{Type: t})
}

func frameHasMetaType(f *data.Frame, t data.FrameType) bool {
	return f != nil && f.Meta != nil && f.Meta.Type == t
}

func timeIsSorted(field *data.Field) (bool, error) {
	switch {
	case field == nil:
		return false, fmt.Errorf("field is nil")
	case field.Type() != data.FieldTypeTime:
		return false, fmt.Errorf("field is not a time field")
	case field.Len() == 0:
		return true, nil
	}

	for tIdx := 1; tIdx < field.Len(); tIdx++ {
		prevTime := field.At(tIdx - 1).(time.Time)
		curTime := field.At(tIdx).(time.Time)
		if curTime.Before(prevTime) {
			return false, nil
		}
	}
	return true, nil
}
