package dataframe

import (
	"database/sql"
	"fmt"
	"reflect"
)

// NewFromSQLRows returns a new dataframe populated with the data from the rows.
// Fields will be named to match name of the SQL columns. The SQL column names must be
// unique. All the types must be supported by in dataframe or there must be a SQLStringConverter
// for columns that have types that do not match. The SQLStringConverter's ConversionFunc will
// be applied to matching rows if it is not nil. A map of Field/Column index to the corresponding SQLStringConverter is used so once can
// do additional frame modifications.
// If the database driver does not indicate if the columns are nullable, all columns are assumed to be nullable.
func NewFromSQLRows(rows *sql.Rows, converters ...SQLStringConverter) (*Frame, map[int]SQLStringConverter, error) {
	frame, mappers, err := newForSQLRows(rows, converters...)
	if err != nil {
		return nil, nil, err
	}

	for rows.Next() {
		sRow := frame.scannableRow()
		err := rows.Scan(sRow...)
		if err != nil {
			return nil, nil, err
		}
	}

	for fieldIdx, mapper := range mappers {
		if mapper.ConversionFunc == nil {
			continue
		}
		vec := frame.Fields[fieldIdx]
		for i := 0; i < vec.Len(); i++ {
			v, err := mapper.ConversionFunc(vec.Vector.At(i).(*string))
			if err != nil {
				return nil, nil, err
			}
			vec.Vector.Set(i, v)
		}
		if mapper.Replacer == nil {
			continue
		}
		if err := Replace(frame, fieldIdx, mapper.Replacer); err != nil {
			return nil, nil, err
		}
	}

	return frame, mappers, nil
}

// newForSQLRows creates a new Frame appropriate for scanning SQL rows with
// the the new Frame's ScannableRow() method.
func newForSQLRows(rows *sql.Rows, converters ...SQLStringConverter) (*Frame, map[int]SQLStringConverter, error) {
	mapping := make(map[int]SQLStringConverter)
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, err
	}
	colNames, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]struct{}{}
	for _, name := range colNames {
		if _, ok := seen[name]; ok {
			return nil, nil, fmt.Errorf(`duplicate column names no allowed, found identical name: "%v"`, name)
		}
		seen[name] = struct{}{}
	}
	frame := &Frame{}
	for i, colType := range colTypes {
		colName := colNames[i]
		nullable, ok := colType.Nullable()
		if !ok {
			nullable = true // If we don't know if it is nullable, assume it is
		}
		scanType := colType.ScanType()
		for _, converter := range converters {
			if converter.InputScanKind == scanType.Kind() && converter.InputTypeName == colType.DatabaseTypeName() {
				nullable = true // String converters are always nullable
				scanType = reflect.TypeOf("")
				mapping[i] = converter
			}
		}
		var vec interface{}
		if !nullable {
			vec = reflect.MakeSlice(reflect.SliceOf(scanType), 0, 0).Interface()
		} else {
			ptrType := reflect.TypeOf(reflect.New(scanType).Interface())
			// Nullabe types get passed to scan as a pointer to a pointer
			vec = reflect.MakeSlice(reflect.SliceOf(ptrType), 0, 0).Interface()
		}
		if !ValidVectorType(vec) {
			// Automatically create string mapper if we end up with an unsupported type
			mapping[i] = SQLStringConverter{
				Name:          fmt.Sprintf("Autogenerated for column %v", i),
				InputTypeName: colType.DatabaseTypeName(),
				InputScanKind: colType.ScanType().Kind(),
			}
			ptrType := reflect.TypeOf(reflect.New(reflect.TypeOf("")).Interface())
			vec = reflect.MakeSlice(reflect.SliceOf(ptrType), 0, 0).Interface()
		}
		frame.Fields = append(frame.Fields, NewField(colName, nil, vec))
	}
	return frame, mapping, nil
}

// scannableRow adds a row to the dataframe, and returns a slice of references
// that can be passed to rows.Scan() in the in sql package.
func (f *Frame) scannableRow() []interface{} {
	row := make([]interface{}, len(f.Fields))
	for i, field := range f.Fields {
		vec := field.Vector
		vec.Extend(1)
		vecItemPointer := vec.PointerAt(vec.Len() - 1)
		row[i] = vecItemPointer
	}
	return row
}

// SQLStringConverter can be used to store types not supported by
// a dataframe into a string. When scanning, if a SQL's row's InputScanType's Kind
// and InputScanKind match that returned by the sql response, then the
// conversion func will be run on the row.
type SQLStringConverter struct {
	// Name is an optional property that can be used to identify a converter
	Name          string
	InputScanKind reflect.Kind
	InputTypeName string

	// Conversion func may be nil to do no additional operations on the string conversion.
	ConversionFunc func(in *string) (*string, error)

	// If the Replacer is not nil, the replacement will be performed.
	Replacer *StringFieldReplacer
}

// StringFieldReplacer is used to replace a *string Field in a dataframe. The type
// returned by the ReplaceFunc must match the elements of VectorType.
type StringFieldReplacer struct {
	VectorType  interface{}
	ReplaceFunc func(in *string) (interface{}, error)
}

// Replace will replace a *string Vector of the specified Field's index
// using the StringFieldReplacer.
func Replace(frame *Frame, fieldIdx int, replacer *StringFieldReplacer) error {
	if fieldIdx > len(frame.Fields) {
		return fmt.Errorf("fieldIdx is out of bounds, field len: %v", len(frame.Fields))
	}
	field := frame.Fields[fieldIdx]
	if field.Vector.PrimitiveType() != VectorPTypeNullableString {
		return fmt.Errorf("can only replace []*string vectors, vector is of type %s", field.Vector.PrimitiveType())
	}

	if !ValidVectorType(replacer.VectorType) {
		return fmt.Errorf("can not replace column with unsupported type %T", replacer.VectorType)
	}
	newVector := newVector(replacer.VectorType, field.Vector.Len())
	for i := 0; i < newVector.Len(); i++ {
		oldVal := field.Vector.At(i).(*string)
		newVal, err := replacer.ReplaceFunc(oldVal)
		if err != nil {
			return err
		}
		newVector.Set(i, newVal)
	}
	field.Vector = newVector
	return nil
}
