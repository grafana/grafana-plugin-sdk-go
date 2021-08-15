package data

// FrameType indicates the frame structure
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
	FrameTypeTimeSeriesWide = "timeseries-wide"

	// FrameTypeTimeSeriesLong has at least two fields:
	// field[0]:
	//  * type time
	//  * ascending values
	//  * duplicate times used for different dimensions
	// field[1..n]:
	//  * string fields convert to labels
	FrameTypeTimeSeriesLong = "timeseries-long"

	// FrameTypeTimeSeriesMany has exacty two fields
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
