package dataframe

import (
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

// Frame represents a columnar storage with optional labels.
type Frame struct {
	Name   string
	Fields []*Field

	RefID string
	Meta  *QueryResultMeta
}

// Field represents a column of data with a specific type.
type Field struct {
	Name   string
	Config *FieldConfig
	Vector Vector // TODO? in the frontend, the variable is called "Values"
	Labels Labels
}

// Fields is a slice of Field pointers.
type Fields []*Field

// AppendRow adds a new row to the Frame by appending to each value to
// the corresponding Field in the dataframe.
// The dataframe's Fields and the Fields' Vectors must be initalized or AppendRow will panic.
// The number of arguments must match the number of Fields in the Frame and each type must coorespond
// to the Field type or AppendRow will panic.
func (f *Frame) AppendRow(vals ...interface{}) {
	for i, v := range vals {
		f.Fields[i].Vector.Append(v)
	}
}

// AppendRowSafe adds a new row to the Frame by appending to each value to
// the corresponding Field in the dataframe.
// The dataframe's Fields and the Fields' Vectors must be initalized or AppendRow will error.
// The number of arguments must match the number of Fields in the Frame and each type must coorespond
// to the Field type or AppendRow will error.
func (f *Frame) AppendRowSafe(vals ...interface{}) error {
	if len(vals) != len(f.Fields) {
		return fmt.Errorf("failed to append vals to Frame. Frame has %v fields but was given %v to append", len(f.Fields), len(vals))
	}
	// check validity before any modification
	for i, v := range vals {
		if f.Fields[i] == nil {
			return fmt.Errorf("can not append to uninitalized Field at field index %v", i)
		}
		if f.Fields[i].Vector == nil {
			return fmt.Errorf("can not append to uninitalized Field Vector at field index %v", i)
		}
		dfPType := f.Fields[i].Vector.PrimitiveType()
		if v == nil {
			if dfPType.Nullable() {
				continue
			}
			return fmt.Errorf("can not append nil to non-nullable vector with underlying type %s at field index %v", dfPType, i)
		}

		if pTypeFromVal(v) != dfPType {
			return fmt.Errorf("invalid type appending row at index %v, got %T want %v", i, v, dfPType.ItemTypeString())
		}
	}
	// second loop that modifies
	f.AppendRow(vals...)
	return nil
}

func (f *Frame) AppendRows(rows ...[]interface{}) {
	// WIP
	for _, row := range rows {
		// Should probably increase capacity by len(rows) and then .Set?
		f.AppendRow(row...)
	}
}

// ScannableRow adds a row to the dataframe, and returns a slice of references
// that can be passed to rows.Scan() in the in sql package.
func (f *Frame) ScannableRow() []interface{} {
	row := make([]interface{}, len(f.Fields))
	for i, field := range f.Fields {
		vec := field.Vector
		vec.Extend(1)
		row[i] = vec.PointerAt(vec.Len() - 1)
	}
	return row
}

// NewField returns a new instance of Field.
func NewField(name string, labels Labels, values interface{}) *Field {
	var vec Vector
	switch v := values.(type) {
	case []int8:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []*int8:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []int16:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []*int16:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []int32:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []*int32:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []int64:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []*int64:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []uint8:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []*uint8:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []uint16:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []*uint16:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []uint32:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []*uint32:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []uint64:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []*uint64:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []float32:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []*float32:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []float64:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []*float64:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []string:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []*string:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []bool:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []*bool:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []time.Time:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	case []*time.Time:
		vec = newVector(v, len(v))
		for i := 0; i < len(v); i++ {
			vec.Set(i, v[i])
		}
	default:
		panic(fmt.Errorf("unsupported field type %T", v))
	}

	return &Field{
		Name:   name,
		Vector: vec,
		Labels: labels,
	}
}

// Len returns the number of elements in the field.
func (f *Field) Len() int {
	return f.Vector.Len()
}

// SetConfig modifies the Field's Config property to
// be set to conf and returns the Field.
func (f *Field) SetConfig(conf *FieldConfig) *Field {
	f.Config = conf
	return f
}

// Labels are used to add metadata to an object.
type Labels map[string]string

// Equals returns true if the argument has the same k=v pairs as the receiver.
func (l Labels) Equals(arg Labels) bool {
	if len(l) != len(arg) {
		return false
	}
	for k, v := range l {
		if argVal, ok := arg[k]; !ok || argVal != v {
			return false
		}
	}
	return true
}

// Contains returns true if all k=v pairs of the argument are in the receiver.
func (l Labels) Contains(arg Labels) bool {
	if len(arg) > len(l) {
		return false
	}
	for k, v := range arg {
		if argVal, ok := l[k]; !ok || argVal != v {
			return false
		}
	}
	return true
}

func (l Labels) String() string {
	// Better structure, should be sorted, copy prom probably
	keys := make([]string, len(l))
	i := 0
	for k := range l {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	var sb strings.Builder

	i = 0
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(l[k])
		if i != len(keys)-1 {
			sb.WriteString(", ")
		}
		i++
	}
	return sb.String()
}

// LabelsFromString parses the output of Labels.String() into
// a Labels object. It probably has some flaws.
func LabelsFromString(s string) (Labels, error) {
	if s == "" {
		return nil, nil
	}
	labels := make(map[string]string)

	for _, rawKV := range strings.Split(s, ", ") {
		kV := strings.SplitN(rawKV, "=", 2)
		if len(kV) != 2 {
			return nil, fmt.Errorf(`invalid label key=value pair "%v"`, rawKV)
		}
		labels[kV[0]] = kV[1]
	}

	return labels, nil
}

// New returns a new instance of a Frame.
func New(name string, fields ...*Field) *Frame {
	return &Frame{
		Name:   name,
		Fields: fields,
	}
}

// Rows returns the number of rows in the frame.
func (f *Frame) Rows() int {
	if len(f.Fields) > 0 {
		return f.Fields[0].Len()
	}
	return 0
}

// NewForSQLRows creates a new Frame approriate for scanning SQL rows with
// the the new Frame's ScannableRow() method.
func NewForSQLRows(rows *sql.Rows) (*Frame, error) {
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	colNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	frame := &Frame{}
	for i, colType := range colTypes {
		colName := colNames[i]
		nullable, ok := colType.Nullable()
		if !ok {
			return nil, fmt.Errorf("sql driver won't tell me if this is nullable....?")
		}
		scanType := colType.ScanType()
		if !nullable {
			vec := reflect.MakeSlice(reflect.SliceOf(scanType), 0, 0).Interface()
			frame.Fields = append(frame.Fields, NewField(colName, nil, vec))
			continue
		}
		ptrType := reflect.TypeOf(reflect.New(scanType).Interface())
		vec := reflect.MakeSlice(reflect.SliceOf(ptrType), 0, 0).Interface()
		frame.Fields = append(frame.Fields, NewField(colName, nil, vec))
	}
	return frame, nil
}
