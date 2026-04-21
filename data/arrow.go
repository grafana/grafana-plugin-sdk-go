package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
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

// Object pools for frequently allocated objects to reduce GC pressure
var (
	// arrowAllocatorPool pools Arrow memory allocators to avoid repeated allocation overhead
	arrowAllocatorPool = sync.Pool{
		New: func() interface{} { return memory.NewGoAllocator() },
	}

	// fileBufferPool pools filebuffer.Buffer instances for Arrow marshaling/unmarshaling
	fileBufferPool = sync.Pool{
		New: func() interface{} { return filebuffer.New(nil) },
	}
)

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

	// Get filebuffer from pool to reduce allocations
	fb := fileBufferPool.Get().(*filebuffer.Buffer)
	fb.Buff.Reset() // Reset buffer for reuse
	defer fileBufferPool.Put(fb)

	fw, err := ipc.NewFileWriter(fb, ipc.WithSchema(tableReader.Schema()))
	if err != nil {
		return nil, err
	}

	for tableReader.Next() {
		rec := tableReader.Record() //nolint:staticcheck // SA1019: Using deprecated Record() API for backwards compatibility

		if err := fw.Write(rec); err != nil {
			rec.Release()
			return nil, err
		}
		rec.Release()
	}

	if err := fw.Close(); err != nil {
		return nil, err
	}

	// Copy bytes before returning buffer to pool
	result := make([]byte, fb.Buff.Len())
	copy(result, fb.Buff.Bytes())
	return result, nil
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
	// Get allocator from pool to reduce allocation overhead
	pool := arrowAllocatorPool.Get().(memory.Allocator)
	defer arrowAllocatorPool.Put(pool)

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

func initializeFrameFields(schema *arrow.Schema, frame *Frame, capacity int) ([]bool, error) {
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
		if err := initializeFrameField(field, idx, nullable, &sdkField, capacity); err != nil {
			return nil, err
		}

		frame.Fields = append(frame.Fields, &sdkField)
	}
	return nullable, nil
}

// nolint:gocyclo
func initializeFrameField(field arrow.Field, idx int, nullable []bool, sdkField *Field, capacity int) error {
	switch field.Type.ID() {
	case arrow.STRING:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[string](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[string](capacity)
	case arrow.STRING_VIEW:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[string](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[string](capacity)
	case arrow.INT8:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[int8](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[int8](capacity)
	case arrow.INT16:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[int16](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[int16](capacity)
	case arrow.INT32:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[int32](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[int32](capacity)
	case arrow.INT64:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[int64](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[int64](capacity)
	case arrow.UINT8:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[uint8](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[uint8](capacity)
	case arrow.UINT16:
		tstype, ok := getMDKey(metadataKeyTSType, field.Metadata)
		if ok && tstype == simpleTypeEnum {
			if nullable[idx] {
				sdkField.vector = newNullableGenericVectorWithCapacity[EnumItemIndex](capacity)
			} else {
				sdkField.vector = newGenericVectorWithCapacity[EnumItemIndex](capacity)
			}
			break
		}
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[uint16](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[uint16](capacity)
	case arrow.UINT32:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[uint32](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[uint32](capacity)
	case arrow.UINT64:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[uint64](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[uint64](capacity)
	case arrow.FLOAT32:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[float32](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[float32](capacity)
	case arrow.FLOAT64:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[float64](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[float64](capacity)
	case arrow.BOOL:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[bool](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[bool](capacity)
	case arrow.TIMESTAMP:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[time.Time](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[time.Time](capacity)
	case arrow.BINARY:
		if nullable[idx] {
			sdkField.vector = newNullableGenericVectorWithCapacity[json.RawMessage](capacity)
			break
		}
		sdkField.vector = newGenericVectorWithCapacity[json.RawMessage](capacity)
	default:
		return fmt.Errorf("unsupported conversion from arrow to sdk type for arrow type %v", field.Type.ID().String())
	}

	return nil
}

func populateFrameFieldsFromRecord(record arrow.Record, nullable []bool, frame *Frame) error { //nolint:staticcheck // SA1019: Using deprecated Record type for backwards compatibility
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
		// Note: True zero-copy isn't possible for strings in Go due to immutability
		// Arrow's Value() method already optimizes string conversion internally
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
		// STRING_VIEW is already optimized in Arrow for avoiding copies
		// Our vectors still need to materialize strings (Go immutability requirement)
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
		values := v.Int8Values()
		if nullable[i] {
			if nvec, ok := frame.Fields[i].vector.(*nullableGenericVector[int8]); ok {
				nvec.AppendManyWithNulls(values, v.IsNull)
			} else {
				for rIdx := range values {
					if v.IsNull(rIdx) {
						var ns *int8
						appendTypedToVector(frame.Fields[i].vector, ns)
						continue
					}
					rv := values[rIdx]
					appendTypedToVector(frame.Fields[i].vector, &rv)
				}
			}
		} else {
			if gvec, ok := frame.Fields[i].vector.(*genericVector[int8]); ok {
				gvec.AppendManyTyped(values)
			} else {
				for _, val := range values {
					appendTypedToVector(frame.Fields[i].vector, val)
				}
			}
		}
	case arrow.INT16:
		v := array.NewInt16Data(col.Data())
		values := v.Int16Values()
		if nullable[i] {
			if nvec, ok := frame.Fields[i].vector.(*nullableGenericVector[int16]); ok {
				nvec.AppendManyWithNulls(values, v.IsNull)
			} else {
				for rIdx := range values {
					if v.IsNull(rIdx) {
						var ns *int16
						appendTypedToVector(frame.Fields[i].vector, ns)
						continue
					}
					rv := values[rIdx]
					appendTypedToVector(frame.Fields[i].vector, &rv)
				}
			}
		} else {
			if gvec, ok := frame.Fields[i].vector.(*genericVector[int16]); ok {
				gvec.AppendManyTyped(values)
			} else {
				for _, val := range values {
					appendTypedToVector(frame.Fields[i].vector, val)
				}
			}
		}
	case arrow.INT32:
		v := array.NewInt32Data(col.Data())
		values := v.Int32Values()
		if nullable[i] {
			if nvec, ok := frame.Fields[i].vector.(*nullableGenericVector[int32]); ok {
				nvec.AppendManyWithNulls(values, v.IsNull)
			} else {
				for rIdx := range values {
					if v.IsNull(rIdx) {
						var ns *int32
						appendTypedToVector(frame.Fields[i].vector, ns)
						continue
					}
					rv := values[rIdx]
					appendTypedToVector(frame.Fields[i].vector, &rv)
				}
			}
		} else {
			if gvec, ok := frame.Fields[i].vector.(*genericVector[int32]); ok {
				gvec.AppendManyTyped(values)
			} else {
				for _, val := range values {
					appendTypedToVector(frame.Fields[i].vector, val)
				}
			}
		}
	case arrow.INT64:
		v := array.NewInt64Data(col.Data())
		// Use zero-copy API to get direct slice access
		values := v.Int64Values()
		if nullable[i] {
			// Batch append with null handling
			if nvec, ok := frame.Fields[i].vector.(*nullableGenericVector[int64]); ok {
				nvec.AppendManyWithNulls(values, v.IsNull)
			} else {
				// Fallback for unexpected vector type
				for rIdx := range values {
					if v.IsNull(rIdx) {
						var ns *int64
						appendTypedToVector(frame.Fields[i].vector, ns)
						continue
					}
					rv := values[rIdx]
					appendTypedToVector(frame.Fields[i].vector, &rv)
				}
			}
		} else {
			// Non-nullable: direct batch append
			if gvec, ok := frame.Fields[i].vector.(*genericVector[int64]); ok {
				gvec.AppendManyTyped(values)
			} else {
				// Fallback
				for _, val := range values {
					appendTypedToVector(frame.Fields[i].vector, val)
				}
			}
		}
	case arrow.UINT8:
		v := array.NewUint8Data(col.Data())
		values := v.Uint8Values()
		if nullable[i] {
			if nvec, ok := frame.Fields[i].vector.(*nullableGenericVector[uint8]); ok {
				nvec.AppendManyWithNulls(values, v.IsNull)
			} else {
				for rIdx := range values {
					if v.IsNull(rIdx) {
						var ns *uint8
						appendTypedToVector(frame.Fields[i].vector, ns)
						continue
					}
					rv := values[rIdx]
					appendTypedToVector(frame.Fields[i].vector, &rv)
				}
			}
		} else {
			if gvec, ok := frame.Fields[i].vector.(*genericVector[uint8]); ok {
				gvec.AppendManyTyped(values)
			} else {
				for _, val := range values {
					appendTypedToVector(frame.Fields[i].vector, val)
				}
			}
		}
	case arrow.UINT32:
		v := array.NewUint32Data(col.Data())
		values := v.Uint32Values()
		if nullable[i] {
			if nvec, ok := frame.Fields[i].vector.(*nullableGenericVector[uint32]); ok {
				nvec.AppendManyWithNulls(values, v.IsNull)
			} else {
				for rIdx := range values {
					if v.IsNull(rIdx) {
						var ns *uint32
						appendTypedToVector(frame.Fields[i].vector, ns)
						continue
					}
					rv := values[rIdx]
					appendTypedToVector(frame.Fields[i].vector, &rv)
				}
			}
		} else {
			if gvec, ok := frame.Fields[i].vector.(*genericVector[uint32]); ok {
				gvec.AppendManyTyped(values)
			} else {
				for _, val := range values {
					appendTypedToVector(frame.Fields[i].vector, val)
				}
			}
		}
	case arrow.UINT64:
		v := array.NewUint64Data(col.Data())
		values := v.Uint64Values()
		if nullable[i] {
			if nvec, ok := frame.Fields[i].vector.(*nullableGenericVector[uint64]); ok {
				nvec.AppendManyWithNulls(values, v.IsNull)
			} else {
				for rIdx := range values {
					if v.IsNull(rIdx) {
						var ns *uint64
						appendTypedToVector(frame.Fields[i].vector, ns)
						continue
					}
					rv := values[rIdx]
					appendTypedToVector(frame.Fields[i].vector, &rv)
				}
			}
		} else {
			if gvec, ok := frame.Fields[i].vector.(*genericVector[uint64]); ok {
				gvec.AppendManyTyped(values)
			} else {
				for _, val := range values {
					appendTypedToVector(frame.Fields[i].vector, val)
				}
			}
		}
	case arrow.UINT16:
		v := array.NewUint16Data(col.Data())
		values := v.Uint16Values()
		if frame.Fields[i].Type().NonNullableType() == FieldTypeEnum {
			// Handle Enum type
			if nullable[i] {
				if nvec, ok := frame.Fields[i].vector.(*nullableGenericVector[EnumItemIndex]); ok {
					// Convert []uint16 to []EnumItemIndex efficiently
					enumValues := make([]EnumItemIndex, len(values))
					for idx, val := range values {
						enumValues[idx] = EnumItemIndex(val)
					}
					nvec.AppendManyWithNulls(enumValues, v.IsNull)
				} else {
					for rIdx := range values {
						if v.IsNull(rIdx) {
							var ns *EnumItemIndex
							appendTypedToVector(frame.Fields[i].vector, ns)
							continue
						}
						rv := EnumItemIndex(values[rIdx])
						appendTypedToVector(frame.Fields[i].vector, &rv)
					}
				}
			} else {
				if gvec, ok := frame.Fields[i].vector.(*genericVector[EnumItemIndex]); ok {
					enumValues := make([]EnumItemIndex, len(values))
					for idx, val := range values {
						enumValues[idx] = EnumItemIndex(val)
					}
					gvec.AppendManyTyped(enumValues)
				} else {
					for _, val := range values {
						appendTypedToVector(frame.Fields[i].vector, EnumItemIndex(val))
					}
				}
			}
		} else {
			// Handle regular uint16
			if nullable[i] {
				if nvec, ok := frame.Fields[i].vector.(*nullableGenericVector[uint16]); ok {
					nvec.AppendManyWithNulls(values, v.IsNull)
				} else {
					for rIdx := range values {
						if v.IsNull(rIdx) {
							var ns *uint16
							appendTypedToVector(frame.Fields[i].vector, ns)
							continue
						}
						rv := values[rIdx]
						appendTypedToVector(frame.Fields[i].vector, &rv)
					}
				}
			} else {
				if gvec, ok := frame.Fields[i].vector.(*genericVector[uint16]); ok {
					gvec.AppendManyTyped(values)
				} else {
					for _, val := range values {
						appendTypedToVector(frame.Fields[i].vector, val)
					}
				}
			}
		}
	case arrow.FLOAT32:
		v := array.NewFloat32Data(col.Data())
		values := v.Float32Values()
		if nullable[i] {
			if nvec, ok := frame.Fields[i].vector.(*nullableGenericVector[float32]); ok {
				nvec.AppendManyWithNulls(values, v.IsNull)
			} else {
				for vIdx, f := range values {
					if v.IsNull(vIdx) {
						var nf *float32
						appendTypedToVector(frame.Fields[i].vector, nf)
						continue
					}
					vF := f
					appendTypedToVector(frame.Fields[i].vector, &vF)
				}
			}
		} else {
			if gvec, ok := frame.Fields[i].vector.(*genericVector[float32]); ok {
				gvec.AppendManyTyped(values)
			} else {
				for _, f := range values {
					appendTypedToVector(frame.Fields[i].vector, f)
				}
			}
		}
	case arrow.FLOAT64:
		v := array.NewFloat64Data(col.Data())
		values := v.Float64Values()
		if nullable[i] {
			if nvec, ok := frame.Fields[i].vector.(*nullableGenericVector[float64]); ok {
				nvec.AppendManyWithNulls(values, v.IsNull)
			} else {
				for vIdx, f := range values {
					if v.IsNull(vIdx) {
						var nf *float64
						appendTypedToVector(frame.Fields[i].vector, nf)
						continue
					}
					vF := f
					appendTypedToVector(frame.Fields[i].vector, &vF)
				}
			}
		} else {
			if gvec, ok := frame.Fields[i].vector.(*genericVector[float64]); ok {
				gvec.AppendManyTyped(values)
			} else {
				for _, f := range values {
					appendTypedToVector(frame.Fields[i].vector, f)
				}
			}
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
		timestamps := v.TimestampValues()
		// Convert Arrow timestamps to time.Time (nanosecond assumption)
		times := make([]time.Time, len(timestamps))
		for idx, ts := range timestamps {
			times[idx] = time.Unix(0, int64(ts))
		}

		if nullable[i] {
			if nvec, ok := frame.Fields[i].vector.(*nullableGenericVector[time.Time]); ok {
				nvec.AppendManyWithNulls(times, v.IsNull)
			} else {
				for vIdx, t := range times {
					if v.IsNull(vIdx) {
						var nt *time.Time
						appendTypedToVector(frame.Fields[i].vector, nt)
						continue
					}
					appendTypedToVector(frame.Fields[i].vector, &t)
				}
			}
		} else {
			if gvec, ok := frame.Fields[i].vector.(*genericVector[time.Time]); ok {
				gvec.AppendManyTyped(times)
			} else {
				for _, t := range times {
					appendTypedToVector(frame.Fields[i].vector, t)
				}
			}
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
func FromArrowRecord(record arrow.Record) (*Frame, error) { //nolint:staticcheck // SA1019: Using deprecated Record type for backwards compatibility
	schema := record.Schema()
	frame := &Frame{}
	if err := populateFrameFromSchema(schema, frame); err != nil {
		return nil, err
	}

	// Pre-allocate vectors with the known row count for better performance
	capacity := int(record.NumRows())
	nullable, err := initializeFrameFields(schema, frame, capacity)
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
	// Get filebuffer from pool to reduce allocations
	fB := fileBufferPool.Get().(*filebuffer.Buffer)
	fB.Buff.Reset()
	fB.Buff.Write(b)
	defer fileBufferPool.Put(fB)

	fR, err := ipc.NewFileReader(fB)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fR.Close() }()

	schema := fR.Schema()
	frame := &Frame{}
	if err := populateFrameFromSchema(schema, frame); err != nil {
		return nil, err
	}

	// Calculate total capacity by reading all record batch sizes
	// This pre-allocates vectors to avoid repeated reallocations
	capacity := 0
	for i := 0; i < fR.NumRecords(); i++ {
		rec, err := fR.RecordBatch(i)
		if err != nil {
			return nil, err
		}
		capacity += int(rec.NumRows())
		rec.Release()
	}

	nullable, err := initializeFrameFields(schema, frame, capacity)
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
