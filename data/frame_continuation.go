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
// response has the frame that one conversion of all rows gives.
type FrameContinuation struct {
	// FrameID is the frame ID of the earlier frame, in the same refId.
	FrameID string `json:"frameId"`

	// FillValue is the value of the new fields in the rows of the earlier frame. When it is nil, these
	// rows keep the zero value of the field type, which is null for a nullable field. LongToWide gives
	// the same values to the earlier rows of a series that starts late.
	FillValue *float64 `json:"fillValue,omitempty"`
}

// Experimental: JoinContinuation joins next into earlier, when next.Meta.Continues refers to earlier.
//
// Each field of earlier must have a field in next with the same name, labels and type. The other fields
// of next are new. In the rows of earlier, a new field gets next.Meta.Continues.FillValue, or the zero
// value of its type. After the join, earlier has the fields in the order of next, followed by the rows
// of next. earlier keeps its name and meta, and it keeps its rows without a copy. On an error, earlier
// does not change.
func JoinContinuation(earlier, next *Frame) error {
	if next.Meta == nil || next.Meta.Continues == nil {
		return errors.New("the frame does not continue an earlier frame")
	}
	rowLen, err := earlier.RowLen()
	if err != nil {
		return fmt.Errorf("earlier frame: %w", err)
	}
	if _, err := next.RowLen(); err != nil {
		return fmt.Errorf("continuing frame: %w", err)
	}

	// A frame can have more than one field with the same name and labels, so these fields match in order.
	unmatched := make(map[continuationKey][]*Field, len(earlier.Fields))
	for _, f := range earlier.Fields {
		key := newContinuationKey(f)
		unmatched[key] = append(unmatched[key], f)
	}

	fields := make([]*Field, len(next.Fields))
	matched := 0
	for i, nf := range next.Fields {
		key := newContinuationKey(nf)
		same := unmatched[key]
		if len(same) == 0 {
			fields[i] = newContinuedField(nf, rowLen, next.Meta.Continues.FillValue)
			continue
		}
		if same[0].Type() != nf.Type() {
			return fmt.Errorf("field %q has type %s in the earlier frame and %s in the continuing frame",
				nf.Name, same[0].Type(), nf.Type())
		}
		fields[i] = same[0]
		unmatched[key] = same[1:]
		matched++
	}
	if matched != len(earlier.Fields) {
		for _, f := range earlier.Fields {
			if len(unmatched[newContinuationKey(f)]) > 0 {
				return fmt.Errorf("field %q %s of the earlier frame is missing in the continuing frame", f.Name, f.Labels)
			}
		}
	}

	for i, nf := range next.Fields {
		fields[i].AppendAll(nf)
	}
	earlier.Fields = fields
	return nil
}

type continuationKey struct {
	name   string
	labels string
}

func newContinuationKey(f *Field) continuationKey {
	return continuationKey{name: f.Name, labels: f.Labels.String()}
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
