package dataframe

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

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

// TimeSeriesSchema returns the TimeSeriesSchema of the frame. The TimeSeriesSchema's Type
// value will be TimeSeriesNot if it is not a time series.
func (f *Frame) TimeSeriesSchema() (tsSchema TimeSeriesSchema) {
	tsSchema.Type = TimeSeriesTypeNot
	if f.Fields == nil || len(f.Fields) == 0 {
		return
	}

	timeIndices := f.TypeIndices(VectorPTypeTime, VectorPTypeNullableTime)
	if len(timeIndices) != 1 {
		return
	}
	tsSchema.TimeIndex = timeIndices[0]
	tsSchema.TimeIsNullable = f.Fields[tsSchema.TimeIndex].Vector.PrimitiveType().Nullable()

	tsSchema.ValueIndices = f.TypeIndices(NumericVectorPTypes()...)
	if len(tsSchema.ValueIndices) == 0 {
		return
	}

	tsSchema.FactorIndices = f.TypeIndices(VectorPTypeString, VectorPTypeNullableString)

	// Extra Columns not Allowed
	if 1+len(tsSchema.ValueIndices)+len(tsSchema.FactorIndices) != len(f.Fields) {
		return
	}

	if len(tsSchema.FactorIndices) == 0 {
		tsSchema.Type = TimeSeriesTypeWide
		return
	}
	tsSchema.Type = TimeSeriesTypeLong
	return
}

// LongToWide converts a Long formated time series Frame to a Wide format.
// The input series must be sorted ascending by time.
func LongToWide(inFrame *Frame) (*Frame, error) {
	tsSchema := inFrame.TimeSeriesSchema()
	if tsSchema.Type != TimeSeriesTypeLong {
		return nil, fmt.Errorf("can not convert to wide series, expected long format series input but got %s series", tsSchema.Type)
	}

	inLen, err := inFrame.RowLen()
	if err != nil {
		return nil, err
	}
	if inLen == 0 {
		return nil, fmt.Errorf("can not convert to wide series, input fields have no rows")
	}

	newFrame := New(inFrame.Name, NewField(inFrame.Fields[tsSchema.TimeIndex].Name, nil, []time.Time{}))
	newFrameRowCounter := 0

	timeAt := func(idx int) (time.Time, error) { // get time.Time regardless if pointer
		if tsSchema.TimeIsNullable {
			timePtr := inFrame.At(tsSchema.TimeIndex, idx).(*time.Time)
			if timePtr == nil {
				return time.Time{}, fmt.Errorf("can not convert to wide series, input has null time values")
			}
			return *timePtr, nil
		}
		return inFrame.At(tsSchema.TimeIndex, idx).(time.Time), nil
	}

	lastTime, err := timeAt(0)
	newFrame.Fields[0].Vector.Append(lastTime) // set initial time value
	if err != nil {
		return nil, err
	}

	seenFactors := map[string]struct{}{}                        // seen factor combinations
	valueIdxFactorKeyToFieldIdx := make(map[int]map[string]int) // value key and factors -> fieldIdx of newFrame
	for _, i := range tsSchema.ValueIndices {                   // initialize nested maps
		valueIdxFactorKeyToFieldIdx[i] = make(map[string]int)
	}

	for rowIdx := 0; rowIdx < inLen; rowIdx++ { // loop over each Row of inFrame
		currentTime, err := timeAt(rowIdx)
		if err != nil {
			return nil, err
		}

		if currentTime.After(lastTime) { // time advance means new row in newFrame
			newFrameRowCounter++
			lastTime = currentTime
			for _, field := range newFrame.Fields {
				// extend all Field Vectors for new row. If no value found will have zero value
				field.Vector.Extend(1)
			}
			newFrame.Set(0, newFrameRowCounter, currentTime)
		}
		if currentTime.Before(lastTime) {
			return nil, fmt.Errorf("long series must be sorted ascending by time to be converted")
		}

		sliceKey := make([][2]string, len(tsSchema.FactorIndices)) // Factor Columns idx:value tuples
		namedKey := make([][2]string, len(tsSchema.FactorIndices)) // Factor Columns name:value tuples (Used for Labels)
		for i, factorIdx := range tsSchema.FactorIndices {
			rawVal := inFrame.At(factorIdx, rowIdx)
			var val string // null and empty factor values are both treated identically - as an empty string
			switch s := rawVal.(type) {
			case string:
				val = s
			case *string:
				if s != nil {
					val = *s
				}
			}
			sliceKey[i] = [2]string{strconv.FormatInt(int64(factorIdx), 10), val}
			namedKey[i] = [2]string{inFrame.Fields[factorIdx].Name, val}
		}
		factorKeyRaw, err := json.Marshal(sliceKey)
		if err != nil {
			return nil, err
		}
		factorKey := string(factorKeyRaw)

		// Make New Fields as new Factor combinations are found
		if _, ok := seenFactors[factorKey]; !ok {
			// First index for the set of factors.
			currentFieldLen := len(newFrame.Fields)
			// New Field created for each Value Field from inFrame
			seenFactors[factorKey] = struct{}{}
			for i, vIdx := range tsSchema.ValueIndices {
				name := inFrame.Fields[tsSchema.ValueIndices[i]].Name
				pType := inFrame.Fields[tsSchema.ValueIndices[i]].Vector.PrimitiveType()
				newVector := NewVectorFromPType(pType, newFrameRowCounter+1)
				labels, err := labelsFromTupleSlice(namedKey)
				if err != nil {
					return nil, err
				}
				newField := &Field{
					// Note: currently duplicate names won't marshal to Arrow (https://github.com/grafana/grafana-plugin-sdk-go/issues/59)
					Name:   name,
					Labels: labels,
					Vector: newVector,
				}
				newFrame.Fields = append(newFrame.Fields, newField)
				valueIdxFactorKeyToFieldIdx[vIdx][factorKey] = currentFieldLen + i
			}
		}
		for _, fieldIdx := range tsSchema.ValueIndices {
			newFieldIdx := valueIdxFactorKeyToFieldIdx[fieldIdx][factorKey]
			newFrame.Set(newFieldIdx, newFrameRowCounter, inFrame.CopyAt(fieldIdx, rowIdx))
		}
	}

	return newFrame, nil
}

// TimeSeriesSchema is information about a Dataframe's schema.  It is populated from
// the Frame's TimeSeriesSchema() method.
type TimeSeriesSchema struct {
	Type           TimeSeriesType
	TimeIndex      int
	TimeIsNullable bool
	ValueIndices   []int
	FactorIndices  []int
}

func labelsFromTupleSlice(tuples [][2]string) (Labels, error) {
	labels := make(map[string]string)
	for _, tuple := range tuples {
		if key, ok := labels[tuple[0]]; ok {
			return nil, fmt.Errorf("duplicate key '%v' in lables: %v", key, tuples)
		}
		labels[tuple[0]] = tuple[1]
	}
	return labels, nil
}
