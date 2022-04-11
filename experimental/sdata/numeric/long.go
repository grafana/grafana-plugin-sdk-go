package numeric

import (
	"sort"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type LongFrame struct {
	*data.Frame
}

func (lfn *LongFrame) GetMetricRefs() []MetricRef {
	if lfn == nil || lfn.Frame == nil {
		return []MetricRef{}
	}
	stringFieldIdxs, numericFieldIdxs := []int{}, []int{}
	stringFieldNames, numericFieldNames := []string{}, []string{}

	refs := []MetricRef{}

	for i, field := range lfn.Fields {
		fType := field.Type()
		switch {
		case fType.Numeric():
			numericFieldIdxs = append(numericFieldIdxs, i)
			numericFieldNames = append(numericFieldNames, field.Name)
		case fType == data.FieldTypeString || fType == data.FieldTypeNullableString:
			stringFieldIdxs = append(stringFieldIdxs, i)
			stringFieldNames = append(stringFieldNames, field.Name)
		}
	}

	for rowIdx := 0; rowIdx < lfn.Rows(); rowIdx++ {
		l := data.Labels{}
		for i := range stringFieldIdxs {
			key := stringFieldNames[i]
			val, _ := lfn.ConcreteAt(stringFieldIdxs[i], rowIdx)
			l[key] = val.(string)
		}

		for i, fieldIdx := range numericFieldIdxs {
			fType := lfn.Fields[fieldIdx].Type()
			field := data.NewFieldFromFieldType(fType, 1)
			field.Name = numericFieldNames[i]
			field.Labels = l
			field.Set(0, lfn.Fields[fieldIdx].At(rowIdx))
			refs = append(refs, MetricRef{
				ValueField: field,
			})
		}
	}
	sortNumericMetricRef(refs)
	return refs
}

func (lfn *LongFrame) Validate() (isEmpty bool, errors []error) {
	panic("not implemented")
}

func sortNumericMetricRef(refs []MetricRef) {
	sort.SliceStable(refs, func(i, j int) bool {
		iRef := refs[i]
		jRef := refs[j]

		if iRef.GetMetricName() < jRef.GetMetricName() {
			return true
		}
		if iRef.GetMetricName() > jRef.GetMetricName() {
			return false
		}

		// If here Names are equal, next sort based on if there are labels.

		if iRef.GetLabels() == nil && jRef.GetLabels() == nil {
			return true // no labels first
		}
		if iRef.GetLabels() == nil && jRef.GetLabels() != nil {
			return true
		}
		if iRef.GetLabels() != nil && jRef.GetLabels() == nil {
			return false
		}

		return iRef.GetLabels().String() < jRef.GetLabels().String()
	})
}
