package data

import (
	"errors"
	"fmt"
)

// Experimental: FrameContinuation is the value of FrameMeta.Continues.
//
// A later chunk of a frame in a chunked response has only data, so a producer cannot add fields to a
// frame that it already sent. When a wide time series gets a new series after its first chunk, the
// producer starts a new frame with all the fields, and sets FrameMeta.Continues on its first chunk.
// A receiver that builds the full response joins the two frames with JoinContinuation, so that the
// response has the frame that one conversion of all rows gives. A frame can be continued only once,
// and its frame ID takes no more chunks after that.
type FrameContinuation struct {
	// FrameID is the frame ID of the earlier frame, in the same refId.
	FrameID string `json:"frameId"`

	// Fields has one entry for each field of the frame that continues. The entry is the index of the
	// same field in the earlier frame, or -1 for a new field. Each field of the earlier frame must have
	// exactly one entry.
	Fields []int `json:"fields"`

	// FillValue is the value of the new fields in the rows of the earlier frame. When it is nil, these
	// rows keep the zero value of the field type, which is null for a nullable field. LongToWide gives
	// the same values to the earlier rows of a series that starts late.
	FillValue *float64 `json:"fillValue,omitempty"`
}

// Experimental: JoinContinuation joins next into earlier, when next.Meta.Continues refers to earlier.
// It is JoinContinuationWithLimit with no limit.
func JoinContinuation(earlier, next *Frame) error {
	return JoinContinuationWithLimit(earlier, next, 0)
}

// Experimental: JoinContinuationWithLimit joins next into earlier, when next.Meta.Continues refers to
// earlier. cellLimit is a ceiling on the size of the joined Frame, in cells (rows times Fields). A
// cellLimit of 0 or less disables the ceiling. When the joined Frame is larger, it returns
// ErrorWideFrameTooLarge.
//
// Continues.Fields maps each field of next to a field of earlier, or marks it as new. In the rows of
// earlier, a new field gets Continues.FillValue, or the zero value of its type. After the join, earlier
// has the fields in the order of next, followed by the rows of next. earlier keeps its name and meta,
// and it keeps its rows without a copy. The function validates the mapping and the size before it
// allocates, and on an error earlier does not change.
func JoinContinuationWithLimit(earlier, next *Frame, cellLimit int64) error {
	if next.Meta == nil || next.Meta.Continues == nil {
		return errors.New("the frame does not continue an earlier frame")
	}
	continues := next.Meta.Continues
	earlierRows, err := earlier.RowLen()
	if err != nil {
		return fmt.Errorf("earlier frame: %w", err)
	}
	nextRows, err := next.RowLen()
	if err != nil {
		return fmt.Errorf("continuing frame: %w", err)
	}

	if len(continues.Fields) != len(next.Fields) {
		return fmt.Errorf("continues.fields has length %d, and the field count of the frame is %d", len(continues.Fields), len(next.Fields))
	}
	// mappedFrom is the field of next that each field of earlier maps to, or -1.
	mappedFrom := make([]int, len(earlier.Fields))
	for i := range mappedFrom {
		mappedFrom[i] = -1
	}
	for i, j := range continues.Fields {
		switch {
		case j == -1:
			continue
		case j < 0 || j >= len(earlier.Fields):
			return fmt.Errorf("field %d maps to index %d, and the field count of the earlier frame is %d", i, j, len(earlier.Fields))
		case mappedFrom[j] != -1:
			return fmt.Errorf("fields %d and %d map to the same field %d of the earlier frame", mappedFrom[j], i, j)
		case earlier.Fields[j].Type() != next.Fields[i].Type():
			return fmt.Errorf("field %d has type %s, and field %d of the earlier frame has type %s",
				i, next.Fields[i].Type(), j, earlier.Fields[j].Type())
		}
		mappedFrom[j] = i
	}
	for j, i := range mappedFrom {
		if i == -1 {
			return fmt.Errorf("field %d of the earlier frame has no field in the continuing frame", j)
		}
	}

	if cellLimit > 0 && (int64(earlierRows)+int64(nextRows))*int64(len(next.Fields)) > cellLimit {
		return ErrorWideFrameTooLarge
	}

	fields := make([]*Field, len(next.Fields))
	for i, j := range continues.Fields {
		if j == -1 {
			fields[i] = newContinuedField(next.Fields[i], earlierRows, continues.FillValue)
		} else {
			fields[i] = earlier.Fields[j]
		}
	}
	for i, nf := range next.Fields {
		fields[i].AppendAll(nf)
	}
	earlier.Fields = fields
	return nil
}

// newContinuedField returns a field with the schema of template and length rows of the fill value.
func newContinuedField(template *Field, length int, fillValue *float64) *Field {
	f := NewFieldFromFieldType(template.Type(), length)
	f.Name = template.Name
	f.Labels = template.Labels
	f.Config = template.Config
	if fillValue == nil || length == 0 {
		return f
	}
	// LongToWide also gives all earlier rows one value, and keeps the zero value when the fill value
	// does not convert to the type of the field.
	v, err := GetMissing(&FillMissing{Mode: FillModeValue, Value: *fillValue}, f, -1)
	if err != nil {
		return f
	}
	for i := range length {
		f.Set(i, v)
	}
	return f
}
