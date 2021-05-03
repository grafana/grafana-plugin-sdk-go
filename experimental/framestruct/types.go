package framestruct

import (
	"fmt"
	"reflect"
	"time"
)

func sliceFor(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case int8:
		return []int8{}, nil
	case *int8:
		return []*int8{}, nil
	case int16:
		return []int16{}, nil
	case *int16:
		return []*int16{}, nil
	case int32:
		return []int32{}, nil
	case *int32:
		return []*int32{}, nil
	case int64:
		return []int64{}, nil
	case *int64:
		return []*int64{}, nil
	case uint8:
		return []uint8{}, nil
	case *uint8:
		return []*uint8{}, nil
	case uint16:
		return []uint16{}, nil
	case *uint16:
		return []*uint16{}, nil
	case uint32:
		return []uint32{}, nil
	case *uint32:
		return []*uint32{}, nil
	case uint64:
		return []uint64{}, nil
	case *uint64:
		return []*uint64{}, nil
	case float32:
		return []float32{}, nil
	case *float32:
		return []*float32{}, nil
	case float64:
		return []float64{}, nil
	case *float64:
		return []*float64{}, nil
	case string:
		return []string{}, nil
	case *string:
		return []*string{}, nil
	case bool:
		return []bool{}, nil
	case *bool:
		return []*bool{}, nil
	case time.Time:
		return []time.Time{}, nil
	case *time.Time:
		return []*time.Time{}, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}

func supportedToplevelType(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			s := v.Index(i)
			return supportedToplevelType(s)
		}
		return true
	case reflect.Struct:
		_, ok := v.Interface().(time.Time)
		if ok {
			return false //times are structs, but not toplevel ones
		}
		return true
	default:
		return v.Kind() == reflect.Map
	}
}
