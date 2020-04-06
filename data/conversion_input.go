package data

import "fmt"


type FrameInputConverter struct {
	Frame           *Frame
	fieldConverters []FieldConverter
}

type Converter func(v interface{}) (interface{}, error)

var AsStringConverter Converter = func(v interface{}) (interface{}, error) {
	return fmt.Sprintf("%v", v), nil
}

func NewFrameInputConverter(fieldConvs []FieldConverter, rowLen int) (*FrameInputConverter, error) {
	fTypes := make([]FieldType, len(fieldConvs))
	for i, fc := range fieldConvs {
		fTypes[i] = fc.OutputFieldType
	}

	f := NewFrameOfFieldTypes("", rowLen, fTypes...)
	return &FrameInputConverter{
		Frame:           f,
		fieldConverters: fieldConvs,
	}, nil
}

func (fcb *FrameInputConverter) Set(fieldIdx, rowIdx int, val interface{}) error {
	convertedVal, err := fcb.fieldConverters[fieldIdx].Converter(val)
	if err != nil {
		return err
	}
	fcb.Frame.Set(fieldIdx, rowIdx, convertedVal)
	return nil
}

type FieldConverter struct {
	OutputFieldType FieldType
	Converter       Converter
}

var AsStringFieldConverter = FieldConverter{
	OutputFieldType: FieldTypeString,
	Converter:       AsStringConverter,
}
