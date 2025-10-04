package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/arrio"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/mattetti/filebuffer"
)

// keys added to arrow field metadata
const metadataKeyName = "name"     // standard property
const metadataKeyConfig = "config" // FieldConfig serialized as JSON
const metadataKeyLabels = "labels" // labels serialized as JSON
const metadataKeyTSType = "tstype" // typescript type
const metadataKeyRefID = "refId"   // added to the table metadata

// MarshalArrow converts the Frame to an arrow table and returns a byte
// representation of that table.
// All fields of a Frame must be of the same length or an error is returned.
func (f *Frame) MarshalArrow() ([]byte, error) {
	table, err := FrameToArrowTable(f)
	if err != nil {
		return nil, err
	}
	defer table.Release()

	tableReader := array.NewTableReader(table, -1)
	defer tableReader.Release()

	// Arrow tables with the Go API are written to files, so we create a fake
	// file buffer that the FileWriter can write to. In the future, and with
	// streaming, I think will likely be using the Arrow message type some how.
	fb := filebuffer.New(nil)

	fw, err := ipc.NewFileWriter(fb, ipc.WithSchema(tableReader.Schema()))
	if err != nil {
		return nil, err
	}

	for tableReader.Next() {
		rec := tableReader.Record()

		if err := fw.Write(rec); err != nil {
			rec.Release()
			return nil, err
		}
		rec.Release()
	}

	if err := fw.Close(); err != nil {
		return nil, err
	}

	return fb.Buff.Bytes(), nil
}

// FrameToArrowTable creates a new arrow.Table from a data frame
// To release the allocated memory be sure to call:
//
//	defer table.Release()
func FrameToArrowTable(f *Frame) (arrow.Table, error) {
	if _, err := f.RowLen(); err != nil {
		return nil, err
	}

	arrowFields, err := buildArrowFields(f)
	if err != nil {
		return nil, err
	}

	schema, err := buildArrowSchema(f, arrowFields)
	if err != nil {
		return nil, err
	}

	columns, err := buildArrowColumns(f, arrowFields)
	if err != nil {
		for _, col := range columns {
			col.Release()
		}
		return nil, err
	}

	// Create a table from the schema and columns.
	return array.NewTable(schema, columns, -1), nil
}

// buildArrowFields builds Arrow field definitions from a Frame.
func buildArrowFields(f *Frame) ([]arrow.Field, error) {
	arrowFields := make([]arrow.Field, len(f.Fields))

	for i, field := range f.Fields {
		t, nullable, err := fieldToArrow(field)
		if err != nil {
			return nil, err
		}
		tstype, _ := getTypeScriptTypeString(field.Type())
		fieldMeta := map[string]string{
			metadataKeyTSType: tstype,
		}

		if field.Labels != nil {
			if fieldMeta[metadataKeyLabels], err = toJSONString(field.Labels); err != nil {
				return nil, err
			}
		}

		if field.Config != nil {
			str, err := toJSONString(field.Config)
			if err != nil {
				return nil, err
			}
			fieldMeta[metadataKeyConfig] = str
		}

		arrowFields[i] = arrow.Field{
			Name:     field.Name,
			Type:     t,
			Metadata: arrow.MetadataFrom(fieldMeta),
			Nullable: nullable,
		}
	}

	return arrowFields, nil
}

// buildArrowColumns builds Arrow columns from a Frame.
// nolint:gocyclo
func buildArrowColumns(f *Frame, arrowFields []arrow.Field) ([]arrow.Column, error) {
	pool := memory.NewGoAllocator()
	columns := make([]arrow.Column, len(f.Fields))

	for fieldIdx, field := range f.Fields {
		switch v := field.vector.(type) {
		// Time, JSON, and Enum types
		case *genericVector[time.Time]:
			columns[fieldIdx] = *buildTimeColumnGeneric(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[time.Time]:
			columns[fieldIdx] = *buildNullableTimeColumnGeneric(pool, arrowFields[fieldIdx], v)

		case *genericVector[json.RawMessage]:
			columns[fieldIdx] = *buildJSONColumnGeneric(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[json.RawMessage]:
			columns[fieldIdx] = *buildNullableJSONColumnGeneric(pool, arrowFields[fieldIdx], v)

		case *genericVector[EnumItemIndex]:
			columns[fieldIdx] = *buildEnumColumnGeneric(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[EnumItemIndex]:
			columns[fieldIdx] = *buildNullableEnumColumnGeneric(pool, arrowFields[fieldIdx], v)

		// Generic vectors - use directly without conversion
		case *genericVector[int8]:
			columns[fieldIdx] = *buildInt8Column(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[int8]:
			columns[fieldIdx] = *buildNullableInt8Column(pool, arrowFields[fieldIdx], v)
		case *genericVector[int16]:
			columns[fieldIdx] = *buildInt16Column(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[int16]:
			columns[fieldIdx] = *buildNullableInt16Column(pool, arrowFields[fieldIdx], v)
		case *genericVector[int32]:
			columns[fieldIdx] = *buildInt32Column(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[int32]:
			columns[fieldIdx] = *buildNullableInt32Column(pool, arrowFields[fieldIdx], v)
		case *genericVector[int64]:
			columns[fieldIdx] = *buildInt64Column(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[int64]:
			columns[fieldIdx] = *buildNullableInt64Column(pool, arrowFields[fieldIdx], v)
		case *genericVector[uint8]:
			columns[fieldIdx] = *buildUInt8Column(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[uint8]:
			columns[fieldIdx] = *buildNullableUInt8Column(pool, arrowFields[fieldIdx], v)
		case *genericVector[uint16]:
			columns[fieldIdx] = *buildUInt16Column(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[uint16]:
			columns[fieldIdx] = *buildNullableUInt16Column(pool, arrowFields[fieldIdx], v)
		case *genericVector[uint32]:
			columns[fieldIdx] = *buildUInt32Column(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[uint32]:
			columns[fieldIdx] = *buildNullableUInt32Column(pool, arrowFields[fieldIdx], v)
		case *genericVector[uint64]:
			columns[fieldIdx] = *buildUInt64Column(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[uint64]:
			columns[fieldIdx] = *buildNullableUInt64Column(pool, arrowFields[fieldIdx], v)
		case *genericVector[float32]:
			columns[fieldIdx] = *buildFloat32Column(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[float32]:
			columns[fieldIdx] = *buildNullableFloat32Column(pool, arrowFields[fieldIdx], v)
		case *genericVector[float64]:
			columns[fieldIdx] = *buildFloat64Column(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[float64]:
			columns[fieldIdx] = *buildNullableFloat64Column(pool, arrowFields[fieldIdx], v)
		case *genericVector[string]:
			columns[fieldIdx] = *buildStringColumn(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[string]:
			columns[fieldIdx] = *buildNullableStringColumn(pool, arrowFields[fieldIdx], v)
		case *genericVector[bool]:
			columns[fieldIdx] = *buildBoolColumn(pool, arrowFields[fieldIdx], v)
		case *nullableGenericVector[bool]:
			columns[fieldIdx] = *buildNullableBoolColumn(pool, arrowFields[fieldIdx], v)

		default:
			return nil, fmt.Errorf("unsupported field vector type for conversion to arrow: %T", v)
		}
	}
	return columns, nil
}

// buildArrowSchema builds an Arrow schema for a Frame.
func buildArrowSchema(f *Frame, fs []arrow.Field) (*arrow.Schema, error) {
	tableMetaMap := map[string]string{
		metadataKeyName:  f.Name,
		metadataKeyRefID: f.RefID,
	}
	if f.Meta != nil {
		str, err := toJSONString(f.Meta)
		if err != nil {
			return nil, err
		}
		tableMetaMap["meta"] = str
	}
	tableMeta := arrow.MetadataFrom(tableMetaMap)

	return arrow.NewSchema(fs, &tableMeta), nil
}

// fieldToArrow returns the corresponding Arrow primitive type and nullable property to the fields'
// Vector primitives.
// nolint:gocyclo
func fieldToArrow(f *Field) (arrow.DataType, bool, error) {
	switch f.vector.(type) {
	// Time, JSON, and Enum types
	case *genericVector[time.Time]:
		return &arrow.TimestampType{Unit: arrow.Nanosecond}, false, nil
	case *nullableGenericVector[time.Time]:
		return &arrow.TimestampType{Unit: arrow.Nanosecond}, true, nil

	case *genericVector[json.RawMessage]:
		return &arrow.BinaryType{}, false, nil
	case *nullableGenericVector[json.RawMessage]:
		return &arrow.BinaryType{}, true, nil

	case *genericVector[EnumItemIndex]:
		return &arrow.Uint16Type{}, false, nil
	case *nullableGenericVector[EnumItemIndex]:
		return &arrow.Uint16Type{}, true, nil

	// Generic vectors
	case *genericVector[string]:
		return &arrow.StringType{}, false, nil
	case *nullableGenericVector[string]:
		return &arrow.StringType{}, true, nil
	case *genericVector[int8]:
		return &arrow.Int8Type{}, false, nil
	case *nullableGenericVector[int8]:
		return &arrow.Int8Type{}, true, nil
	case *genericVector[int16]:
		return &arrow.Int16Type{}, false, nil
	case *nullableGenericVector[int16]:
		return &arrow.Int16Type{}, true, nil
	case *genericVector[int32]:
		return &arrow.Int32Type{}, false, nil
	case *nullableGenericVector[int32]:
		return &arrow.Int32Type{}, true, nil
	case *genericVector[int64]:
		return &arrow.Int64Type{}, false, nil
	case *nullableGenericVector[int64]:
		return &arrow.Int64Type{}, true, nil
	case *genericVector[uint8]:
		return &arrow.Uint8Type{}, false, nil
	case *nullableGenericVector[uint8]:
		return &arrow.Uint8Type{}, true, nil
	case *genericVector[uint16]:
		return &arrow.Uint16Type{}, false, nil
	case *nullableGenericVector[uint16]:
		return &arrow.Uint16Type{}, true, nil
	case *genericVector[uint32]:
		return &arrow.Uint32Type{}, false, nil
	case *nullableGenericVector[uint32]:
		return &arrow.Uint32Type{}, true, nil
	case *genericVector[uint64]:
		return &arrow.Uint64Type{}, false, nil
	case *nullableGenericVector[uint64]:
		return &arrow.Uint64Type{}, true, nil
	case *genericVector[float32]:
		return &arrow.Float32Type{}, false, nil
	case *nullableGenericVector[float32]:
		return &arrow.Float32Type{}, true, nil
	case *genericVector[float64]:
		return &arrow.Float64Type{}, false, nil
	case *nullableGenericVector[float64]:
		return &arrow.Float64Type{}, true, nil
	case *genericVector[bool]:
		return &arrow.BooleanType{}, false, nil
	case *nullableGenericVector[bool]:
		return &arrow.BooleanType{}, true, nil

	default:
		return nil, false, fmt.Errorf("unsupported type for conversion to arrow: %T", f.vector)
	}
}

func getMDKey(key string, metaData arrow.Metadata) (string, bool) {
	idx := metaData.FindKey(key)
	if idx < 0 {
		return "", false
	}
	return metaData.Values()[idx], true
}

func initializeFrameFields(schema *arrow.Schema, frame *Frame) ([]bool, error) {
	nullable := make([]bool, len(schema.Fields()))
	for idx, field := range schema.Fields() {
		sdkField := Field{
			Name: field.Name,
		}
		if labelsAsString, ok := getMDKey(metadataKeyLabels, field.Metadata); ok {
			if err := json.Unmarshal([]byte(labelsAsString), &sdkField.Labels); err != nil {
				return nil, err
			}
		}
		if configAsString, ok := getMDKey(metadataKeyConfig, field.Metadata); ok {
			// make sure that Config is not nil, otherwise create a new one
			if sdkField.Config == nil {
				sdkField.Config = &FieldConfig{}
			}
			if err := json.Unmarshal([]byte(configAsString), sdkField.Config); err != nil {
				return nil, err
			}
		}
		nullable[idx] = field.Nullable
		if err := initializeFrameField(field, idx, nullable, &sdkField); err != nil {
			return nil, err
		}

		frame.Fields = append(frame.Fields, &sdkField)
	}
	return nullable, nil
}

// nolint:gocyclo
func initializeFrameField(field arrow.Field, idx int, nullable []bool, sdkField *Field) error {
	switch field.Type.ID() {
	case arrow.STRING:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[string](0)
			break
		}
		sdkField.vector = newGenericVector[string](0)
	case arrow.STRING_VIEW:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[string](0)
			break
		}
		sdkField.vector = newGenericVector[string](0)
	case arrow.INT8:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[int8](0)
			break
		}
		sdkField.vector = newGenericVector[int8](0)
	case arrow.INT16:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[int16](0)
			break
		}
		sdkField.vector = newGenericVector[int16](0)
	case arrow.INT32:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[int32](0)
			break
		}
		sdkField.vector = newGenericVector[int32](0)
	case arrow.INT64:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[int64](0)
			break
		}
		sdkField.vector = newGenericVector[int64](0)
	case arrow.UINT8:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[uint8](0)
			break
		}
		sdkField.vector = newGenericVector[uint8](0)
	case arrow.UINT16:
		tstype, ok := getMDKey(metadataKeyTSType, field.Metadata)
		if ok && tstype == simpleTypeEnum {
			if nullable[idx] {
				sdkField.vector = newNullableGenericVector[EnumItemIndex](0)
			} else {
				sdkField.vector = newGenericVector[EnumItemIndex](0)
			}
			break
		}
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[uint16](0)
			break
		}
		sdkField.vector = newGenericVector[uint16](0)
	case arrow.UINT32:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[uint32](0)
			break
		}
		sdkField.vector = newGenericVector[uint32](0)
	case arrow.UINT64:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[uint64](0)
			break
		}
		sdkField.vector = newGenericVector[uint64](0)
	case arrow.FLOAT32:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[float32](0)
			break
		}
		sdkField.vector = newGenericVector[float32](0)
	case arrow.FLOAT64:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[float64](0)
			break
		}
		sdkField.vector = newGenericVector[float64](0)
	case arrow.BOOL:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[bool](0)
			break
		}
		sdkField.vector = newGenericVector[bool](0)
	case arrow.TIMESTAMP:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[time.Time](0)
			break
		}
		sdkField.vector = newGenericVector[time.Time](0)
	case arrow.BINARY:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVector[json.RawMessage](0)
			break
		}
		sdkField.vector = newGenericVector[json.RawMessage](0)
	default:
		return fmt.Errorf("unsupported conversion from arrow to sdk type for arrow type %v", field.Type.ID().String())
	}

	return nil
}

func populateFrameFieldsFromRecord(record arrow.Record, nullable []bool, frame *Frame) error {
	for i := 0; i < len(frame.Fields); i++ {
		col := record.Column(i)
		if err := parseColumn(col, i, nullable, frame); err != nil {
			return err
		}
	}
	return nil
}

func populateFrameFields(fR arrio.Reader, nullable []bool, frame *Frame) error {
	for {
		record, err := fR.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		if err = populateFrameFieldsFromRecord(record, nullable, frame); err != nil {
			return err
		}
	}
	return nil
}

// nolint:gocyclo
func parseColumn(col arrow.Array, i int, nullable []bool, frame *Frame) error {
	switch col.DataType().ID() {
	case arrow.STRING:
		v := array.NewStringData(col.Data())
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *string
					appendTypedToVector(frame.Fields[i].vector, ns)
					continue
				}
				rv := v.Value(rIdx)
				appendTypedToVector(frame.Fields[i].vector, &rv)
				continue
			}
			appendTypedToVector(frame.Fields[i].vector, v.Value(rIdx))
		}
	case arrow.STRING_VIEW:
		v := array.NewStringViewData(col.Data())
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *string
					appendTypedToVector(frame.Fields[i].vector, ns)
					continue
				}
				rv := v.Value(rIdx)
				appendTypedToVector(frame.Fields[i].vector, &rv)
				continue
			}
			appendTypedToVector(frame.Fields[i].vector, v.Value(rIdx))
		}
	case arrow.INT8:
		v := array.NewInt8Data(col.Data())
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *int8
					appendTypedToVector(frame.Fields[i].vector, ns)
					continue
				}
				rv := v.Value(rIdx)
				appendTypedToVector(frame.Fields[i].vector, &rv)
				continue
			}
			appendTypedToVector(frame.Fields[i].vector, v.Value(rIdx))
		}
	case arrow.INT16:
		v := array.NewInt16Data(col.Data())
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *int16
					appendTypedToVector(frame.Fields[i].vector, ns)
					continue
				}
				rv := v.Value(rIdx)
				appendTypedToVector(frame.Fields[i].vector, &rv)
				continue
			}
			appendTypedToVector(frame.Fields[i].vector, v.Value(rIdx))
		}
	case arrow.INT32:
		v := array.NewInt32Data(col.Data())
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *int32
					appendTypedToVector(frame.Fields[i].vector, ns)
					continue
				}
				rv := v.Value(rIdx)
				appendTypedToVector(frame.Fields[i].vector, &rv)
				continue
			}
			appendTypedToVector(frame.Fields[i].vector, v.Value(rIdx))
		}
	case arrow.INT64:
		v := array.NewInt64Data(col.Data())
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *int64
					appendTypedToVector(frame.Fields[i].vector, ns)
					continue
				}
				rv := v.Value(rIdx)
				appendTypedToVector(frame.Fields[i].vector, &rv)
				continue
			}
			appendTypedToVector(frame.Fields[i].vector, v.Value(rIdx))
		}
	case arrow.UINT8:
		v := array.NewUint8Data(col.Data())
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *uint8
					appendTypedToVector(frame.Fields[i].vector, ns)
					continue
				}
				rv := v.Value(rIdx)
				appendTypedToVector(frame.Fields[i].vector, &rv)
				continue
			}
			appendTypedToVector(frame.Fields[i].vector, v.Value(rIdx))
		}
	case arrow.UINT32:
		v := array.NewUint32Data(col.Data())
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *uint32
					appendTypedToVector(frame.Fields[i].vector, ns)
					continue
				}
				rv := v.Value(rIdx)
				appendTypedToVector(frame.Fields[i].vector, &rv)
				continue
			}
			appendTypedToVector(frame.Fields[i].vector, v.Value(rIdx))
		}
	case arrow.UINT64:
		v := array.NewUint64Data(col.Data())
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *uint64
					appendTypedToVector(frame.Fields[i].vector, ns)
					continue
				}
				rv := v.Value(rIdx)
				appendTypedToVector(frame.Fields[i].vector, &rv)
				continue
			}
			appendTypedToVector(frame.Fields[i].vector, v.Value(rIdx))
		}
	case arrow.UINT16:
		v := array.NewUint16Data(col.Data())
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if frame.Fields[i].Type().NullableType() == FieldTypeNullableEnum {
				if nullable[i] {
					if v.IsNull(rIdx) {
						var ns *EnumItemIndex
						appendTypedToVector(frame.Fields[i].vector, ns)
						continue
					}
					rv := EnumItemIndex(v.Value(rIdx))
					appendTypedToVector(frame.Fields[i].vector, &rv)
					continue
				}
				appendTypedToVector(frame.Fields[i].vector, EnumItemIndex(v.Value(rIdx)))
			} else {
				if nullable[i] {
					if v.IsNull(rIdx) {
						var ns *uint16
						appendTypedToVector(frame.Fields[i].vector, ns)
						continue
					}
					rv := v.Value(rIdx)
					appendTypedToVector(frame.Fields[i].vector, &rv)
					continue
				}
				appendTypedToVector(frame.Fields[i].vector, v.Value(rIdx))
			}
		}
	case arrow.FLOAT32:
		v := array.NewFloat32Data(col.Data())
		for vIdx, f := range v.Float32Values() {
			if nullable[i] {
				if v.IsNull(vIdx) {
					var nf *float32
					appendTypedToVector(frame.Fields[i].vector, nf)
					continue
				}
				vF := f
				appendTypedToVector(frame.Fields[i].vector, &vF)
				continue
			}
			appendTypedToVector(frame.Fields[i].vector, f)
		}
	case arrow.FLOAT64:
		v := array.NewFloat64Data(col.Data())
		for vIdx, f := range v.Float64Values() {
			if nullable[i] {
				if v.IsNull(vIdx) {
					var nf *float64
					appendTypedToVector(frame.Fields[i].vector, nf)
					continue
				}
				vF := f
				appendTypedToVector(frame.Fields[i].vector, &vF)
				continue
			}
			appendTypedToVector(frame.Fields[i].vector, f)
		}
	case arrow.BOOL:
		v := array.NewBooleanData(col.Data())
		for sIdx := 0; sIdx < col.Len(); sIdx++ {
			if nullable[i] {
				if v.IsNull(sIdx) {
					var ns *bool
					appendTypedToVector(frame.Fields[i].vector, ns)
					continue
				}
				vB := v.Value(sIdx)
				appendTypedToVector(frame.Fields[i].vector, &vB)
				continue
			}
			appendTypedToVector(frame.Fields[i].vector, v.Value(sIdx))
		}
	case arrow.TIMESTAMP:
		v := array.NewTimestampData(col.Data())
		for vIdx, ts := range v.TimestampValues() {
			t := time.Unix(0, int64(ts)) // nanosecond assumption
			if nullable[i] {
				if v.IsNull(vIdx) {
					var nt *time.Time
					appendTypedToVector(frame.Fields[i].vector, nt)
					continue
				}
				appendTypedToVector(frame.Fields[i].vector, &t)
				continue
			}
			appendTypedToVector(frame.Fields[i].vector, t)
		}
	case arrow.BINARY:
		v := array.NewBinaryData(col.Data())
		for sIdx := 0; sIdx < v.Len(); sIdx++ {
			if nullable[i] {
				if v.IsNull(sIdx) {
					var nb *json.RawMessage
					appendTypedToVector(frame.Fields[i].vector, nb)
					continue
				}
				r := json.RawMessage(v.Value(sIdx))
				appendTypedToVector(frame.Fields[i].vector, &r)
				continue
			}
			r := json.RawMessage(v.Value(sIdx))
			appendTypedToVector(frame.Fields[i].vector, r)
		}
	default:
		return fmt.Errorf("unsupported arrow type %s for conversion", col.DataType().ID())
	}

	return nil
}

func populateFrameFromSchema(schema *arrow.Schema, frame *Frame) error {
	metaData := schema.Metadata()
	frame.Name, _ = getMDKey(metadataKeyName, metaData) // No need to check ok, zero value ("") is returned
	frame.RefID, _ = getMDKey(metadataKeyRefID, metaData)

	var err error
	if metaAsString, ok := getMDKey("meta", metaData); ok {
		frame.Meta, err = FrameMetaFromJSON(metaAsString)
	}

	return err
}

// FromArrowRecord converts a an Arrow record batch into a Frame.
func FromArrowRecord(record arrow.Record) (*Frame, error) {
	schema := record.Schema()
	frame := &Frame{}
	if err := populateFrameFromSchema(schema, frame); err != nil {
		return nil, err
	}

	nullable, err := initializeFrameFields(schema, frame)
	if err != nil {
		return nil, err
	}

	if err = populateFrameFieldsFromRecord(record, nullable, frame); err != nil {
		return nil, err
	}
	return frame, nil
}

// UnmarshalArrowFrame converts a byte representation of an arrow table to a Frame.
func UnmarshalArrowFrame(b []byte) (*Frame, error) {
	fB := filebuffer.New(b)
	fR, err := ipc.NewFileReader(fB)
	if err != nil {
		return nil, err
	}
	defer fR.Close()

	schema := fR.Schema()
	frame := &Frame{}
	if err := populateFrameFromSchema(schema, frame); err != nil {
		return nil, err
	}

	nullable, err := initializeFrameFields(schema, frame)
	if err != nil {
		return nil, err
	}

	if err = populateFrameFields(fR, nullable, frame); err != nil {
		return nil, err
	}

	return frame, nil
}

// ToJSONString calls json.Marshal on val and returns it as a string. An
// error is returned if json.Marshal errors.
func toJSONString(val interface{}) (string, error) {
	b, err := json.Marshal(val)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// UnmarshalArrowFrames decodes a slice of Arrow encoded frames to Frames ([]*Frame) by calling
// the UnmarshalArrow function on each encoded frame.
// If an error occurs Frames will be nil.
// See Frames.UnMarshalArrow() for the inverse operation.
func UnmarshalArrowFrames(bFrames [][]byte) (Frames, error) {
	frames := make(Frames, len(bFrames))
	var err error
	for i, encodedFrame := range bFrames {
		frames[i], err = UnmarshalArrowFrame(encodedFrame)
		if err != nil {
			return nil, err
		}
	}
	return frames, nil
}

// MarshalArrow encodes Frames into a slice of []byte using *Frame's MarshalArrow method on each Frame.
// If an error occurs [][]byte will be nil.
// See UnmarshalArrowFrames for the inverse operation.
func (frames Frames) MarshalArrow() ([][]byte, error) {
	bs := make([][]byte, len(frames))
	var err error
	for i, frame := range frames {
		if frame == nil {
			return nil, errors.New("frame can not be nil")
		}
		bs[i], err = frame.MarshalArrow()
		if err != nil {
			return nil, err
		}
	}
	return bs, nil
}
