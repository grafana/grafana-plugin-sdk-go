package data

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/arrio"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// keys added to arrow field metadata
const metadataKeyName = "name"     // standard property
const metadataKeyConfig = "config" // FieldConfig serialized as JSON
const metadataKeyLabels = "labels" // labels serialized as JSON
const metadataKeyTSType = "tstype" // typescript type
const metadataKeyRefID = "refId"   // added to the table metadata

// MarshalArrow encodes the Frame as an Arrow IPC file with one record batch.
// The returned slice has no spare capacity.
// All fields of a Frame must be of the same length or an error is returned.
func (f *Frame) MarshalArrow() ([]byte, error) {
	var buf bytes.Buffer
	if err := f.marshalArrow(&buf); err != nil {
		return nil, err
	}
	return exactSizeCopy(buf.Bytes()), nil
}

// marshalArrow writes the Frame to w as an Arrow IPC file with one record batch.
func (f *Frame) marshalArrow(w io.Writer) error {
	rec, err := frameToArrowRecord(f, memory.DefaultAllocator)
	if err != nil {
		return err
	}
	defer rec.Release()

	fw, err := ipc.NewFileWriter(w, ipc.WithSchema(rec.Schema()))
	if err != nil {
		return err
	}
	// A frame with no rows is encoded as schema and footer only, without a record batch.
	if rec.NumRows() > 0 {
		if err := fw.Write(rec); err != nil {
			return err
		}
	}
	return fw.Close()
}

func frameToArrowRecord(f *Frame, mem memory.Allocator) (arrow.RecordBatch, error) {
	rows, err := f.RowLen()
	if err != nil {
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

	arrays, err := buildArrowArrays(f, mem)
	if err != nil {
		return nil, err
	}
	rec := array.NewRecordBatch(schema, arrays, int64(rows))
	for _, arr := range arrays {
		arr.Release()
	}
	return rec, nil
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

	arrays, err := buildArrowArrays(f, memory.DefaultAllocator)
	if err != nil {
		return nil, err
	}
	columns := make([]arrow.Column, len(arrays))
	for i, arr := range arrays {
		columns[i] = arrow.NewColumnFromArr(arrowFields[i], arr)
		arr.Release()
	}

	// Create a table from the schema and columns.
	table := array.NewTable(schema, columns, -1)
	for i := range columns {
		columns[i].Release()
	}
	return table, nil
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

// buildArrowArrays builds one Arrow array per field. The caller releases the returned arrays.
// nolint:gocyclo
func buildArrowArrays(f *Frame, pool memory.Allocator) ([]arrow.Array, error) {
	arrays := make([]arrow.Array, len(f.Fields))

	for fieldIdx, field := range f.Fields {
		switch v := field.vector.(type) {
		case *int8Vector:
			arrays[fieldIdx] = buildInt8Array(pool, v)
		case *nullableInt8Vector:
			arrays[fieldIdx] = buildNullableInt8Array(pool, v)

		case *int16Vector:
			arrays[fieldIdx] = buildInt16Array(pool, v)
		case *nullableInt16Vector:
			arrays[fieldIdx] = buildNullableInt16Array(pool, v)

		case *int32Vector:
			arrays[fieldIdx] = buildInt32Array(pool, v)
		case *nullableInt32Vector:
			arrays[fieldIdx] = buildNullableInt32Array(pool, v)

		case *int64Vector:
			arrays[fieldIdx] = buildInt64Array(pool, v)
		case *nullableInt64Vector:
			arrays[fieldIdx] = buildNullableInt64Array(pool, v)

		case *uint8Vector:
			arrays[fieldIdx] = buildUInt8Array(pool, v)
		case *nullableUint8Vector:
			arrays[fieldIdx] = buildNullableUInt8Array(pool, v)

		case *uint16Vector:
			arrays[fieldIdx] = buildUInt16Array(pool, v)
		case *nullableUint16Vector:
			arrays[fieldIdx] = buildNullableUInt16Array(pool, v)

		case *uint32Vector:
			arrays[fieldIdx] = buildUInt32Array(pool, v)
		case *nullableUint32Vector:
			arrays[fieldIdx] = buildNullableUInt32Array(pool, v)

		case *uint64Vector:
			arrays[fieldIdx] = buildUInt64Array(pool, v)
		case *nullableUint64Vector:
			arrays[fieldIdx] = buildNullableUInt64Array(pool, v)

		case *stringVector:
			arrays[fieldIdx] = buildStringArray(pool, v)
		case *nullableStringVector:
			arrays[fieldIdx] = buildNullableStringArray(pool, v)

		case *float32Vector:
			arrays[fieldIdx] = buildFloat32Array(pool, v)
		case *nullableFloat32Vector:
			arrays[fieldIdx] = buildNullableFloat32Array(pool, v)

		case *float64Vector:
			arrays[fieldIdx] = buildFloat64Array(pool, v)
		case *nullableFloat64Vector:
			arrays[fieldIdx] = buildNullableFloat64Array(pool, v)

		case *boolVector:
			arrays[fieldIdx] = buildBoolArray(pool, v)
		case *nullableBoolVector:
			arrays[fieldIdx] = buildNullableBoolArray(pool, v)

		case *timeTimeVector:
			arrays[fieldIdx] = buildTimeArray(pool, v)
		case *nullableTimeTimeVector:
			arrays[fieldIdx] = buildNullableTimeArray(pool, v)

		case *jsonRawMessageVector:
			arrays[fieldIdx] = buildJSONArray(pool, v)
		case *nullableJsonRawMessageVector:
			arrays[fieldIdx] = buildNullableJSONArray(pool, v)

		case *enumVector:
			arrays[fieldIdx] = buildEnumArray(pool, v)
		case *nullableEnumVector:
			arrays[fieldIdx] = buildNullableEnumArray(pool, v)

		default:
			for _, arr := range arrays[:fieldIdx] {
				arr.Release()
			}
			return nil, fmt.Errorf("unsupported field vector type for conversion to arrow: %T", v)
		}
	}
	return arrays, nil
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
	case *stringVector:
		return &arrow.StringType{}, false, nil
	case *nullableStringVector:
		return &arrow.StringType{}, true, nil

	// Ints
	case *int8Vector:
		return &arrow.Int8Type{}, false, nil
	case *nullableInt8Vector:
		return &arrow.Int8Type{}, true, nil

	case *int16Vector:
		return &arrow.Int16Type{}, false, nil
	case *nullableInt16Vector:
		return &arrow.Int16Type{}, true, nil

	case *int32Vector:
		return &arrow.Int32Type{}, false, nil
	case *nullableInt32Vector:
		return &arrow.Int32Type{}, true, nil

	case *int64Vector:
		return &arrow.Int64Type{}, false, nil
	case *nullableInt64Vector:
		return &arrow.Int64Type{}, true, nil

	// Uints
	case *uint8Vector:
		return &arrow.Uint8Type{}, false, nil
	case *nullableUint8Vector:
		return &arrow.Uint8Type{}, true, nil

	case *uint16Vector, *enumVector:
		return &arrow.Uint16Type{}, false, nil
	case *nullableUint16Vector, *nullableEnumVector:
		return &arrow.Uint16Type{}, true, nil

	case *uint32Vector:
		return &arrow.Uint32Type{}, false, nil
	case *nullableUint32Vector:
		return &arrow.Uint32Type{}, true, nil

	case *uint64Vector:
		return &arrow.Uint64Type{}, false, nil
	case *nullableUint64Vector:
		return &arrow.Uint64Type{}, true, nil

	case *float32Vector:
		return &arrow.Float32Type{}, false, nil
	case *nullableFloat32Vector:
		return &arrow.Float32Type{}, true, nil

	case *float64Vector:
		return &arrow.Float64Type{}, false, nil
	case *nullableFloat64Vector:
		return &arrow.Float64Type{}, true, nil

	case *boolVector:
		return &arrow.BooleanType{}, false, nil
	case *nullableBoolVector:
		return &arrow.BooleanType{}, true, nil

	case *timeTimeVector:
		return &arrow.TimestampType{Unit: arrow.Nanosecond}, false, nil
	case *nullableTimeTimeVector:
		return &arrow.TimestampType{Unit: arrow.Nanosecond}, true, nil

	case *jsonRawMessageVector:
		return &arrow.BinaryType{}, false, nil
	case *nullableJsonRawMessageVector:
		return &arrow.BinaryType{}, true, nil

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
			sdkField.vector = newNullableStringVector(0)
			break
		}
		sdkField.vector = newStringVector(0)
	case arrow.STRING_VIEW:
		if nullable[idx] {
			sdkField.vector = newNullableStringVector(0)
			break
		}
		sdkField.vector = newStringVector(0)
	case arrow.INT8:
		if nullable[idx] {
			sdkField.vector = newNullableInt8Vector(0)
			break
		}
		sdkField.vector = newInt8Vector(0)
	case arrow.INT16:
		if nullable[idx] {
			sdkField.vector = newNullableInt16Vector(0)
			break
		}
		sdkField.vector = newInt16Vector(0)
	case arrow.INT32:
		if nullable[idx] {
			sdkField.vector = newNullableInt32Vector(0)
			break
		}
		sdkField.vector = newInt32Vector(0)
	case arrow.INT64:
		if nullable[idx] {
			sdkField.vector = newNullableInt64Vector(0)
			break
		}
		sdkField.vector = newInt64Vector(0)
	case arrow.UINT8:
		if nullable[idx] {
			sdkField.vector = newNullableUint8Vector(0)
			break
		}
		sdkField.vector = newUint8Vector(0)
	case arrow.UINT16:
		tstype, ok := getMDKey(metadataKeyTSType, field.Metadata)
		if ok && tstype == simpleTypeEnum {
			if nullable[idx] {
				sdkField.vector = newNullableEnumVector(0)
			} else {
				sdkField.vector = newEnumVector(0)
			}
			break
		}
		if nullable[idx] {
			sdkField.vector = newNullableUint16Vector(0)
			break
		}
		sdkField.vector = newUint16Vector(0)
	case arrow.UINT32:
		if nullable[idx] {
			sdkField.vector = newNullableUint32Vector(0)
			break
		}
		sdkField.vector = newUint32Vector(0)
	case arrow.UINT64:
		if nullable[idx] {
			sdkField.vector = newNullableUint64Vector(0)
			break
		}
		sdkField.vector = newUint64Vector(0)
	case arrow.FLOAT32:
		if nullable[idx] {
			sdkField.vector = newNullableFloat32Vector(0)
			break
		}
		sdkField.vector = newFloat32Vector(0)
	case arrow.FLOAT64:
		if nullable[idx] {
			sdkField.vector = newNullableFloat64Vector(0)
			break
		}
		sdkField.vector = newFloat64Vector(0)
	case arrow.BOOL:
		if nullable[idx] {
			sdkField.vector = newNullableBoolVector(0)
			break
		}
		sdkField.vector = newBoolVector(0)
	case arrow.TIMESTAMP:
		if nullable[idx] {
			sdkField.vector = newNullableTimeTimeVector(0)
			break
		}
		sdkField.vector = newTimeTimeVector(0)
	case arrow.BINARY:
		if nullable[idx] {
			sdkField.vector = newNullableJsonRawMessageVector(0)
			break
		}
		sdkField.vector = newJsonRawMessageVector(0)
	default:
		return fmt.Errorf("unsupported conversion from arrow to sdk type for arrow type %v", field.Type.ID().String())
	}

	return nil
}

func populateFrameFieldsFromRecord(record arrow.Record, nullable []bool, frame *Frame) error { //nolint:staticcheck // SA1019: Using deprecated Record type for backwards compatibility
	frame.SetRowCapacity(int(record.NumRows()))
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
		appendStrings(frame.Fields[i].vector, array.NewStringData(col.Data()), nullable[i])
	case arrow.STRING_VIEW:
		v := array.NewStringViewData(col.Data())
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *string
					frame.Fields[i].vector.Append(ns)
					continue
				}
				rv := strings.Clone(v.Value(rIdx))
				frame.Fields[i].vector.Append(&rv)
				continue
			}
			frame.Fields[i].vector.Append(strings.Clone(v.Value(rIdx)))
		}
	case arrow.INT8:
		v := array.NewInt8Data(col.Data())
		if vec, ok := frame.Fields[i].vector.(*int8Vector); ok {
			*vec = append(*vec, v.Int8Values()...)
			break
		}
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *int8
					frame.Fields[i].vector.Append(ns)
					continue
				}
				rv := v.Value(rIdx)
				frame.Fields[i].vector.Append(&rv)
				continue
			}
			frame.Fields[i].vector.Append(v.Value(rIdx))
		}
	case arrow.INT16:
		v := array.NewInt16Data(col.Data())
		if vec, ok := frame.Fields[i].vector.(*int16Vector); ok {
			*vec = append(*vec, v.Int16Values()...)
			break
		}
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *int16
					frame.Fields[i].vector.Append(ns)
					continue
				}
				rv := v.Value(rIdx)
				frame.Fields[i].vector.Append(&rv)
				continue
			}
			frame.Fields[i].vector.Append(v.Value(rIdx))
		}
	case arrow.INT32:
		v := array.NewInt32Data(col.Data())
		if vec, ok := frame.Fields[i].vector.(*int32Vector); ok {
			*vec = append(*vec, v.Int32Values()...)
			break
		}
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *int32
					frame.Fields[i].vector.Append(ns)
					continue
				}
				rv := v.Value(rIdx)
				frame.Fields[i].vector.Append(&rv)
				continue
			}
			frame.Fields[i].vector.Append(v.Value(rIdx))
		}
	case arrow.INT64:
		v := array.NewInt64Data(col.Data())
		if vec, ok := frame.Fields[i].vector.(*int64Vector); ok {
			*vec = append(*vec, v.Int64Values()...)
			break
		}
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *int64
					frame.Fields[i].vector.Append(ns)
					continue
				}
				rv := v.Value(rIdx)
				frame.Fields[i].vector.Append(&rv)
				continue
			}
			frame.Fields[i].vector.Append(v.Value(rIdx))
		}
	case arrow.UINT8:
		v := array.NewUint8Data(col.Data())
		if vec, ok := frame.Fields[i].vector.(*uint8Vector); ok {
			*vec = append(*vec, v.Uint8Values()...)
			break
		}
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *uint8
					frame.Fields[i].vector.Append(ns)
					continue
				}
				rv := v.Value(rIdx)
				frame.Fields[i].vector.Append(&rv)
				continue
			}
			frame.Fields[i].vector.Append(v.Value(rIdx))
		}
	case arrow.UINT32:
		v := array.NewUint32Data(col.Data())
		if vec, ok := frame.Fields[i].vector.(*uint32Vector); ok {
			*vec = append(*vec, v.Uint32Values()...)
			break
		}
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *uint32
					frame.Fields[i].vector.Append(ns)
					continue
				}
				rv := v.Value(rIdx)
				frame.Fields[i].vector.Append(&rv)
				continue
			}
			frame.Fields[i].vector.Append(v.Value(rIdx))
		}
	case arrow.UINT64:
		v := array.NewUint64Data(col.Data())
		if vec, ok := frame.Fields[i].vector.(*uint64Vector); ok {
			*vec = append(*vec, v.Uint64Values()...)
			break
		}
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if nullable[i] {
				if v.IsNull(rIdx) {
					var ns *uint64
					frame.Fields[i].vector.Append(ns)
					continue
				}
				rv := v.Value(rIdx)
				frame.Fields[i].vector.Append(&rv)
				continue
			}
			frame.Fields[i].vector.Append(v.Value(rIdx))
		}
	case arrow.UINT16:
		v := array.NewUint16Data(col.Data())
		if vec, ok := frame.Fields[i].vector.(*uint16Vector); ok {
			*vec = append(*vec, v.Uint16Values()...)
			break
		}
		for rIdx := 0; rIdx < col.Len(); rIdx++ {
			if frame.Fields[i].Type().NullableType() == FieldTypeNullableEnum {
				if nullable[i] {
					if v.IsNull(rIdx) {
						var ns *EnumItemIndex
						frame.Fields[i].vector.Append(ns)
						continue
					}
					rv := EnumItemIndex(v.Value(rIdx))
					frame.Fields[i].vector.Append(&rv)
					continue
				}
				frame.Fields[i].vector.Append(EnumItemIndex(v.Value(rIdx)))
			} else {
				if nullable[i] {
					if v.IsNull(rIdx) {
						var ns *uint16
						frame.Fields[i].vector.Append(ns)
						continue
					}
					rv := v.Value(rIdx)
					frame.Fields[i].vector.Append(&rv)
					continue
				}
				frame.Fields[i].vector.Append(v.Value(rIdx))
			}
		}
	case arrow.FLOAT32:
		v := array.NewFloat32Data(col.Data())
		if vec, ok := frame.Fields[i].vector.(*float32Vector); ok {
			*vec = append(*vec, v.Float32Values()...)
			break
		}
		for vIdx, f := range v.Float32Values() {
			if nullable[i] {
				if v.IsNull(vIdx) {
					var nf *float32
					frame.Fields[i].vector.Append(nf)
					continue
				}
				vF := f
				frame.Fields[i].vector.Append(&vF)
				continue
			}
			frame.Fields[i].vector.Append(f)
		}
	case arrow.FLOAT64:
		v := array.NewFloat64Data(col.Data())
		if vec, ok := frame.Fields[i].vector.(*float64Vector); ok {
			*vec = append(*vec, v.Float64Values()...)
			break
		}
		for vIdx, f := range v.Float64Values() {
			if nullable[i] {
				if v.IsNull(vIdx) {
					var nf *float64
					frame.Fields[i].vector.Append(nf)
					continue
				}
				vF := f
				frame.Fields[i].vector.Append(&vF)
				continue
			}
			frame.Fields[i].vector.Append(f)
		}
	case arrow.BOOL:
		v := array.NewBooleanData(col.Data())
		if vec, ok := frame.Fields[i].vector.(*boolVector); ok {
			for sIdx := 0; sIdx < col.Len(); sIdx++ {
				*vec = append(*vec, v.Value(sIdx))
			}
			break
		}
		for sIdx := 0; sIdx < col.Len(); sIdx++ {
			if nullable[i] {
				if v.IsNull(sIdx) {
					var ns *bool
					frame.Fields[i].vector.Append(ns)
					continue
				}
				vB := v.Value(sIdx)
				frame.Fields[i].vector.Append(&vB)
				continue
			}
			frame.Fields[i].vector.Append(v.Value(sIdx))
		}
	case arrow.TIMESTAMP:
		v := array.NewTimestampData(col.Data())
		if vec, ok := frame.Fields[i].vector.(*timeTimeVector); ok {
			for _, ts := range v.TimestampValues() {
				*vec = append(*vec, time.Unix(0, int64(ts)))
			}
			break
		}
		for vIdx, ts := range v.TimestampValues() {
			t := time.Unix(0, int64(ts)) // nanosecond assumption
			if nullable[i] {
				if v.IsNull(vIdx) {
					var nt *time.Time
					frame.Fields[i].vector.Append(nt)
					continue
				}
				frame.Fields[i].vector.Append(&t)
				continue
			}
			frame.Fields[i].vector.Append(t)
		}
	case arrow.BINARY:
		v := array.NewBinaryData(col.Data())
		if v.Len() == 0 {
			break
		}
		values := bytes.Clone(v.ValueBytes())
		offsets := v.ValueOffsets()
		for sIdx := 0; sIdx < v.Len(); sIdx++ {
			if nullable[i] && v.IsNull(sIdx) {
				var nb *json.RawMessage
				frame.Fields[i].vector.Append(nb)
				continue
			}
			start, end := offsets[sIdx]-offsets[0], offsets[sIdx+1]-offsets[0]
			r := json.RawMessage(values[start:end:end])
			if nullable[i] {
				frame.Fields[i].vector.Append(&r)
				continue
			}
			frame.Fields[i].vector.Append(r)
		}
	default:
		return fmt.Errorf("unsupported arrow type %s for conversion", col.DataType().ID())
	}

	return nil
}

// appendStrings copies the column's bytes once so the decoded strings do not alias the input.
func appendStrings(vec vector, v *array.String, nullable bool) {
	n := v.Len()
	if n == 0 {
		return
	}
	values := string(v.ValueBytes())
	offsets := v.ValueOffsets()
	value := func(r int) string { return values[offsets[r]-offsets[0] : offsets[r+1]-offsets[0]] }
	if sv, ok := vec.(*stringVector); ok {
		for r := range n {
			*sv = append(*sv, value(r))
		}
		return
	}
	for r := range n {
		if nullable && v.IsNull(r) {
			var ns *string
			vec.Append(ns)
			continue
		}
		s := value(r)
		if nullable {
			vec.Append(&s)
			continue
		}
		vec.Append(s)
	}
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
func FromArrowRecord(record arrow.Record) (*Frame, error) { //nolint:staticcheck // SA1019: Using deprecated Record type for backwards compatibility
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

// UnmarshalArrowFrame decodes an Arrow IPC file into a Frame.
// The returned Frame does not retain b.
func UnmarshalArrowFrame(b []byte) (*Frame, error) {
	fR, err := newFileReader(b)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fR.Close() }()

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

// newFileReader falls back to ipc.NewFileReader because ipc.NewMappedFileReader panics on dictionary-encoded input (apache/arrow-go#1364).
func newFileReader(b []byte) (r *ipc.FileReader, err error) {
	defer func() {
		if recover() != nil {
			r, err = ipc.NewFileReader(bytes.NewReader(b))
		}
	}()
	return ipc.NewMappedFileReader(b)
}

// ToJSONString calls json.Marshal on val and returns it as a string. An
// error is returned if json.Marshal errors.
func toJSONString(val any) (string, error) {
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

// MarshalArrow encodes Frames into a slice of []byte, one Arrow IPC file per Frame.
// Each returned slice has no spare capacity. If an error occurs [][]byte will be nil.
// See UnmarshalArrowFrames for the inverse operation.
func (frames Frames) MarshalArrow() ([][]byte, error) {
	bs := make([][]byte, len(frames))
	var buf bytes.Buffer
	for i, frame := range frames {
		if frame == nil {
			return nil, errors.New("frame can not be nil")
		}
		buf.Reset()
		if err := frame.marshalArrow(&buf); err != nil {
			return nil, err
		}
		bs[i] = exactSizeCopy(buf.Bytes())
	}
	return bs, nil
}

func exactSizeCopy(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
