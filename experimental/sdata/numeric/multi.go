package numeric

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
)

const FrameTypeNumericMulti = "numeric_multi"

type MultiFrame []*data.Frame

func NewMultiFrame() *MultiFrame {
	return &MultiFrame{
		emptyFrameWithTypeMD(FrameTypeNumericMulti),
	}
}

func (mf *MultiFrame) AddMetric(metricName string, l data.Labels, value interface{}) error {
	fType := data.FieldTypeFor(value)
	if !fType.Numeric() {
		return fmt.Errorf("unsupported values type %T, must be numeric", value)
	}
	if mf == nil || len(*mf) == 0 {
		return fmt.Errorf("zero frames when calling AddMetric must call NewMultiFrame first")
	}

	field := data.NewFieldFromFieldType(fType, 1)
	field.Name = metricName
	field.Labels = l
	field.Set(0, value)

	if len(*mf) == 1 && len((*mf)[0].Fields) == 0 {
		(*mf)[0].Fields = append((*mf)[0].Fields, field)
		return nil
	}

	*mf = append(*mf, data.NewFrame("", field).SetMeta(&data.FrameMeta{
		Type: data.FrameType(FrameTypeNumericMulti), // TODO: make type
	}))

	return nil
}

func (mf *MultiFrame) GetMetricRefs() ([]MetricRef, []sdata.FrameFieldIndex, error) {
	return validateAndGetRefsMulti(mf, true)
}

func (mf *MultiFrame) Validate() (isEmpty bool, errors []error) {
	panic("not implemented")
}

func (mf *MultiFrame) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {
	panic("not implemented")
}

/*
Rules:
- Whenever an error is returned, there are no ignored fields returned
- Must have at least one frame
- The first frame must be valid or will error, additional invalid frames with the type indicator will error,
    frames without type indicator are ignored
- A valid individual Frame (in the non empty case) has a numeric field and a type indicator
- Any nil Frames or Fields will cause an error (e.g. [Frame, Frame, nil, Frame] or [nil])
- If any frame has fields within the frame of different lengths, an error will be returned
- If validateData is true, duplicate metricName+Labels will error
- If all frames and their fields are ignored, and it is not the empty response case, an error is returned

Things to decide:
 - Seems like allowing (ignoring) more than 1 row is not a good idea (outside of Long)
 - Will allow for extra frames

TODO: Change this to follow the above
*/
func validateAndGetRefsMulti(mf *MultiFrame, getRefs bool) ([]MetricRef, []sdata.FrameFieldIndex, error) {
	refs := []MetricRef{}
	for _, frame := range *mf {
		valueFields := frame.TypeIndices(sdata.ValidValueFields()...)
		if len(valueFields) == 0 {
			continue
		}
		refs = append(refs, MetricRef{frame.Fields[valueFields[0]]})
	}
	sortNumericMetricRef(refs)
	return refs, nil, nil
}
