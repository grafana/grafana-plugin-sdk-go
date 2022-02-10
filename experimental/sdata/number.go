package sdata

import (
	"fmt"
	"sort"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type NumericCollectionWriter interface {
	AddMetric(metricName string, l data.Labels, value interface{}) error
	SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig)
}

type NumericCollection interface {
	NumericCollectionWriter
	NumericCollectionReader
}

type NumericCollectionReader interface {
	Validate() (isEmpty bool, errors []error)
	GetMetricRefs() []NumericMetricRef
}

type NumericMetricRef struct {
	ValueField *data.Field
}

func (n NumericMetricRef) GetMetricName() string {
	if n.ValueField != nil {
		return n.ValueField.Name
	}
	return ""
}

func (n NumericMetricRef) GetLabels() data.Labels {
	if n.ValueField != nil {
		return n.ValueField.Labels
	}
	return nil
}

type MultiFrameNumeric []*data.Frame

func (mfn *MultiFrameNumeric) AddMetric(metricName string, l data.Labels, value interface{}) error {
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

func (mfn *MultiFrameNumeric) GetMetricRefs() []NumericMetricRef {
	refs := []NumericMetricRef{}
	for _, frame := range *mfn {
		valueFields := frame.TypeIndices(ValidValueFields()...)
		if len(valueFields) == 0 {
			continue
		}
		refs = append(refs, NumericMetricRef{frame.Fields[valueFields[0]]})
	}
	sortNumericMetricRef(refs)
	return refs
}

func (mfn *MultiFrameNumeric) Validate() (isEmpty bool, errors []error) {
	panic("not implemented")
}

func (mfn *MultiFrameNumeric) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {
	panic("not implemented")
}

type WideFrameNumeric struct {
	*data.Frame
}

func (mfn *WideFrameNumeric) AddMetric(metricName string, l data.Labels, value interface{}) error {
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

func (mfn *WideFrameNumeric) GetMetricRefs() []NumericMetricRef {
	refs := []NumericMetricRef{}
	for _, field := range mfn.Fields {
		if !field.Type().Numeric() {
			continue
		}
		refs = append(refs, NumericMetricRef{field})
	}
	sortNumericMetricRef(refs)
	return refs
}

func (mfn *WideFrameNumeric) Validate() (isEmpty bool, errors []error) {
	panic("not implemented")
}

func (mfn *WideFrameNumeric) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {
	panic("not implemented")
}

type LongFrameNumeric struct {
	*data.Frame
}

func (lfn *LongFrameNumeric) GetMetricRefs() []NumericMetricRef {
	if lfn == nil || lfn.Frame == nil {
		return []NumericMetricRef{}
	}
	stringFieldIdxs, numericFieldIdxs := []int{}, []int{}
	stringFieldNames, numericFieldNames := []string{}, []string{}

	refs := []NumericMetricRef{}

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

		for _, fieldIdx := range numericFieldIdxs {
			fType := lfn.Fields[fieldIdx].Type()
			field := data.NewFieldFromFieldType(fType, 1)
			field.Name = lfn.Fields[fieldIdx].Name
			field.Labels = l
			field.Set(0, lfn.Fields[fieldIdx].At(rowIdx))
			refs = append(refs, NumericMetricRef{
				ValueField: field,
			})
		}
	}
	sortNumericMetricRef(refs)
	return refs
}

func (lfn *LongFrameNumeric) Validate() (isEmpty bool, errors []error) {
	panic("not implemented")
}

func sortNumericMetricRef(refs []NumericMetricRef) {
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
