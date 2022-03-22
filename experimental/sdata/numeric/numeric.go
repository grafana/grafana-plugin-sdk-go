package numeric

import (
	"fmt"
	"sort"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
)

type CollectionWriter interface {
	AddMetric(metricName string, l data.Labels, value interface{}) error
	SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig)
}

type Collection interface {
	CollectionWriter
	CollectionReader
}

type CollectionReader interface {
	Validate() (isEmpty bool, errors []error)
	GetMetricRefs() []MetricRef
}

type MetricRef struct {
	ValueField *data.Field
}

func (n MetricRef) GetMetricName() string {
	if n.ValueField != nil {
		return n.ValueField.Name
	}
	return ""
}

func (n MetricRef) GetLabels() data.Labels {
	if n.ValueField != nil {
		return n.ValueField.Labels
	}
	return nil
}

type MultiFrame []*data.Frame

func (mfn *MultiFrame) AddMetric(metricName string, l data.Labels, value interface{}) error {
	fType := data.FieldTypeFor(value)
	if !fType.Numeric() {
		return fmt.Errorf("unsupported value type %T, must be numeric", value)
	}
	field := data.NewFieldFromFieldType(fType, 1)
	field.Name = metricName
	field.Labels = l
	field.Set(0, value)
	*mfn = append(*mfn, data.NewFrame("", field).SetMeta(&data.FrameMeta{
		Type: data.FrameType("numeric_multi"), // TODO: make type
	}))
	return nil
}

func (mfn *MultiFrame) GetMetricRefs() []MetricRef {
	refs := []MetricRef{}
	for _, frame := range *mfn {
		valueFields := frame.TypeIndices(sdata.ValidValueFields()...)
		if len(valueFields) == 0 {
			continue
		}
		refs = append(refs, MetricRef{frame.Fields[valueFields[0]]})
	}
	sortNumericMetricRef(refs)
	return refs
}

func (mfn *MultiFrame) Validate() (isEmpty bool, errors []error) {
	panic("not implemented")
}

func (mfn *MultiFrame) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {
	panic("not implemented")
}

type WideFrame struct {
	*data.Frame
}

func (mfn *WideFrame) AddMetric(metricName string, l data.Labels, value interface{}) error {
	fType := data.FieldTypeFor(value)
	if !fType.Numeric() {
		return fmt.Errorf("unsupported value type %T, must be numeric", value)
	}
	if mfn.Frame == nil {
		mfn.Frame = data.NewFrame("").SetMeta(&data.FrameMeta{
			Type: data.FrameType("numeric_wide"), // TODO: make type
		})
	}
	field := data.NewFieldFromFieldType(fType, 1)
	field.Name = metricName
	field.Labels = l
	field.Set(0, value)
	mfn.Fields = append(mfn.Fields, field)
	return nil
}

func (mfn *WideFrame) GetMetricRefs() []MetricRef {
	refs := []MetricRef{}
	for _, field := range mfn.Fields {
		if !field.Type().Numeric() {
			continue
		}
		refs = append(refs, MetricRef{field})
	}
	sortNumericMetricRef(refs)
	return refs
}

func (mfn *WideFrame) Validate() (isEmpty bool, errors []error) {
	panic("not implemented")
}

func (mfn *WideFrame) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {
	panic("not implemented")
}

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
