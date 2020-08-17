package converters_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data/converters"
	"github.com/stretchr/testify/require"
)

func CheckStringConversions(t *testing.T) {
	val, err := converters.AnyToOptionalString.Converter(12.3)
	require.Nil(t, err)
	require.Equal(t, "12.3", val)

	val, err = converters.AnyToOptionalString.Converter(nil)
	require.Nil(t, err)
	require.Nil(t, val)
}

func CheckNumericConversions(t *testing.T) {
	val, err := converters.Float64ToOptionalFloat64.Converter(12.34)
	require.Nil(t, err)
	require.Equal(t, 12.34, val)
}
