package dataframe

// TimeSeriesType represents the type of time series the schema can be treated as (if any).
type TimeSeriesType int

// TODO: Create and link to Grafana documentation on Long vs Wide
const (
	// TimeSeriesTypeNot means this dataframe is not a valid time series.
	TimeSeriesTypeNot TimeSeriesType = iota
	// TimeSeriesTypeLong means this dataframe can be treated as a "Long" time series. TODO link (see above).
	TimeSeriesTypeLong
	// TimeSeriesTypeLong means this dataframe can be treated as a "Wide" time series. TODO link (see above).
	TimeSeriesTypeWide
)

func (t TimeSeriesType) String() string {
	switch t {
	case TimeSeriesTypeLong:
		return "long"
	case TimeSeriesTypeWide:
		return "wide"
	}
	return "not"
}

// TimeSeriesType returns the TimeSeriesType of the frame. The value will be
// TimeSeriesNot if it is not a time series.
func (f *Frame) TimeSeriesType() TimeSeriesType {
	if f.Fields == nil || len(f.Fields) == 0 {
		return TimeSeriesTypeNot
	}

	timeIndices := f.TypeIndices(VectorPTypeTime, VectorPTypeNullableTime)
	if len(timeIndices) != 1 {
		return TimeSeriesTypeNot
	}

	valueIndices := f.TypeIndices(NumericVectorPTypes()...)
	if len(valueIndices) == 0 {
		return TimeSeriesTypeNot
	}

	factorIndices := f.TypeIndices(VectorPTypeString, VectorPTypeNullableString)

	// Extra Columns not Allowed
	if len(timeIndices)+len(valueIndices)+len(factorIndices) != len(f.Fields) {
		return TimeSeriesTypeNot
	}

	if len(factorIndices) == 0 {
		return TimeSeriesTypeWide
	}
	return TimeSeriesTypeLong
}