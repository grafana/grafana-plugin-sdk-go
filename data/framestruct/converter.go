package framestruct

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

const frameTag = "frame"

type converter struct {
	fieldNames []string
	fields     map[string]*data.Field
	tags       []string
	anyMap     bool
	col0       string
}

// ToDataFrame flattens an arbitrary struct or slice of structs into a *data.Frame
func ToDataFrame(name string, toConvert interface{}) (*data.Frame, error) {
	cr := &converter{
		fields: make(map[string]*data.Field),
		tags:   make([]string, 3),
	}

	return cr.toDataframe(name, toConvert)
}

// ToDataFrames is a convenience wrapper around ToDataFrame. It will wrap the
// converted DataFrame in a data.Frames. Additionally, if the passed type
// satisfies the data.Framer interface, the function will delegate to that
// for the type conversion. If this function delegates to a data.Framer, it
// will use the data.Frame name defined by the type rather than passed to this
// function
func ToDataFrames(name string, toConvert interface{}) (data.Frames, error) {
	framer, ok := toConvert.(data.Framer)
	if ok {
		return framer.Frames()
	}

	frame, err := ToDataFrame(name, toConvert)
	if err != nil {
		return nil, err
	}

	return []*data.Frame{frame}, nil
}

func (c *converter) toDataframe(name string, toConvert interface{}) (*data.Frame, error) {
	v := c.ensureValue(reflect.ValueOf(toConvert))
	if !supportedToplevelType(v) {
		return nil, errors.New("unsupported type: can only convert structs, slices, and maps")
	}

	if err := c.handleValue(v, "", ""); err != nil {
		return nil, err
	}

	return c.createFrame(name), nil
}

func (c *converter) ensureValue(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

func (c *converter) handleValue(field reflect.Value, tags, fieldName string) error {
	switch field.Kind() {
	case reflect.Slice:
		return c.convertSlice(field, fieldName)
	case reflect.Struct:
		return c.convertStruct(field, fieldName)
	case reflect.Map:
		return c.convertMap(field.Interface(), tags, fieldName)
	default:
		return c.upsertField(field, fieldName)
	}
}

func (c *converter) convertStruct(field reflect.Value, fieldName string) error {
	_, ok := field.Interface().(time.Time)
	if ok {
		return c.upsertField(field, fieldName)
	}

	return c.convertStructFields(field, fieldName)
}

func (c *converter) convertSlice(s reflect.Value, prefix string) error {
	for i := 0; i < s.Len(); i++ {
		v := s.Index(i)
		switch v.Kind() {
		case reflect.Map:
			if err := c.convertMap(v.Interface(), "", prefix); err != nil {
				return err
			}
		default:
			if err := c.convertStruct(v, prefix); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *converter) convertStructFields(v reflect.Value, prefix string) error {
	if v.Kind() != reflect.Struct {
		return errors.New("unsupported type: converted types may not contain slices")
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !exported(field) {
			continue
		}

		structField := v.Type().Field(i)
		tags := structField.Tag.Get(frameTag)

		if tags == "-" {
			continue
		}

		fieldName := c.fieldName(structField.Name, tags, prefix)
		if err := c.handleValue(field, tags, fieldName); err != nil {
			return err
		}

		c.parseTags(tags)
		if c.tags[2] != "" {
			c.col0 = fieldName
		}
	}
	return nil
}

func exported(v reflect.Value) bool {
	return v.CanInterface()
}

func (c *converter) convertMap(toConvert interface{}, tags, prefix string) error {
	c.anyMap = true
	m, ok := toConvert.(map[string]interface{})
	if !ok {
		return errors.New("map must be map[string]interface{}")
	}

	for name, value := range m {
		fieldName := c.fieldName(name, tags, prefix)
		v := c.ensureValue(reflect.ValueOf(value))
		if err := c.handleValue(v, "", fieldName); err != nil {
			return err
		}
	}

	return nil
}

func (c *converter) upsertField(v reflect.Value, fieldName string) error {
	if _, exists := c.fields[fieldName]; !exists {
		// keep track of unique fields in the order they appear
		c.fieldNames = append(c.fieldNames, fieldName)
		v, err := sliceFor(v.Interface())
		if err != nil {
			return err
		}

		c.fields[fieldName] = data.NewField(fieldName, nil, v)
	}
	c.fields[fieldName].Append(v.Interface())
	return nil
}

func (c *converter) createFrame(name string) *data.Frame {
	frame := data.NewFrame(name)
	for _, f := range c.getFieldnames() {
		frame.Fields = append(frame.Fields, c.fields[f])
	}
	return frame
}

func (c *converter) getFieldnames() []string {
	if c.anyMap {
		// Ensure stable order of fields across
		// runs, because maps
		sort.Strings(c.fieldNames)
	}

	fieldnames := []string{}
	if c.col0 != "" {
		fieldnames = append(fieldnames, c.col0)
	}
	for _, f := range c.fieldNames {
		if f != c.col0 {
			fieldnames = append(fieldnames, f)
		}
	}

	return fieldnames
}

func (c *converter) fieldName(fieldName, tags, prefix string) string {
	c.parseTags(tags)
	if c.tags[1] == "omitparent" {
		prefix = ""
	}

	if c.tags[0] != "" {
		fieldName = c.tags[0]
	}

	if prefix == "" {
		return fieldName
	}

	return prefix + "." + fieldName
}

func (c *converter) parseTags(s string) {
	// if we do it this way, we avoid all the allocs
	// of strings.Split
	c.tags[0] = ""
	c.tags[1] = ""
	c.tags[2] = ""

	sep := ","

	i := 0
	for i < 2 {
		m := strings.Index(s, sep)
		if m < 0 {
			break
		}
		c.tags[i] = strings.TrimSpace(s[:m])
		s = s[m+len(sep):]
		i++
	}

	if i < len(c.tags) {
		c.tags[i] = s
	}
}
