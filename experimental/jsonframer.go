package experimental

import (
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	jsoniter "github.com/json-iterator/go"
)

type doc struct {
	path []string
	iter *jsoniter.Iterator

	fields []*data.Field
}

func (d *doc) next() error {
	switch d.iter.WhatIsNext() {
	case jsoniter.StringValue:
		d.addString(d.iter.ReadString())
	case jsoniter.NumberValue:
		d.addNumber(d.iter.ReadFloat64())
	case jsoniter.BoolValue:
		d.addBool(d.iter.ReadBool())
	case jsoniter.NilValue:
		d.addNil()
	case jsoniter.ArrayValue:
		index := 0
		size := len(d.path)
		for d.iter.ReadArray() {
			d.path = append(d.path, fmt.Sprintf("[%d]", index))
			err := d.next()
			if err != nil {
				return err
			}
			d.path = d.path[:size]
			index++
		}
	case jsoniter.ObjectValue:
		size := len(d.path)
		for fname := d.iter.ReadObject(); fname != ""; fname = d.iter.ReadObject() {
			if size > 0 {
				d.path = append(d.path, ".")
			}
			d.path = append(d.path, fname)
			err := d.next()
			if err != nil {
				return err
			}
			d.path = d.path[:size]
		}
	case jsoniter.InvalidValue:
		return fmt.Errorf("invalid value")
	}
	return nil
}

func (d *doc) key() string {
	return strings.Join(d.path, "")
}

func (d *doc) addString(v string) {
	f := data.NewFieldFromFieldType(data.FieldTypeNullableString, 1)
	f.Name = d.key() // labels?
	f.SetConcrete(0, v)
	d.fields = append(d.fields, f)
}

func (d *doc) addNumber(v float64) {
	f := data.NewFieldFromFieldType(data.FieldTypeFloat64, 1)
	f.Name = d.key() // labels?
	f.SetConcrete(0, v)
	d.fields = append(d.fields, f)
}

func (d *doc) addBool(v bool) {
	f := data.NewFieldFromFieldType(data.FieldTypeNullableBool, 1)
	f.Name = d.key() // labels?
	f.SetConcrete(0, v)
	d.fields = append(d.fields, f)
}

func (d *doc) addNil() {
	fmt.Printf("??? nil: %v\n", d.key())
}

func JSONDocToFrame(body []byte) (*data.Frame, error) {
	d := doc{
		iter: jsoniter.ParseBytes(jsoniter.ConfigDefault, body),
		path: make([]string, 0),
	}
	err := d.next()

	if len(d.fields) < 1 {
		return nil, fmt.Errorf("no fields found")
	}

	f := data.NewFrame("", d.fields...)

	return f, err
}
