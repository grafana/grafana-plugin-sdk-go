package data

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFieldConfigClone(t *testing.T) {
	require.Nil(t, (*FieldConfig)(nil).clone())

	cfg := (&FieldConfig{
		Unit:              "short",
		DisplayNameFromDS: "orig",
		TypeConfig:        &FieldTypeConfig{Enum: &EnumFieldConfig{Text: []string{"INFO", "ERROR"}}},
	}).SetDecimals(2)

	clone := cfg.clone()
	require.Equal(t, cfg, clone)
	require.NotSame(t, cfg, clone)
	require.NotSame(t, cfg.TypeConfig, clone.TypeConfig)
	require.NotSame(t, cfg.TypeConfig.Enum, clone.TypeConfig.Enum)

	clone.DisplayNameFromDS = "other"
	clone.SetDecimals(5)
	clone.TypeConfig.Enum.Text[0] = "DEBUG"

	require.Equal(t, "orig", cfg.DisplayNameFromDS)
	require.EqualValues(t, 2, *cfg.Decimals)
	require.Equal(t, []string{"INFO", "ERROR"}, cfg.TypeConfig.Enum.Text)
}

// TestFieldConfigCloneSharesNothing fills every exported field, so a new pointer, map or slice field fails here until clone handles it.
func TestFieldConfigCloneSharesNothing(t *testing.T) {
	cfg := &FieldConfig{}
	fillValue(reflect.ValueOf(cfg).Elem(), 0)
	from, to := ConfFloat64(1), ConfFloat64(2)
	cfg.Mappings = ValueMappings{
		ValueMapper{"a": {Text: "A"}},
		&ValueMapper{"b": {Text: "B"}},
		RangeValueMapper{From: &from, To: &to},
		&RangeValueMapper{From: &from, To: &to},
		SpecialValueMapper{Match: SpecialValueNull},
		&SpecialValueMapper{Match: SpecialValueNaN},
	}

	clone := cfg.clone()
	require.Equal(t, cfg, clone)
	requireNoSharedRefs(t, "FieldConfig", reflect.ValueOf(cfg).Elem(), reflect.ValueOf(clone).Elem())
}

func fillValue(v reflect.Value, depth int) {
	if depth > 10 {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		fillValue(v.Elem(), depth+1)
	case reflect.Struct:
		for _, f := range v.Fields() {
			if f.CanSet() {
				fillValue(f, depth+1)
			}
		}
	case reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		fillValue(v.Index(0), depth+1)
	case reflect.Map:
		k, e := reflect.New(v.Type().Key()).Elem(), reflect.New(v.Type().Elem()).Elem()
		fillValue(k, depth+1)
		fillValue(e, depth+1)
		m := reflect.MakeMap(v.Type())
		m.SetMapIndex(k, e)
		v.Set(m)
	case reflect.Interface:
		if v.NumMethod() == 0 {
			v.Set(reflect.ValueOf(map[string]any{"k": []any{map[string]any{"n": 1.0}}}))
		}
	case reflect.String:
		v.SetString("x")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(1)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(1.5)
	}
}

func requireNoSharedRefs(t *testing.T, path string, a, b reflect.Value) {
	t.Helper()
	switch a.Kind() {
	case reflect.Pointer:
		if a.IsNil() {
			return
		}
		require.NotEqual(t, a.Pointer(), b.Pointer(), "%s is shared with the clone", path)
		requireNoSharedRefs(t, path, a.Elem(), b.Elem())
	case reflect.Map:
		if a.Len() == 0 {
			return
		}
		require.NotEqual(t, a.Pointer(), b.Pointer(), "%s is shared with the clone", path)
		iter := a.MapRange()
		for iter.Next() {
			requireNoSharedRefs(t, fmt.Sprintf("%s[%v]", path, iter.Key()), iter.Value(), b.MapIndex(iter.Key()))
		}
	case reflect.Slice:
		if a.Len() == 0 {
			return
		}
		require.NotEqual(t, a.Pointer(), b.Pointer(), "%s is shared with the clone", path)
		for i := range a.Len() {
			requireNoSharedRefs(t, fmt.Sprintf("%s[%d]", path, i), a.Index(i), b.Index(i))
		}
	case reflect.Interface:
		if !a.IsNil() {
			requireNoSharedRefs(t, path, a.Elem(), b.Elem())
		}
	case reflect.Struct:
		for f, af := range a.Fields() {
			if f.IsExported() {
				requireNoSharedRefs(t, path+"."+f.Name, af, b.FieldByIndex(f.Index))
			}
		}
	}
}
