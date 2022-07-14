package numeric

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
)

const FrameTypeNumericWide = "numeric_wide"

type WideFrame struct {
	*data.Frame
}

func NewWideFrame() *WideFrame {
	return &WideFrame{emptyFrameWithTypeMD(FrameTypeNumericWide)}
}

func (wf *WideFrame) AddMetric(metricName string, l data.Labels, value interface{}) error {
	fType := data.FieldTypeFor(value)
	if !fType.Numeric() {
		return fmt.Errorf("unsupported value type %T, must be numeric", value)
	}

	if wf == nil || wf.Frame == nil {
		return fmt.Errorf("zero frames when calling AddMetric must call NewWideFrame first")
	}

	if wf.Frame == nil {
		wf.Frame = data.NewFrame("").SetMeta(&data.FrameMeta{
			Type: data.FrameType(FrameTypeNumericWide), // TODO: make type
		})
	}
	field := data.NewFieldFromFieldType(fType, 1)
	field.Name = metricName
	field.Labels = l
	field.Set(0, value)
	if len(wf.Fields) == 0 {
		wf.Fields = append(wf.Fields, field)
		return nil
	}
	wf.Fields = append(wf.Fields, field)
	return nil
}

func (wf *WideFrame) GetMetricRefs(validateData bool) ([]MetricRef, []sdata.FrameFieldIndex, error) {
	return validateAndGetRefsWide(wf, validateData)
}

// TODO: Update with current rules to match(ish) time series
func validateAndGetRefsWide(wf *WideFrame, validateData bool) ([]MetricRef, []sdata.FrameFieldIndex, error) {
	if validateData {
		panic("validateData option is not implemented")
	}
	refs := []MetricRef{}
	for _, field := range wf.Fields {
		if !field.Type().Numeric() {
			continue
		}
		refs = append(refs, MetricRef{field})
	}
	sortNumericMetricRef(refs)
	return refs, nil, nil
}

func (wf *WideFrame) Validate() (isEmpty bool, errors []error) {
	panic("not implemented")
}

func (wf *WideFrame) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {
	panic("not implemented")
}
