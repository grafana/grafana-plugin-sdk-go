package sqlutil

import (
	"database/sql"
	"fmt"
	"reflect"
)

// A ScanRow is a container for SQL metadata for a single row.
// The row metadata is used to generate dataframe fields and a slice that can be used with sql.Scan
type ScanRow struct {
	Columns []string
	Types   []reflect.Type
}

// NewScanRow creates a new ScanRow with a length of `length`. Use the `Set` function to manually set elements at specific indices.
func NewScanRow(length int) *ScanRow {
	return &ScanRow{
		Columns: make([]string, length),
		Types:   make([]reflect.Type, length),
	}
}

// Append adds data to the end of the list of types and columns
func (s *ScanRow) Append(name string, colType reflect.Type) {
	s.Columns = append(s.Columns, name)
	s.Types = append(s.Types, colType)
}

// Set sets the internal data at i
func (s *ScanRow) Set(i int, name string, colType reflect.Type) {
	s.Columns[i] = name
	s.Types[i] = colType
}

// NewScannableRow creates a slice where each element is usable in a call to `(database/sql.Rows).Scan`
// aka a pointer
func (s *ScanRow) NewScannableRow() []interface{} {
	values := make([]interface{}, len(s.Types))

	for i, v := range s.Types {
		if v.Kind() == reflect.Ptr {
			n := reflect.New(v.Elem())
			values[i] = n.Interface()
		} else {
			values[i] = reflect.New(v).Interface()
		}
	}

	return values
}

// MakeScanRow creates a new scan row given the column types and names.
// Applicable converters will substitute the SQL scan type with the one provided by the converter.
// The list of returned converters is the same length as the SQL rows and corresponds with the rows at the same index. (e.g. value at slice element 3 corresponds with the converter at slice element 3)
// If no converter is provided for a row that has a type that does not fit into a dataframe, it is skipped.
func MakeScanRow(colTypes []*sql.ColumnType, colNames []string, converters ...Converter) (*ScanRow, []Converter, error) {
	// In the future we can probably remove this restriction. But right now we map names to Arrow Field Names.
	// Arrow Field names must be unique: https://github.com/grafana/grafana-plugin-sdk-go/issues/59
	seen := map[string]int{}
	for i, name := range colNames {
		if j, ok := seen[name]; ok {
			return nil, nil, fmt.Errorf(`duplicate column names are not allowed, found identical name "%v" at column indices %v and %v`, name, j, i)
		}
		seen[name] = i
	}

	r := NewScanRow(0)
	c := []Converter{}

	// For each column, define a concrete type in the list of values
	for i, colType := range colTypes {
		colName := colNames[i]
		nullable, ok := colType.Nullable()
		if !ok {
			nullable = true // If we don't know if it is nullable, assume it is
		}

		var converter *Converter
		scanType := colType.ScanType()
		for i, v := range converters {
			if v.InputTypeRegex != nil {
				if v.InputTypeRegex.MatchString(colType.DatabaseTypeName()) {
					scanType = v.InputScanType
					converter = &converters[i]
					break
				}
			}

			// If there's an applicable converter for this column, scan using the InputScanType.
			if v.InputTypeName == colType.DatabaseTypeName() {
				scanType = v.InputScanType
				converter = &converters[i]
				break
			}
		}

		if converter == nil {
			v := NewDefaultConverter(colType.Name(), nullable, scanType)
			converter = &v
			scanType = v.InputScanType
		}
		converter.colType = *colType

		r.Append(colName, scanType)
		c = append(c, *converter)
	}

	return r, c, nil
}
