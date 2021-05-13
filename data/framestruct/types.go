package framestruct

import (
	"fmt"
	"reflect"
	"time"
)

func sliceFor(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case int8:
		return []*int8{}, nil
	case *int8:
		return []*int8{}, nil
	case int16:
		return []*int16{}, nil
	case *int16:
		return []*int16{}, nil
	case int32:
		return []*int32{}, nil
	case *int32:
		return []*int32{}, nil
	case int64:
		return []*int64{}, nil
	case *int64:
		return []*int64{}, nil
	case uint8:
		return []*uint8{}, nil
	case *uint8:
		return []*uint8{}, nil
	case uint16:
		return []*uint16{}, nil
	case *uint16:
		return []*uint16{}, nil
	case uint32:
		return []*uint32{}, nil
	case *uint32:
		return []*uint32{}, nil
	case uint64:
		return []*uint64{}, nil
	case *uint64:
		return []*uint64{}, nil
	case float32:
		return []*float32{}, nil
	case *float32:
		return []*float32{}, nil
	case float64:
		return []*float64{}, nil
	case *float64:
		return []*float64{}, nil
	case string:
		return []*string{}, nil
	case *string:
		return []*string{}, nil
	case bool:
		return []*bool{}, nil
	case *bool:
		return []*bool{}, nil
	case time.Time:
		return []*time.Time{}, nil
	case *time.Time:
		return []*time.Time{}, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}

func toPointer(value interface{}) (interface{}, error) {
	switch t := value.(type) {
	case int8:
		v := value.(int8)
		return &v, nil
	case *int8:
		return value, nil
	case int16:
		v := value.(int16)
		return &v, nil
	case *int16:
		return value, nil
	case int32:
		v := value.(int32)
		return &v, nil
	case *int32:
		return value, nil
	case int64:
		v := value.(int64)
		return &v, nil
	case *int64:
		return value, nil
	case uint8:
		v := value.(uint8)
		return &v, nil
	case *uint8:
		return value, nil
	case uint16:
		v := value.(uint16)
		return &v, nil
	case *uint16:
		return value, nil
	case uint32:
		v := value.(uint32)
		return &v, nil
	case *uint32:
		return value, nil
	case uint64:
		v := value.(uint64)
		return &v, nil
	case *uint64:
		return value, nil
	case float32:
		v := value.(float32)
		return &v, nil
	case *float32:
		return value, nil
	case float64:
		v := value.(float64)
		return &v, nil
	case *float64:
		return value, nil
	case string:
		v := value.(string)
		return &v, nil
	case *string:
		return value, nil
	case bool:
		v := value.(bool)
		return &v, nil
	case *bool:
		return value, nil
	case time.Time:
		v := value.(time.Time)
		return &v, nil
	case *time.Time:
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", t)
	}
}

func supportedToplevelType(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice:
		if v.Len() > 0 {
			return supportedToplevelType(v.Index(0))
		}
		return true
	case reflect.Struct:
		_, ok := v.Interface().(time.Time)
		if ok {
			return false // times are structs, but not toplevel ones
		}
		return true
	default:
		return v.Kind() == reflect.Map
	}
}
