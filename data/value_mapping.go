package data

import (
	"encoding/json"
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

// MappingType see https://github.com/grafana/grafana/blob/main/packages/grafana-data/src/types/valueMapping.ts
type MappingType string

const (
	ValueToText  MappingType = "value"
	RangeToText  MappingType = "range"
	SpecialValue MappingType = "special"
)

type SpecialValueMatch string

const (
	SpecialValueTrue       SpecialValueMatch = "true"
	SpecialValueFalse      SpecialValueMatch = "false"
	SpecialValueNull       SpecialValueMatch = "null"
	SpecialValueNaN        SpecialValueMatch = "nan"
	SpecialValueNullAndNaN SpecialValueMatch = "null+nan"
	SpecialValueEmpty      SpecialValueMatch = "empty"
)

// ValueMappingResult is the results from mapping a value
type ValueMappingResult struct {
	Text  string `json:"text,omitempty"`
	Color string `json:"color,omitempty"`
	Index int    `json:"index,omitempty"` // just used ofr ui ordering
}

// ValueMapping allows mapping input values to text and color
type ValueMapping interface {
	GetType() MappingType
}

type ValueMappings []ValueMapping

// MarshalJSON writes the results as json
func (m ValueMappings) MarshalJSON() ([]byte, error) {
	cfg := jsoniter.ConfigCompatibleWithStandardLibrary
	stream := cfg.BorrowStream(nil)
	defer cfg.ReturnStream(stream)

	stream.WriteArrayStart()
	for idx, v := range m {
		if idx > 0 {
			stream.WriteMore()
		}

		stream.WriteObjectStart()
		stream.WriteObjectField("type")
		stream.WriteString(string(v.GetType()))

		stream.WriteMore()
		stream.WriteObjectField("options")
		stream.WriteVal(v)
		stream.WriteObjectEnd()
	}

	stream.WriteArrayEnd()
	return stream.Buffer(), stream.Error
}

// UnmarshalJSON will read JSON into the appropriate go types
func (m *ValueMappings) UnmarshalJSON(b []byte) error {
	iter := jsoniter.ParseBytes(jsoniter.ConfigDefault, b)
	var mappings ValueMappings

	for iter.ReadArray() {
		var objMap map[string]json.RawMessage
		iter.ReadVal(&objMap)
		mt := MappingType(strings.Trim(string(objMap["type"]), `"`))

		switch mt {
		case ValueToText:
			var mapper ValueMapper
			err := json.Unmarshal(objMap["options"], &mapper)
			if err != nil {
				return err
			}
			mappings = append(mappings, mapper)
		case RangeToText:
			var mapper RangeValueMapper
			err := json.Unmarshal(objMap["options"], &mapper)
			if err != nil {
				return err
			}
			mappings = append(mappings, mapper)
		case SpecialValue:
			var mapper SpecialValueMapper
			err := json.Unmarshal(objMap["options"], &mapper)
			if err != nil {
				return err
			}
			mappings = append(mappings, mapper)
		default:
			return fmt.Errorf("unknown mapping type: %s", mt)
		}
	}

	*m = mappings
	return iter.Error
}

// ValueMapper converts one set of strings to another
type ValueMapper map[string]ValueMappingResult

func (m ValueMapper) GetType() MappingType {
	return ValueToText
}

type SpecialValueMapper struct {
	Match  SpecialValueMatch  `json:"match"`
	Result ValueMappingResult `json:"result"`
}

func (m SpecialValueMapper) GetType() MappingType {
	return SpecialValue
}

type RangeValueMapper struct {
	From   ConfFloat64        `json:"from"`
	To     ConfFloat64        `json:"to"`
	Result ValueMappingResult `json:"result"`
}

func (m RangeValueMapper) GetType() MappingType {
	return RangeToText
}

// Make sure each type implements all required interfaces
var (
	_ ValueMapping = (*ValueMapper)(nil)
	_ ValueMapping = (*RangeValueMapper)(nil)
	_ ValueMapping = (*SpecialValueMapper)(nil)
)
