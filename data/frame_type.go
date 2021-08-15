package data

// FrameType indicates the frame structure
type FrameType string

const (
	// FrameTypeUnknown indicates that we do not know the field type
	FrameTypeUnknown FrameType = ""

	// FrameTypeTimeSeriesWide ...
	FrameTypeTimeSeriesWide = "timeseries-wide"

	// FrameTypeTimeSeriesLong ...
	FrameTypeTimeSeriesLong = "timeseries-long"

	// FrameTypeTimeSeriesMany ...
	FrameTypeTimeSeriesMany = "timeseries-many"

	// // FrameTypeOHLC ... (known fields for open/high/low/close/ volume)
	// FrameTypeOHLC = "timeseries-wide-ohlc"

	// // FrameTypeHistogram ...
	// FrameTypeHistogram = "histogram"

	// // FrameTypeDirectoryListing ...
	// FrameTypeDirectoryListing = "directory-listing"

	// // Trace ...
	// Trace = "trace"
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
