package numeric

import (
	"fmt"
	"sort"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
)

const FrameTypeNumericLong = "numeric_long"

type LongFrame struct {
	*data.Frame
}

func NewLongFrame() *LongFrame {
	return &LongFrame{emptyFrameWithTypeMD(FrameTypeNumericLong)}
}

func (lf *LongFrame) GetMetricRefs(validateData bool) ([]MetricRef, []sdata.FrameFieldIndex, error) {
	return validateAndGetRefsLong(lf, validateData)
}

// TODO: Update with current rules to match(ish) time series
func validateAndGetRefsLong(lf *LongFrame, validateData bool) ([]MetricRef, []sdata.FrameFieldIndex, error) {
	if validateData {
		panic("validateData option is not implemented")
	}
	if lf == nil || lf.Frame == nil {
		return nil, nil, fmt.Errorf("zero frames when calling AddMetric must call NewLongFrame first")
	}
	stringFieldIdxs, numericFieldIdxs := []int{}, []int{}
	stringFieldNames, numericFieldNames := []string{}, []string{}

	refs := []MetricRef{}

	for i, field := range lf.Fields {
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

	for rowIdx := 0; rowIdx < lf.Rows(); rowIdx++ {
		l := data.Labels{}
		for i := range stringFieldIdxs {
			key := stringFieldNames[i]
			val, _ := lf.ConcreteAt(stringFieldIdxs[i], rowIdx)
			l[key] = val.(string)
		}

		for i, fieldIdx := range numericFieldIdxs {
			fType := lf.Fields[fieldIdx].Type()
			field := data.NewFieldFromFieldType(fType, 1)
			field.Name = numericFieldNames[i]
			field.Labels = l
			field.Set(0, lf.Fields[fieldIdx].At(rowIdx))
			refs = append(refs, MetricRef{
				ValueField: field,
			})
		}
	}
	sortNumericMetricRef(refs)
	return refs, nil, nil
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
