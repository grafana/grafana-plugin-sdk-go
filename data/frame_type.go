package data

// A FrameType string, when present in a frame's metadata, asserts that the
// frame's structure conforms to the FrameType's specification.
// This property is currently optional, so FrameType may be FrameTypeUnknown even if the properties of
// the Frame correspond to a defined FrameType.
type FrameType string

// ---
// Docs Note: Constants need to be on their own line for links to work with the pkgsite docs.
// ---

// FrameTypeUnknown indicates that we do not know the frame type
const FrameTypeUnknown FrameType = ""

// FrameTypeTimeSeriesWide uses labels on fields to define dimensions and is documented in [Time Series Wide Format in the Data Plane Contract]. There is additional documentation in the [Developer Data Frame Documentation on the Wide Format].
//
// [Time Series Wide Format in the Data Plane Contract]: https://github.com/grafana/grafana-plugin-sdk-go/blob/main/data/contract_docs/timeseries.md#time-series-wide-format-timeserieswide
// [Developer Data Frame Documentation on the Wide Format]: https://grafana.com/docs/grafana/latest/developers/plugins/data-frames/#wide-format
const FrameTypeTimeSeriesWide FrameType = "timeseries-wide"

// FrameTypeTimeSeriesLong uses string fields to define dimensions and is documented in [Time Series Long Format in the Data Plane Contract]. There is additional documentation in the [Developer Data Frame Documentation on Long Format].
//
// [Time Series Long Format in the Data Plane Contract]: https://github.com/grafana/grafana-plugin-sdk-go/blob/main/data/contract_docs/timeseries.md#time-series-long-format-timeserieslong-sql-like
// [Developer Data Frame Documentation on Long Format]: https://grafana.com/docs/grafana/latest/developers/plugins/data-frames/#long-format
const FrameTypeTimeSeriesLong FrameType = "timeseries-long"

// FrameTypeTimeSeriesMany is the same as "Wide" with exactly one numeric value field
// Deprecated: use FrameTypeTimeSeriesMulti instead.
const FrameTypeTimeSeriesMany FrameType = "timeseries-many"

// FrameTypeTimeSeriesMulti is documented in the [Time Series Multi Format in the Data Plane Contract]
// This replaces FrameTypeTimeSeriesMany.
//
// [Time Series Multi Format in the Data Plane Contract]: https://github.com/grafana/grafana-plugin-sdk-go/blob/main/data/contract_docs/timeseries.md#time-series-multi-format-timeseriesmulti
const FrameTypeTimeSeriesMulti FrameType = "timeseries-multi"

// FrameTypeDirectoryListing represents the items in a directory
// field[0]:
//  * name
//  * new paths can be constructed from the parent path + separator + name
// field[1]:
//  * media-type
//  * when "directory" it can be nested
const FrameTypeDirectoryListing FrameType = "directory-listing"

// FrameTypeTable represents an arbitrary table structure with no constraints.
const FrameTypeTable FrameType = "table"

// FrameTypeNumericWide is documented in the [Numeric Wide Format in the Data Plane Contract].
//
// [Numeric Wide Format in the Data Plane Contract]: https://github.com/grafana/grafana-plugin-sdk-go/blob/main/data/contract_docs/numeric.md#numeric-wide-format-numericwide
const FrameTypeNumericWide FrameType = "numeric-wide"

// FrameTypeNumericMulti is documented in the [Numeric Multi Format in the Data Plane Contract].
//
// [Numeric Multi Format in the Data Plane Contract]: https://github.com/grafana/grafana-plugin-sdk-go/blob/main/data/contract_docs/numeric.md#numeric-multi-format-numericmulti
const FrameTypeNumericMulti FrameType = "numeric-multi"

// FrameTypeNumericLong is documented in the [Numeric Long Format in the Data Plane Contract].
//
// [Numeric Long Format in the Data Plane Contract]: https://github.com/grafana/grafana-plugin-sdk-go/blob/main/data/contract_docs/numeric.md#numeric-long-format-numericlong-sql-table-like
const FrameTypeNumericLong FrameType = "numeric-long"

// Soon?
// "timeseries-wide-ohlc" -- known fields for open/high/low/close
// "histogram" -- BucketMin, BucketMax, values...
// "trace" -- ??
// "node-graph-nodes"
// "node-graph-edges"

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

// IsTimeSeries checks if the type represents a timeseries
func (p FrameType) IsTimeSeries() bool {
	switch p {
	case
		FrameTypeTimeSeriesWide,
		FrameTypeTimeSeriesLong,
		FrameTypeTimeSeriesMany:
		return true
	}
	return false
}
