package numeric

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

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
