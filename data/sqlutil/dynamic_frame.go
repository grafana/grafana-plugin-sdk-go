package sqlutil

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/converters"
)

const STRING = "string"

// removeDynamicConverter filters out the dynamic converter.  It is not a valid converter.
// Deprecated: This function is preserved for backward compatibility with the legacy Dynamic field.
func removeDynamicConverter(converters []Converter) (bool, []Converter) {
	var filtered []Converter
	var isDynamic bool
	for _, conv := range converters {
		if conv.Dynamic {
			isDynamic = true
		} else {
			filtered = append(filtered, conv)
		}
	}
	return isDynamic, filtered
}

// findDynamicPerColumnConverters identifies which column indices should use dynamic type inference
// based on the DynamicPerColumn field. Returns a map of column indices and a filtered list of converters.
func findDynamicPerColumnConverters(colTypes []*sql.ColumnType, convs []Converter) (map[int]bool, []Converter) {
	dynamicIndices := make(map[int]bool)
	var filtered []Converter

	for _, conv := range convs {
		if conv.DynamicPerColumn {
			// Mark all columns that match this dynamic converter
			for i, colType := range colTypes {
				if converterMatches(conv, colType.DatabaseTypeName(), colType.Name()) {
					dynamicIndices[i] = true
				}
			}
		} else {
			filtered = append(filtered, conv)
		}
	}

	return dynamicIndices, filtered
}

func findDataTypes(rows Rows, rowLimit int64, types []*sql.ColumnType) ([]Field, [][]interface{}, error) {
	var i int64
	fields := make(map[int]Field)

	var returnData [][]interface{}

	for {
		for rows.Next() {
			if i == rowLimit {
				break
			}
			row := make([]interface{}, len(types))
			for i := range row {
				row[i] = new(interface{})
			}
			err := rows.Scan(row)
			if err != nil {
				return nil, nil, err
			}

			returnData = append(returnData, row)

			if len(fields) == len(types) {
				// found all data types.  keep looping to load all the return data
				continue
			}

			for colIdx, col := range row {
				val := *col.(*interface{})
				var field Field
				colType := types[colIdx]
				switch val.(type) {
				case time.Time, *time.Time:
					field.converter = &TimeToNullableTime
					field.kind = "time"
					field.name = colType.Name()
				case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
					field.converter = &IntOrFloatToNullableFloat64
					field.kind = "float64"
					field.name = colType.Name()
				case string:
					field.converter = &converters.AnyToNullableString
					field.kind = STRING
					field.name = colType.Name()
				case []uint8:
					field.converter = &converters.Uint8ArrayToNullableString
					field.kind = STRING
					field.name = colType.Name()
				case nil:
					continue
				default:
					field.converter = &converters.AnyToNullableString
					field.kind = STRING
					field.name = colType.Name()
				}

				fields[colIdx] = field
			}

			i++
		}
		if i == rowLimit || !rows.NextResultSet() {
			break
		}
	}

	fieldList := make([]Field, len(types))
	for colIdx, col := range types {
		field, ok := fields[colIdx]
		field.name = col.Name()
		if !ok {
			field = Field{
				converter: &converters.AnyToNullableString,
				kind:      "string",
				name:      col.Name(),
			}
		}
		fieldList[colIdx] = field
	}

	return fieldList, returnData, nil
}

// frameDynamic uses runtime type inference for ALL columns in the frame (legacy behavior).
// This is preserved for backward compatibility with the deprecated Dynamic field.
// For better type fidelity, use frameHybrid with DynamicPerColumn instead.
// Deprecated: This function maintains the legacy Dynamic field behavior.
func frameDynamic(rows Rows, rowLimit int64, types []*sql.ColumnType, convs []Converter) (*data.Frame, error) {
	// find data type(s) from the data
	fields, rawRows, err := findDataTypes(rows, rowLimit, types)
	if err != nil {
		return nil, err
	}

	// if a converter is defined by column name, override data type that was found
	fields = overrideConverter(fields, convs)

	frameFields := make(data.Fields, len(fields))
	for i, f := range fields {
		frameFields[i] = data.NewFieldFromFieldType(f.converter.OutputFieldType, 0)
		frameFields[i].Name = f.name
	}

	frame := data.NewFrame("", frameFields...)

	for _, row := range rawRows {
		var rowData []interface{}

		for colIdx, col := range row {
			field := fields[colIdx]

			val := col
			ptr, ok := col.(*interface{})
			if ok {
				val = *ptr
			}

			val, err := field.converter.Converter(val)
			if err != nil {
				return nil, err
			}

			rowData = append(rowData, val)
		}
		frame.AppendRow(rowData...)
	}

	return frame, nil
}

// if a converter is defined by column name, override data type that was found
func overrideConverter(fields []Field, converters []Converter) []Field {
	var overrides []Field
	for _, field := range fields {
		converter := field.converter
		for _, c := range converters {
			if c.InputColumnName == field.name {
				var conv = data.FieldConverter{
					OutputFieldType: c.FrameConverter.FieldType,
					Converter:       c.FrameConverter.ConverterFunc,
				}
				converter = &conv
				break
			}
		}
		override := Field{
			name:      field.name,
			converter: converter,
			kind:      field.kind,
		}
		overrides = append(overrides, override)
	}
	return overrides
}

// frameHybrid processes a frame with a mix of static and dynamic columns.
// Dynamic columns (identified in dynamicIndices) use runtime type inference,
// while static columns use SQL-type-based converters.
func frameHybrid(rows Rows, rowLimit int64, types []*sql.ColumnType, convs []Converter, dynamicIndices map[int]bool) (*data.Frame, error) {
	// First pass: scan rows and infer types for dynamic columns
	fields, rawRows, limitReached, err := findDataTypesHybrid(rows, rowLimit, types, convs, dynamicIndices)
	if err != nil {
		return nil, err
	}

	// Create frame fields
	frameFields := make(data.Fields, len(fields))
	for i, f := range fields {
		frameFields[i] = data.NewFieldFromFieldType(f.converter.OutputFieldType, 0)
		frameFields[i].Name = f.name
	}

	frame := data.NewFrame("", frameFields...)

	// Second pass: convert and append rows
	for _, row := range rawRows {
		var rowData []interface{}
		for colIdx, col := range row {
			field := fields[colIdx]

			val := col
			ptr, ok := col.(*interface{})
			if ok {
				val = *ptr
			}

			val, err := field.converter.Converter(val)
			if err != nil {
				return nil, err
			}

			rowData = append(rowData, val)
		}
		frame.AppendRow(rowData...)
	}

	if limitReached {
		frame.AppendNotices(data.Notice{
			Severity: data.NoticeSeverityWarning,
			Text:     fmt.Sprintf("Results have been limited to %v because the SQL row limit was reached", rowLimit),
		})
	}

	return frame, nil
}

// findDataTypesHybrid determines field types for all columns via runtime type inference,
// scanning everything into *interface{} (same approach as findDataTypes/frameDynamic).
// dynamicIndices is used only to track when all dynamic columns have been resolved so
// we can stop early once every column's type is known.
// Returns limitReached=true when rowLimit rows were consumed.
//
// Note: because all columns are scanned into *interface{}, typed SQL converters that
// expect specific scan-type pointers (e.g. *sql.NullInt32) cannot be used here.
// Type fidelity for static columns is achieved through overrideConverter using
// InputColumnName-based converters whose ConverterFunc accepts interface{} values.
func findDataTypesHybrid(rows Rows, rowLimit int64, types []*sql.ColumnType, convs []Converter, dynamicIndices map[int]bool) ([]Field, [][]interface{}, bool, error) {
	var i int64
	fields := make(map[int]Field)

	var returnData [][]interface{}

	for {
		for rows.Next() {
			if i == rowLimit {
				break
			}
			row := make([]interface{}, len(types))
			for j := range row {
				row[j] = new(interface{})
			}
			err := rows.Scan(row)
			if err != nil {
				return nil, nil, false, err
			}

			returnData = append(returnData, row)

			if len(fields) < len(types) {
				// Infer types from this row for all columns not yet resolved
				for colIdx, col := range row {
					if _, ok := fields[colIdx]; ok {
						continue // Type already determined for this column
					}

					val := *col.(*interface{})
					var field Field
					colType := types[colIdx]
					switch val.(type) {
					case time.Time, *time.Time:
						field.converter = &TimeToNullableTime
						field.kind = "time"
						field.name = colType.Name()
					case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
						field.converter = &IntOrFloatToNullableFloat64
						field.kind = "float64"
						field.name = colType.Name()
					case string:
						field.converter = &converters.AnyToNullableString
						field.kind = STRING
						field.name = colType.Name()
					case []uint8:
						field.converter = &converters.Uint8ArrayToNullableString
						field.kind = STRING
						field.name = colType.Name()
					case nil:
						continue // Can't infer from null, wait for next row
					default:
						field.converter = &converters.AnyToNullableString
						field.kind = STRING
						field.name = colType.Name()
					}

					fields[colIdx] = field
				}
			}

			i++
		}
		if i == rowLimit || !rows.NextResultSet() {
			break
		}
	}

	limitReached := i == rowLimit

	// Build complete field list, defaulting to string for any column with no non-null rows
	fieldList := make([]Field, len(types))
	for colIdx, colType := range types {
		field, ok := fields[colIdx]
		if !ok {
			field = Field{
				converter: &converters.AnyToNullableString,
				kind:      "string",
				name:      colType.Name(),
			}
		}
		field.name = colType.Name()
		fieldList[colIdx] = field
	}

	// Apply column-name-specific converters (must use ConverterFunc that accepts interface{})
	fieldList = overrideConverter(fieldList, convs)

	return fieldList, returnData, limitReached, nil
}


type Field struct {
	name      string
	converter *data.FieldConverter
	kind      string
}

type ResultSetIterator interface {
	NextResultSet() bool
}

type RowIterator interface {
	Next() bool
	Scan(dest ...interface{}) error
}

type Rows struct {
	itr RowIterator
}

func (rs Rows) NextResultSet() bool {
	if itr, has := rs.itr.(ResultSetIterator); has {
		return itr.NextResultSet()
	}
	return false
}

func (rs Rows) Next() bool {
	return rs.itr.Next()
}

func (rs Rows) Scan(dest []interface{}) error {
	return rs.itr.Scan(dest...)
}
