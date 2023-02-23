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

func (lf *LongFrame) Frames() data.Frames {
	return data.Frames{lf.Frame}
}

var LongFrameVersionLatest = LongFrameVersions()[len(LongFrameVersions())-1]

func LongFrameVersions() []data.FrameTypeVersion {
	return []data.FrameTypeVersion{{0, 1}}
}

func NewLongFrame(refID string, v data.FrameTypeVersion) (*LongFrame, error) {
	if v.Greater(LongFrameVersionLatest) {
		return nil, fmt.Errorf("can not create LongFrame of version %s because it is newer than library version %v", v, LongFrameVersionLatest)
	}
	return &LongFrame{emptyFrameWithTypeMD(refID, data.FrameTypeNumericLong, v)}, nil
}

func (lf *LongFrame) GetCollection(validateData bool) (Collection, error) {
	return validateAndGetRefsLong(lf, validateData)
}

// TODO: Update with current rules to match(ish) time series
func validateAndGetRefsLong(lf *LongFrame, validateData bool) (Collection, error) {
	var c Collection
	if validateData {
		panic("validateData option is not implemented")
	}

	if lf == nil || lf.Frame == nil {
		return c, fmt.Errorf("nil frame is invalid")
	}

	c.RefID = lf.Frame.RefID

	if !frameHasType(lf.Frame, data.FrameTypeNumericLong) {
		return c, fmt.Errorf("frame is missing %s type indicator", data.FrameTypeNumericLong)
	}

	if lf.Meta.TypeVersion != LongFrameVersionLatest {
		c.Warning = &sdata.VersionWarning{DataVersion: lf.Meta.TypeVersion, LibraryVersion: LongFrameVersionLatest, DataType: data.FrameTypeNumericLong}
	}

	stringFieldIdxs, numericFieldIdxs := []int{}, []int{}
	stringFieldNames, numericFieldNames := []string{}, []string{}

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
			c.Refs = append(c.Refs, MetricRef{
				ValueField: field,
			})
		}
	}
	sortNumericMetricRef(c.Refs)
	return c, nil
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
