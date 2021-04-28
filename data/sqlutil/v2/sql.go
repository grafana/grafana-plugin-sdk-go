package sqlutil

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func FrameFromRows(rows *sql.Rows, rowLimit int64, converters ...Converter) (*data.Frame, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	names, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	scanner, converters, err := MakeScanRow(types, names, converters...)
	if err != nil {
		return nil, err
	}

	log.Println(types, names)

	frame := NewFrame(converters...)

	for rows.Next() {
		r := scanner.NewScannableRow()
		if err := rows.Scan(r...); err != nil {
			return nil, err
		}

		if err := Append(frame, r, converters...); err != nil {
			return nil, err
		}
	}

	return frame, nil
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
		for _, v := range converters {
			// If there's an applicable converter for this column, scan using the InputScanType.
			if v.InputTypeName == colType.DatabaseTypeName() {
				scanType = v.InputScanType
				converter = &v
				break
			}
		}

		if converter == nil {
			v, err := NewDefaultFrameConverter(scanType)
			// TODO: get this error into the frame meta
			// TODO: handle the case that this may be the only row to scan
			if err != nil {
				log.Println("Skipping column", colName, "with type", scanType)
				continue
			}
			converter = &Converter{
				Name:           fmt.Sprintf("Default converter for %s", colName),
				InputScanType:  scanType,
				InputTypeName:  colName,
				FrameConverter: v,
			}
		}

		var val interface{}
		if !nullable {
			val = reflect.New(scanType).Interface()
		} else {
			ptrType := reflect.TypeOf(reflect.New(scanType).Interface())
			// Nullabe types get passed to scan as a pointer to a pointer
			val = reflect.New(ptrType).Interface()
		}

		// if !data.ValidFieldType(vec) {
		// 	ptrType := reflect.TypeOf(reflect.New(reflect.TypeOf("")).Interface())
		// 	vec = reflect.MakeSlice(reflect.SliceOf(ptrType), 0, 0).Interface()
		// }

		r.Append(val, colName, scanType)
		c = append(c, *converter)
	}

	return r, c, nil
}
