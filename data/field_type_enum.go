package data

import (
	jsoniter "github.com/json-iterator/go"
)

// EnumItemIndex is used to represent enum values as uint16 indices
type EnumItemIndex uint16

// JSON helpers for enum vectors
func readEnumVectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[EnumItemIndex], error) {
	arr := newGenericVector[EnumItemIndex](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readEnumVectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadUint16()
			arr.SetTyped(i, EnumItemIndex(v))
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func readNullableEnumVectorJSON(iter *jsoniter.Iterator, size int) (*nullableGenericVector[EnumItemIndex], error) {
	arr := newNullableGenericVector[EnumItemIndex](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullableEnumVectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
			arr.SetTyped(i, nil)
		} else {
			v := iter.ReadUint16()
			eII := EnumItemIndex(v)
			arr.SetTyped(i, &eII)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullableEnumVectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}
