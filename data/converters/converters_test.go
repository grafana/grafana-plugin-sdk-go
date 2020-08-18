package converters_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data/converters"
	"github.com/stretchr/testify/require"
)

func TestStringConversions(t *testing.T) {
	val, err := converters.AnyToNullableString.Converter(12.3)
	require.NoError(t, err)
	require.Equal(t, "12.3", *(val.(*string)))

	val, err = converters.AnyToNullableString.Converter(nil)
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestNumericConversions(t *testing.T) {
	val, err := converters.Float64ToNullableFloat64.Converter(12.34)
	require.NoError(t, err)
	require.Equal(t, 12.34, *(val.(*float64)))
}
