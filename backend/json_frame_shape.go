package backend

import "github.com/grafana/grafana-plugin-sdk-go/data"

// Arrow writes a schema for each frame, so JSON costs less only for short frames. JSON decodes strings more slowly.
const jsonMaxFrameRows = 16

// jsonSuitsFrames reports whether every frame is short and has no string or JSON field, so JSON costs less than Arrow.
func jsonSuitsFrames(frames data.Frames) bool {
	for _, f := range frames {
		if f.Rows() > jsonMaxFrameRows {
			return false
		}
		for _, field := range f.Fields {
			if t := field.Type().NonNullableType(); t == data.FieldTypeString || t == data.FieldTypeJSON {
				return false
			}
		}
	}
	return true
}
