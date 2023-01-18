package numeric

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data"
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

	field := data.NewFieldFromFieldType(fType, 1)
	field.Name = metricName
	field.Labels = l
	field.Set(0, value)
	wf.Fields = append(wf.Fields, field)

	return nil
}

func (wf *WideFrame) GetCollection(validateData bool) (Collection, error) {
	return validateAndGetRefsWide(wf, validateData)
}

// TODO: Update with current rules to match(ish) time series
func validateAndGetRefsWide(wf *WideFrame, validateData bool) (Collection, error) {
	if validateData {
		panic("validateData option is not implemented")
	}

	var c Collection

	for _, field := range wf.Fields {
		if !field.Type().Numeric() {
			continue
		}
		c.Refs = append(c.Refs, MetricRef{field})
	}
	sortNumericMetricRef(c.Refs)
	return c, nil
}

func (wf *WideFrame) Validate() (isEmpty bool, errors []error) {
	panic("not implemented")
}

func (wf *WideFrame) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {
	panic("not implemented")
}
