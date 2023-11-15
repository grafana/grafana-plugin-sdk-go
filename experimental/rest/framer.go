package rest

import (
	"strconv"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/framestruct"
)

type Data map[string]any

// JSONFramer converts a json result to a data frame
type JSONFramer struct {
	data []Data
	name string
	opts []framestruct.FramestructOption
}

// Frames implements the interface
func (rf *JSONFramer) Frames() (data.Frames, error) {
	rows := rf.flattenResults()
	return framestruct.ToDataFrames(rf.name, rows, rf.opts...)
}

func (rf *JSONFramer) flattenResults() []Data {
	rows := []Data{}
	for _, r := range rf.data {
		var flat = Data{}
		flattenRow("", r, flat)
		rows = append(rows, flat)
	}
	return rows
}

func flattenRow(prefix string, src Data, dest Data) {
	if len(prefix) > 0 {
		prefix += "."
	}
	for k, v := range src {
		switch child := v.(type) {
		case map[string]any:
			flattenRow(prefix+k, child, dest)
		case []any:
			for i := 0; i < len(child); i++ {
				dest[prefix+k+"."+strconv.Itoa(i)] = child[i]
			}
		default:
			dest[prefix+k] = v
		}
	}
}
