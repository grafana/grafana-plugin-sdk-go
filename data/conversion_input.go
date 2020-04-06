package data

import "fmt"

type FrameConvertBuilder struct {
	*Frame
	converters []Converter
}

type Converter func(v interface{}) (interface{}, error)

func NewFrameConvertBuilder(f *Frame, converters []Converter) (*FrameConvertBuilder, error) {
	if len(f.Fields) != len(converters) {
		return nil, fmt.Errorf("converters length must match frame Field Length")
	}
	return &FrameConvertBuilder{
		Frame:      f,
		converters: converters,
	}, nil
}

func (fcb *FrameConvertBuilder) Set(fieldIdx, rowIdx int, val interface{}) error {
	convertedVal, err := fcb.converters[fieldIdx](val)
	if err != nil {
		return err
	}
	fcb.Set(fieldIdx, rowIdx, convertedVal)
	return nil
}
