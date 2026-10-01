package data

import (
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
