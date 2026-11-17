// Package jsoniter wraps json-iterator/go's Iterator methods with error returns
// so linting can catch unchecked errors.
// The underlying iterator's Error property is returned and not reset.
// When the reader given to Parse fails, the returned error also wraps the reader's error.
// See json-iterator/go for method documentation and additional methods that
// can be added to this library.
package jsoniter

import (
	"errors"
	"io"

	j "github.com/json-iterator/go"
)

const (
	InvalidValue = j.InvalidValue
	StringValue  = j.StringValue
	NumberValue  = j.NumberValue
	NilValue     = j.NilValue
	BoolValue    = j.BoolValue
	ArrayValue   = j.ArrayValue
	ObjectValue  = j.ObjectValue
)

var (
	ConfigDefault                       = j.ConfigDefault
	ConfigCompatibleWithStandardLibrary = j.ConfigCompatibleWithStandardLibrary
)

type Stream = j.Stream
type ValEncoder = j.ValEncoder
type ValDecoder = j.ValDecoder

type Iterator struct {
	// named property instead of embedded so there is no
	// confusion about which method or property is called
	i *j.Iterator
	r *errorReader
}

func NewIterator(i *j.Iterator) *Iterator {
	return &Iterator{i: i}
}

// err returns the iterator's error. json-iterator rewrites errors as text inside struct, slice
// and array decoders, so the reader's error is joined back for errors.Is.
func (iter *Iterator) err() error {
	err := iter.i.Error
	if err == nil || iter.r == nil || iter.r.err == nil || errors.Is(err, iter.r.err) {
		return err
	}
	return &readError{err: err, read: iter.r.err}
}

func (iter *Iterator) ReadError() error {
	return iter.err()
}

func (iter *Iterator) SetError(err error) {
	iter.i.Error = err
}

func (iter *Iterator) Read() (any, error) {
	return iter.i.Read(), iter.err()
}

func (iter *Iterator) ReadAny() (j.Any, error) {
	return iter.i.ReadAny(), iter.err()
}

func (iter *Iterator) ReadArray() (bool, error) {
	return iter.i.ReadArray(), iter.err()
}

func (iter *Iterator) ReadObject() (string, error) {
	return iter.i.ReadObject(), iter.err()
}

func (iter *Iterator) CanReadArray() bool {
	ok, err := iter.ReadArray()
	return ok && err == nil
}

func (iter *Iterator) ReadString() (string, error) {
	return iter.i.ReadString(), iter.err()
}

func (iter *Iterator) ReadStringAsSlice() ([]byte, error) {
	return iter.i.ReadStringAsSlice(), iter.err()
}

func (iter *Iterator) WhatIsNext() (j.ValueType, error) {
	return iter.i.WhatIsNext(), iter.err()
}

func (iter *Iterator) Skip() error {
	iter.i.Skip()
	return iter.err()
}

func (iter *Iterator) SkipAndReturnBytes() ([]byte, error) {
	return iter.i.SkipAndReturnBytes(), iter.err()
}

func (iter *Iterator) ReadVal(obj any) error {
	iter.i.ReadVal(obj)
	return iter.err()
}

func (iter *Iterator) ReadFloat32() (float32, error) {
	return iter.i.ReadFloat32(), iter.err()
}

func (iter *Iterator) ReadFloat64() (float64, error) {
	return iter.i.ReadFloat64(), iter.err()
}

func (iter *Iterator) ReadInt() (int, error) {
	return iter.i.ReadInt(), iter.err()
}

func (iter *Iterator) ReadInt8() (int8, error) {
	return iter.i.ReadInt8(), iter.err()
}

func (iter *Iterator) ReadInt16() (int16, error) {
	return iter.i.ReadInt16(), iter.err()
}

func (iter *Iterator) ReadInt32() (int32, error) {
	return iter.i.ReadInt32(), iter.err()
}

func (iter *Iterator) ReadInt64() (int64, error) {
	return iter.i.ReadInt64(), iter.err()
}

func (iter *Iterator) ReadUint8() (uint8, error) {
	return iter.i.ReadUint8(), iter.err()
}

func (iter *Iterator) ReadUint16() (uint16, error) {
	return iter.i.ReadUint16(), iter.err()
}

func (iter *Iterator) ReadUint32() (uint32, error) {
	return iter.i.ReadUint32(), iter.err()
}

func (iter *Iterator) ReadUint64() (uint64, error) {
	return iter.i.ReadUint64(), iter.err()
}

func (iter *Iterator) ReadUint64Pointer() (*uint64, error) {
	u := iter.i.ReadUint64()
	if err := iter.err(); err != nil {
		return nil, err
	}
	return &u, nil
}

func (iter *Iterator) ReadNil() (bool, error) {
	return iter.i.ReadNil(), iter.err()
}

func (iter *Iterator) ReadBool() (bool, error) {
	return iter.i.ReadBool(), iter.err()
}

func (iter *Iterator) ReportError(op, msg string) error {
	iter.i.ReportError(op, msg)
	return iter.err()
}

func (iter *Iterator) Marshal(v any) ([]byte, error) {
	return ConfigDefault.Marshal(v)
}

func (iter *Iterator) Unmarshal(data []byte, v any) error {
	return ConfigDefault.Unmarshal(data, v)
}

func Parse(cfg j.API, reader io.Reader, bufSize int) (*Iterator, error) {
	r := &errorReader{r: reader}
	iter := &Iterator{i: j.Parse(cfg, r, bufSize), r: r}
	r.iter = iter.i
	return iter, iter.err()
}

func ParseBytes(cfg j.API, input []byte) (*Iterator, error) {
	iter := &Iterator{i: j.ParseBytes(cfg, input)}
	return iter, iter.err()
}

func ParseString(cfg j.API, input string) (*Iterator, error) {
	iter := &Iterator{i: j.ParseString(cfg, input)}
	return iter, iter.err()
}

func RegisterTypeEncoder(typ string, encoder ValEncoder) {
	j.RegisterTypeEncoder(typ, encoder)
}

func RegisterTypeDecoder(typ string, decoder ValDecoder) {
	j.RegisterTypeDecoder(typ, decoder)
}

// errorReader records the first error from the underlying reader, if the iterator had no error before it.
type errorReader struct {
	r    io.Reader
	iter *j.Iterator
	err  error
}

func (e *errorReader) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if err != nil && !errors.Is(err, io.EOF) && e.err == nil && (e.iter == nil || e.iter.Error == nil) {
		e.err = err
	}
	return n, err
}

// readError keeps the message json-iterator built and unwraps to both it and the reader's error.
type readError struct {
	err  error
	read error
}

func (e *readError) Error() string { return e.err.Error() }

func (e *readError) Unwrap() []error { return []error{e.err, e.read} }
