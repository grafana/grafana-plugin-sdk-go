package dataframe

import (
	"encoding/json"
	"fmt"
	"sort"
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
	tsSchema.TimeIsNullable = f.Fields[tsSchema.TimeIndex].Nullable()

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
func LongToWide(longFrame *Frame) (*Frame, error) {
	tsSchema := longFrame.TimeSeriesSchema()
	if tsSchema.Type != TimeSeriesTypeLong {
		return nil, fmt.Errorf("can not convert to wide series, expected long format series input but got %s series", tsSchema.Type)
	}

	longLen, err := longFrame.RowLen()
	if err != nil {
		return nil, err
	} else if longLen == 0 {
		return nil, fmt.Errorf("can not convert to wide series, input fields have no rows")
	}

	wideFrame := New(longFrame.Name, NewField(longFrame.Fields[tsSchema.TimeIndex].Name, nil, []time.Time{}))
	wideFrameRowCounter := 0

	seenFactors := map[string]struct{}{}                  // seen factor combinations
	valueFactorToFieldIdx := make(map[int]map[string]int) // value field idx and factors key -> fieldIdx of longFrame (for insertion)
	for _, i := range tsSchema.ValueIndices {             // initialize nested maps
		valueFactorToFieldIdx[i] = make(map[string]int)
	}

	timeAt := func(idx int) (time.Time, error) { // get time.Time regardless if pointer
		val, ok := longFrame.ConcreteAt(tsSchema.TimeIndex, idx)
		if !ok {
			return time.Time{}, fmt.Errorf("can not convert to wide series, input has null time values")
		}
		return val.(time.Time), nil
	}
	lastTime, err := timeAt(0) // set initial time value
	if err != nil {
		return nil, err
	}
	wideFrame.Fields[0].Append(lastTime)

	for rowIdx := 0; rowIdx < longLen; rowIdx++ { // loop over each row of longFrame
		currentTime, err := timeAt(rowIdx)
		if err != nil {
			return nil, err
		}

		if currentTime.After(lastTime) { // time advance means new row in wideFrame
			wideFrameRowCounter++
			lastTime = currentTime
			for _, field := range wideFrame.Fields {
				// extend all wideFrame Field Vectors for new row. If no value found, it will have zero value
				field.Extend(1)
			}
			wideFrame.Set(0, wideFrameRowCounter, currentTime)
		}

		if currentTime.Before(lastTime) {
			return nil, fmt.Errorf("long series must be sorted ascending by time to be converted")
		}

		sliceKey := make(tupleLabels, len(tsSchema.FactorIndices)) // factor columns idx:value tuples (used for lookup)
		namedKey := make(tupleLabels, len(tsSchema.FactorIndices)) // factor columns name:value tuples (used for labels)

		// build labels
		for i, factorIdx := range tsSchema.FactorIndices {
			val, _ := longFrame.ConcreteAt(factorIdx, rowIdx)
			sliceKey[i] = tupleLabel{strconv.FormatInt(int64(factorIdx), 10), val.(string)}
			namedKey[i] = tupleLabel{longFrame.Fields[factorIdx].Name, val.(string)}
		}
		factorKey, err := sliceKey.MapKey()
		if err != nil {
			return nil, err
		}

		// make new Fields as new factor combinations are found
		if _, ok := seenFactors[factorKey]; !ok {
			currentFieldLen := len(wideFrame.Fields) // first index for the set of factors.
			seenFactors[factorKey] = struct{}{}
			for i, vIdx := range tsSchema.ValueIndices {
				// a new Field is created for each value Field from inFrame
				labels, err := tupleLablesToLabels(namedKey)
				if err != nil {
					return nil, err
				}
				longField := longFrame.Fields[tsSchema.ValueIndices[i]]
				newWideField := &Field{
					Name:   longField.Name, // Note: currently duplicate names won't marshal to Arrow (https://github.com/grafana/grafana-plugin-sdk-go/issues/59)
					Labels: labels,
					Vector: NewVectorFromPType(longField.PrimitiveType(), wideFrameRowCounter+1),
				}
				wideFrame.Fields = append(wideFrame.Fields, newWideField)
				valueFactorToFieldIdx[vIdx][factorKey] = currentFieldLen + i
			}
		}
		for _, fieldIdx := range tsSchema.ValueIndices {
			newFieldIdx := valueFactorToFieldIdx[fieldIdx][factorKey]
			wideFrame.Set(newFieldIdx, wideFrameRowCounter, longFrame.CopyAt(fieldIdx, rowIdx))
		}
	}

	return wideFrame, nil
}

// WideToLong converts a Wide formated time series Frame to a Long formated time series Frame.
func WideToLong(wideFrame *Frame) (*Frame, error) {
	tsSchema := wideFrame.TimeSeriesSchema()
	if tsSchema.Type != TimeSeriesTypeWide {
		return nil, fmt.Errorf("can not convert to long series, expected wide format series input but got %s series", tsSchema.Type)
	}

	wideLen, err := wideFrame.RowLen()
	if err != nil {
		return nil, err
	} else if wideLen == 0 {
		return nil, fmt.Errorf("can not convert to long series, input fields have no rows")
	}

	uniqueValueNames := []string{}
	uniqueValueNamesToType := make(map[string]VectorPType) // identify unique value columns by their name
	// labels become string Fields, where the label keys are Field Names
	uniqueFactorKeys := make(map[string]struct{})
	labelKeyToWideIndices := make(map[string][]int)
	sortedUniqueLabelKeys := []string{}

	for _, vIdx := range tsSchema.ValueIndices { // all columns should be value columns except time
		wideField := wideFrame.Fields[vIdx]
		if pType, ok := uniqueValueNamesToType[wideField.Name]; ok {
			if wideField.PrimitiveType() != pType {
				return nil, fmt.Errorf("two fields in input frame may not have the same name but different types, field name %s has type %s but also type %s and field idx %v", wideField.Name, pType, wideField.PrimitiveType(), vIdx)
			}
		} else {
			uniqueValueNamesToType[wideField.Name] = wideField.PrimitiveType()
			uniqueValueNames = append(uniqueValueNames, wideField.Name)
		}

		tKey, err := labelsTupleKey(wideField.Labels)
		if err != nil {
			return nil, err
		}
		labelKeyToWideIndices[tKey] = append(labelKeyToWideIndices[tKey], vIdx)

		if wideField.Labels != nil {
			for k := range wideField.Labels {
				uniqueFactorKeys[k] = struct{}{}
			}
		}
	}

	for k := range labelKeyToWideIndices {
		sortedUniqueLabelKeys = append(sortedUniqueLabelKeys, k)
	}
	sort.Strings(sortedUniqueLabelKeys)

	sort.Strings(uniqueValueNames)
	uniqueFactorNames := make([]string, 0, len(uniqueFactorKeys))
	for k := range uniqueFactorKeys {
		uniqueFactorNames = append(uniqueFactorNames, k)
	}
	sort.Strings(uniqueFactorNames)

	longFrame := New(wideFrame.Name, // time , value fields (numbers)..., factor fields (strings)...
		NewField(wideFrame.Fields[tsSchema.TimeIndex].Name, nil, []time.Time{})) // time field is first field

	i := 1
	// TODO: These need to be sorted. Since they come from maps (otherwise result order unpredictable).
	valueNameToFieldIdx := map[string]int{} // valueName -> field index of longFrame
	for _, name := range uniqueValueNames {
		longFrame.Fields = append(longFrame.Fields, &Field{ // create value (number) vectors
			Name:   name,
			Vector: NewVectorFromPType(uniqueValueNamesToType[name], 0),
		})
		valueNameToFieldIdx[name] = i
		i++
	}

	factorNameToFieldIdx := map[string]int{} // label Key -> field index for label value of longFrame
	for _, name := range uniqueFactorNames {
		longFrame.Fields = append(longFrame.Fields, NewField(name, nil, []string{})) // create factor fields
		factorNameToFieldIdx[name] = i
		i++
	}
	longFrameCounter := 0
	for rowIdx := 0; rowIdx < wideLen; rowIdx++ { // loop over each row of wideFrame
		time, ok := wideFrame.ConcreteAt(tsSchema.TimeIndex, rowIdx)
		if !ok {
			return nil, fmt.Errorf("time may not have nil values")
		}
		for _, labelKey := range sortedUniqueLabelKeys {
			longFrame.Extend(1) // grow each Fields's vector by 1
			longFrame.Set(0, longFrameCounter, time)

			for i, fieldIdx := range labelKeyToWideIndices[labelKey] {
				wideField := wideFrame.Fields[fieldIdx]
				if i == 0 {
					for k, v := range wideField.Labels {
						longFrame.Set(factorNameToFieldIdx[k], longFrameCounter, v)
					}
				}
				valueFieldIdx := valueNameToFieldIdx[wideField.Name]
				longFrame.Set(valueFieldIdx, longFrameCounter, wideFrame.CopyAt(fieldIdx, rowIdx))

			}

			longFrameCounter++
		}
	}

	return longFrame, nil
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

type tupleLabels []tupleLabel

type tupleLabel [2]string

func tupleLablesToLabels(tuples tupleLabels) (Labels, error) {
	labels := make(map[string]string)
	for _, tuple := range tuples {
		if key, ok := labels[tuple[0]]; ok {
			return nil, fmt.Errorf("duplicate key '%v' in lables: %v", key, tuples)
		}
		labels[tuple[0]] = tuple[1]
	}
	return labels, nil
}

func (t *tupleLabels) MapKey() (string, error) {
	t.SortBtKey()
	b, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (t *tupleLabels) SortBtKey() {
	if t == nil {
		return
	}
	sort.Slice((*t)[:], func(i, j int) bool {
		return (*t)[i][0] < (*t)[j][1]
	})
}

func labelsToTupleLabels(l Labels) tupleLabels {
	t := make(tupleLabels, 0, len(l))
	for k, v := range l {
		t = append(t, tupleLabel{k, v})
	}
	t.SortBtKey()
	return t
}

func labelsTupleKey(l Labels) (string, error) {
	t := labelsToTupleLabels(l)
	return t.MapKey()
}
