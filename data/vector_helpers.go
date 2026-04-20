package data

import (
	"encoding/json"
	"fmt"
	"time"
)

// appendTypedToVector is an internal helper that appends a value to a vector
// using typed methods instead of interface{} boxing. This eliminates allocation
// overhead for internal operations.
// nolint:gocyclo
func appendTypedToVector(vec vector, val interface{}) {
	if val == nil {
		vec.Append(nil)
		return
	}

	switch v := vec.(type) {
	// int types
	case *genericVector[int8]:
		v.AppendTyped(val.(int8))
	case *nullableGenericVector[int8]:
		v.AppendTyped(val.(*int8))
	case *genericVector[int16]:
		v.AppendTyped(val.(int16))
	case *nullableGenericVector[int16]:
		v.AppendTyped(val.(*int16))
	case *genericVector[int32]:
		v.AppendTyped(val.(int32))
	case *nullableGenericVector[int32]:
		v.AppendTyped(val.(*int32))
	case *genericVector[int64]:
		v.AppendTyped(val.(int64))
	case *nullableGenericVector[int64]:
		v.AppendTyped(val.(*int64))

	// uint types
	case *genericVector[uint8]:
		v.AppendTyped(val.(uint8))
	case *nullableGenericVector[uint8]:
		v.AppendTyped(val.(*uint8))
	case *genericVector[uint16]:
		v.AppendTyped(val.(uint16))
	case *nullableGenericVector[uint16]:
		v.AppendTyped(val.(*uint16))
	case *genericVector[uint32]:
		v.AppendTyped(val.(uint32))
	case *nullableGenericVector[uint32]:
		v.AppendTyped(val.(*uint32))
	case *genericVector[uint64]:
		v.AppendTyped(val.(uint64))
	case *nullableGenericVector[uint64]:
		v.AppendTyped(val.(*uint64))

	// float types
	case *genericVector[float32]:
		v.AppendTyped(val.(float32))
	case *nullableGenericVector[float32]:
		v.AppendTyped(val.(*float32))
	case *genericVector[float64]:
		v.AppendTyped(val.(float64))
	case *nullableGenericVector[float64]:
		v.AppendTyped(val.(*float64))

	// string, bool
	case *genericVector[string]:
		v.AppendTyped(val.(string))
	case *nullableGenericVector[string]:
		v.AppendTyped(val.(*string))
	case *genericVector[bool]:
		v.AppendTyped(val.(bool))
	case *nullableGenericVector[bool]:
		v.AppendTyped(val.(*bool))

	// time, json, enum
	case *genericVector[time.Time]:
		v.AppendTyped(val.(time.Time))
	case *nullableGenericVector[time.Time]:
		v.AppendTyped(val.(*time.Time))
	case *genericVector[json.RawMessage]:
		v.AppendTyped(val.(json.RawMessage))
	case *nullableGenericVector[json.RawMessage]:
		v.AppendTyped(val.(*json.RawMessage))
	case *genericVector[EnumItemIndex]:
		enumVal, ok := enumValueFromInterface(val)
		if !ok {
			enumVal = 0
		}
		v.AppendTyped(enumVal)
	case *nullableGenericVector[EnumItemIndex]:
		enumPtr, ok := enumPointerFromInterface(val)
		if !ok {
			v.AppendTyped(nil)
			return
		}
		v.AppendTyped(enumPtr)

	default:
		panic(fmt.Sprintf("unsupported vector type: %T", vec))
	}
}

// setTypedInVector is an internal helper that sets a value in a vector
// using typed methods instead of interface{} boxing.
// nolint:gocyclo
func setTypedInVector(vec vector, idx int, val interface{}) {
	if val == nil {
		vec.Set(idx, nil)
		return
	}

	switch v := vec.(type) {
	// int types
	case *genericVector[int8]:
		v.SetTyped(idx, val.(int8))
	case *nullableGenericVector[int8]:
		v.SetTyped(idx, val.(*int8))
	case *genericVector[int16]:
		v.SetTyped(idx, val.(int16))
	case *nullableGenericVector[int16]:
		v.SetTyped(idx, val.(*int16))
	case *genericVector[int32]:
		v.SetTyped(idx, val.(int32))
	case *nullableGenericVector[int32]:
		v.SetTyped(idx, val.(*int32))
	case *genericVector[int64]:
		v.SetTyped(idx, val.(int64))
	case *nullableGenericVector[int64]:
		v.SetTyped(idx, val.(*int64))

	// uint types
	case *genericVector[uint8]:
		v.SetTyped(idx, val.(uint8))
	case *nullableGenericVector[uint8]:
		v.SetTyped(idx, val.(*uint8))
	case *genericVector[uint16]:
		v.SetTyped(idx, val.(uint16))
	case *nullableGenericVector[uint16]:
		v.SetTyped(idx, val.(*uint16))
	case *genericVector[uint32]:
		v.SetTyped(idx, val.(uint32))
	case *nullableGenericVector[uint32]:
		v.SetTyped(idx, val.(*uint32))
	case *genericVector[uint64]:
		v.SetTyped(idx, val.(uint64))
	case *nullableGenericVector[uint64]:
		v.SetTyped(idx, val.(*uint64))

	// float types
	case *genericVector[float32]:
		v.SetTyped(idx, val.(float32))
	case *nullableGenericVector[float32]:
		v.SetTyped(idx, val.(*float32))
	case *genericVector[float64]:
		v.SetTyped(idx, val.(float64))
	case *nullableGenericVector[float64]:
		v.SetTyped(idx, val.(*float64))

	// string, bool
	case *genericVector[string]:
		v.SetTyped(idx, val.(string))
	case *nullableGenericVector[string]:
		v.SetTyped(idx, val.(*string))
	case *genericVector[bool]:
		v.SetTyped(idx, val.(bool))
	case *nullableGenericVector[bool]:
		v.SetTyped(idx, val.(*bool))

	// time, json, enum
	case *genericVector[time.Time]:
		v.SetTyped(idx, val.(time.Time))
	case *nullableGenericVector[time.Time]:
		v.SetTyped(idx, val.(*time.Time))
	case *genericVector[json.RawMessage]:
		v.SetTyped(idx, val.(json.RawMessage))
	case *nullableGenericVector[json.RawMessage]:
		v.SetTyped(idx, val.(*json.RawMessage))
	case *genericVector[EnumItemIndex]:
		enumVal, ok := enumValueFromInterface(val)
		if !ok {
			enumVal = 0
		}
		v.SetTyped(idx, enumVal)
	case *nullableGenericVector[EnumItemIndex]:
		enumPtr, ok := enumPointerFromInterface(val)
		if !ok {
			v.Set(idx, nil)
			return
		}
		v.SetTyped(idx, enumPtr)

	default:
		panic(fmt.Sprintf("unsupported vector type: %T", vec))
	}
}

// setConcreteTypedInVector is an internal helper that sets a concrete (non-pointer) value
// in a vector using typed methods. For nullable vectors, it converts the value to a pointer.
// nolint:gocyclo
func setConcreteTypedInVector(vec vector, idx int, val interface{}) {
	switch v := vec.(type) {
	// int types
	case *genericVector[int8]:
		v.SetTyped(idx, val.(int8))
	case *nullableGenericVector[int8]:
		concrete := val.(int8)
		v.SetConcreteTyped(idx, concrete)
	case *genericVector[int16]:
		v.SetTyped(idx, val.(int16))
	case *nullableGenericVector[int16]:
		concrete := val.(int16)
		v.SetConcreteTyped(idx, concrete)
	case *genericVector[int32]:
		v.SetTyped(idx, val.(int32))
	case *nullableGenericVector[int32]:
		concrete := val.(int32)
		v.SetConcreteTyped(idx, concrete)
	case *genericVector[int64]:
		v.SetTyped(idx, val.(int64))
	case *nullableGenericVector[int64]:
		concrete := val.(int64)
		v.SetConcreteTyped(idx, concrete)

	// uint types
	case *genericVector[uint8]:
		v.SetTyped(idx, val.(uint8))
	case *nullableGenericVector[uint8]:
		concrete := val.(uint8)
		v.SetConcreteTyped(idx, concrete)
	case *genericVector[uint16]:
		v.SetTyped(idx, val.(uint16))
	case *nullableGenericVector[uint16]:
		concrete := val.(uint16)
		v.SetConcreteTyped(idx, concrete)
	case *genericVector[uint32]:
		v.SetTyped(idx, val.(uint32))
	case *nullableGenericVector[uint32]:
		concrete := val.(uint32)
		v.SetConcreteTyped(idx, concrete)
	case *genericVector[uint64]:
		v.SetTyped(idx, val.(uint64))
	case *nullableGenericVector[uint64]:
		concrete := val.(uint64)
		v.SetConcreteTyped(idx, concrete)

	// float types
	case *genericVector[float32]:
		v.SetTyped(idx, val.(float32))
	case *nullableGenericVector[float32]:
		concrete := val.(float32)
		v.SetConcreteTyped(idx, concrete)
	case *genericVector[float64]:
		v.SetTyped(idx, val.(float64))
	case *nullableGenericVector[float64]:
		concrete := val.(float64)
		v.SetConcreteTyped(idx, concrete)

	// string, bool
	case *genericVector[string]:
		v.SetTyped(idx, val.(string))
	case *nullableGenericVector[string]:
		concrete := val.(string)
		v.SetConcreteTyped(idx, concrete)
	case *genericVector[bool]:
		v.SetTyped(idx, val.(bool))
	case *nullableGenericVector[bool]:
		concrete := val.(bool)
		v.SetConcreteTyped(idx, concrete)

	// time, json, enum
	case *genericVector[time.Time]:
		v.SetTyped(idx, val.(time.Time))
	case *nullableGenericVector[time.Time]:
		concrete := val.(time.Time)
		v.SetConcreteTyped(idx, concrete)
	case *genericVector[json.RawMessage]:
		v.SetTyped(idx, val.(json.RawMessage))
	case *nullableGenericVector[json.RawMessage]:
		concrete := val.(json.RawMessage)
		v.SetConcreteTyped(idx, concrete)
	case *genericVector[EnumItemIndex]:
		enumVal, ok := enumValueFromInterface(val)
		if !ok {
			enumVal = 0
		}
		v.SetTyped(idx, enumVal)
	case *nullableGenericVector[EnumItemIndex]:
		enumVal, ok := enumValueFromInterface(val)
		if !ok {
			v.SetTyped(idx, nil)
			return
		}
		v.SetConcreteTyped(idx, enumVal)

	default:
		panic(fmt.Sprintf("unsupported vector type: %T", vec))
	}
}

// insertTypedInVector is an internal helper that inserts a value in a vector
// using typed methods instead of interface{} boxing.
// nolint:gocyclo
func insertTypedInVector(vec vector, idx int, val interface{}) {
	if val == nil {
		vec.Insert(idx, nil)
		return
	}

	switch v := vec.(type) {
	// int types
	case *genericVector[int8]:
		v.InsertTyped(idx, val.(int8))
	case *nullableGenericVector[int8]:
		v.InsertTyped(idx, val.(*int8))
	case *genericVector[int16]:
		v.InsertTyped(idx, val.(int16))
	case *nullableGenericVector[int16]:
		v.InsertTyped(idx, val.(*int16))
	case *genericVector[int32]:
		v.InsertTyped(idx, val.(int32))
	case *nullableGenericVector[int32]:
		v.InsertTyped(idx, val.(*int32))
	case *genericVector[int64]:
		v.InsertTyped(idx, val.(int64))
	case *nullableGenericVector[int64]:
		v.InsertTyped(idx, val.(*int64))

	// uint types
	case *genericVector[uint8]:
		v.InsertTyped(idx, val.(uint8))
	case *nullableGenericVector[uint8]:
		v.InsertTyped(idx, val.(*uint8))
	case *genericVector[uint16]:
		v.InsertTyped(idx, val.(uint16))
	case *nullableGenericVector[uint16]:
		v.InsertTyped(idx, val.(*uint16))
	case *genericVector[uint32]:
		v.InsertTyped(idx, val.(uint32))
	case *nullableGenericVector[uint32]:
		v.InsertTyped(idx, val.(*uint32))
	case *genericVector[uint64]:
		v.InsertTyped(idx, val.(uint64))
	case *nullableGenericVector[uint64]:
		v.InsertTyped(idx, val.(*uint64))

	// float types
	case *genericVector[float32]:
		v.InsertTyped(idx, val.(float32))
	case *nullableGenericVector[float32]:
		v.InsertTyped(idx, val.(*float32))
	case *genericVector[float64]:
		v.InsertTyped(idx, val.(float64))
	case *nullableGenericVector[float64]:
		v.InsertTyped(idx, val.(*float64))

	// string, bool
	case *genericVector[string]:
		v.InsertTyped(idx, val.(string))
	case *nullableGenericVector[string]:
		v.InsertTyped(idx, val.(*string))
	case *genericVector[bool]:
		v.InsertTyped(idx, val.(bool))
	case *nullableGenericVector[bool]:
		v.InsertTyped(idx, val.(*bool))

	// time, json, enum
	case *genericVector[time.Time]:
		v.InsertTyped(idx, val.(time.Time))
	case *nullableGenericVector[time.Time]:
		v.InsertTyped(idx, val.(*time.Time))
	case *genericVector[json.RawMessage]:
		v.InsertTyped(idx, val.(json.RawMessage))
	case *nullableGenericVector[json.RawMessage]:
		v.InsertTyped(idx, val.(*json.RawMessage))
	case *genericVector[EnumItemIndex]:
		enumVal, ok := enumValueFromInterface(val)
		if !ok {
			enumVal = 0
		}
		v.InsertTyped(idx, enumVal)
	case *nullableGenericVector[EnumItemIndex]:
		enumPtr, ok := enumPointerFromInterface(val)
		if !ok {
			v.Insert(idx, nil)
			return
		}
		v.InsertTyped(idx, enumPtr)

	default:
		panic(fmt.Sprintf("unsupported vector type: %T", vec))
	}
}

func enumValueFromInterface(val interface{}) (EnumItemIndex, bool) {
	switch v := val.(type) {
	case EnumItemIndex:
		return v, true
	case uint16:
		return EnumItemIndex(v), true
	case *EnumItemIndex:
		if v == nil {
			return 0, false
		}
		return *v, true
	case *uint16:
		if v == nil {
			return 0, false
		}
		return EnumItemIndex(*v), true
	default:
		return 0, false
	}
}

func enumPointerFromInterface(val interface{}) (*EnumItemIndex, bool) {
	switch v := val.(type) {
	case *EnumItemIndex:
		return v, true
	case EnumItemIndex:
		cp := v
		return &cp, true
	case uint16:
		cp := EnumItemIndex(v)
		return &cp, true
	case *uint16:
		if v == nil {
			return nil, false
		}
		cp := EnumItemIndex(*v)
		return &cp, true
	default:
		return nil, false
	}
}
