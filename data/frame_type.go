package data

// FrameType, when set, asserts that the frame has a structure that is valid to for corresponding FrameType. This property is currently optional, so FrameType may be FrameTypeUnknown even if the properties of the Frame correspond to a defined FrameType.
type FrameType string

const (
	// FrameTypeUnknown indicates that we do not know the field type
	FrameTypeUnknown FrameType = ""

	// FrameTypeTimeSeriesWide has at least two fields:
	// field[0]:
	//  * type time
	//  * unique ascending values
	// field[1..n]:
	//  * distinct labels may be attached to each field
	//  * numeric & boolean fields can be drawn as lines on a graph
	// See https://grafana.com/docs/grafana/latest/developers/plugins/data-frames/#wide-format
	FrameTypeTimeSeriesWide = "timeseries-wide"

	// FrameTypeTimeSeriesLong uses string fields to define dimensions.  I has at least two fields:
	// field[0]:
	//  * type time
	//  * ascending values
	//  * duplicate times exist for multiple dimensions
	// field[1..n]:
	//  * string fields define series dimensions
	//  * non-string fields define the series progression
	// See https://grafana.com/docs/grafana/latest/developers/plugins/data-frames/#long-format
	FrameTypeTimeSeriesLong = "timeseries-long"

	// FrameTypeTimeSeriesMany is the same as "Wide" with exactly one numeric value field
	// field[0]:
	//  * type time
	//  * ascending values
	// field[1]:
	//  * number field
	//  * labels represent the series dimensions
	// This structure is typically part of a list of frames with the same structure
	FrameTypeTimeSeriesMany = "timeseries-many"

	// Soon?
	// "timeseries-wide-ohlc" -- known fields for open/high/low/close
	// "histogram" -- BucketMin, BucketMax, values...
	// "directory-listing" -- known fields for name, size, mime-type, modified, etc
	// "trace" -- ??
	// "node-graph-nodes"
	// "node-graph-edges"
)

// IsKnownType checks if the value is a known structure
func (p FrameType) IsKnownType() bool {
	switch p {
	case
		FrameTypeTimeSeriesWide,
		FrameTypeTimeSeriesLong,
		FrameTypeTimeSeriesMany:
		return true
	}
	return false
}

// FrameTypes returns a slice of all known frame types
func FrameTypes() []FrameType {
	return []FrameType{
		FrameTypeTimeSeriesWide,
		FrameTypeTimeSeriesLong,
		FrameTypeTimeSeriesMany,
	}
}
