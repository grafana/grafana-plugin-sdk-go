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
	} else if inLen == 0 {
		return nil, fmt.Errorf("can not convert to wide series, input fields have no rows")
	}

	newFrame := New(inFrame.Name, NewField(inFrame.Fields[tsSchema.TimeIndex].Name, nil, []time.Time{}))
	newFrameRowCounter := 0

	timeAt := func(idx int) (time.Time, error) { // get time.Time regardless if pointer
		val, ok := inFrame.ConcreteAt(tsSchema.TimeIndex, idx)
		if !ok {
			return time.Time{}, fmt.Errorf("can not convert to wide series, input has null time values")
		}
		return val.(time.Time), nil
	}

	// Initialize things for upcoming loop
	seenFactors := map[string]struct{}{}                        // seen factor combinations
	valueIdxFactorKeyToFieldIdx := make(map[int]map[string]int) // value key and factors -> fieldIdx of newFrame
	for _, i := range tsSchema.ValueIndices {                   // initialize nested maps
		valueIdxFactorKeyToFieldIdx[i] = make(map[string]int)
	}
	lastTime, err := timeAt(0) // set initial time value
	if err != nil {
		return nil, err
	}
	newFrame.Fields[0].Vector.Append(lastTime)

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

		sliceKey := make([][2]string, len(tsSchema.FactorIndices)) // factor columns idx:value tuples
		namedKey := make([][2]string, len(tsSchema.FactorIndices)) // factor columns name:value tuples (used for labels)

		// build labels
		for i, factorIdx := range tsSchema.FactorIndices {
			val, _ := inFrame.ConcreteAt(factorIdx, rowIdx)
			sliceKey[i] = [2]string{strconv.FormatInt(int64(factorIdx), 10), val.(string)}
			namedKey[i] = [2]string{inFrame.Fields[factorIdx].Name, val.(string)}
		}
		factorKeyRaw, err := json.Marshal(sliceKey)
		if err != nil {
			return nil, err
		}
		factorKey := string(factorKeyRaw)

		// make new Fields as new factor combinations are found
		if _, ok := seenFactors[factorKey]; !ok {
			currentFieldLen := len(newFrame.Fields) // first index for the set of factors.
			seenFactors[factorKey] = struct{}{}
			for i, vIdx := range tsSchema.ValueIndices {
				// a new Field is created for each value Field from inFrame
				labels, err := labelsFromTupleSlice(namedKey)
				if err != nil {
					return nil, err
				}
				inField := inFrame.Fields[tsSchema.ValueIndices[i]]
				newField := &Field{
					// Note: currently duplicate names won't marshal to Arrow (https://github.com/grafana/grafana-plugin-sdk-go/issues/59)
					Name:   inField.Name,
					Labels: labels,
					Vector: NewVectorFromPType(inField.Vector.PrimitiveType(), newFrameRowCounter+1),
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

// WideToLong converts a Wide formated Frame to a Long formated Frame.
func WideToLong(inFrame *Frame) (*Frame, error) {
	// TODO
	return nil, nil
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
