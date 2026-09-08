package data

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFieldConfigClone(t *testing.T) {
	require.Nil(t, (*FieldConfig)(nil).clone())

	cfg := (&FieldConfig{
		Unit:     "short",
		Mappings: ValueMappings{ValueMapper{"1": ValueMappingResult{Text: "one"}}},
		Thresholds: &ThresholdsConfig{
			Mode:  ThresholdsModeAbsolute,
			Steps: []Threshold{{Value: 0, Color: "green"}},
		},
		Links:      []DataLink{{Title: "link"}},
		Color:      map[string]interface{}{"mode": "palette-classic"},
		Custom:     map[string]interface{}{"drawStyle": "line"},
		TypeConfig: &FieldTypeConfig{Enum: &EnumFieldConfig{Text: []string{"INFO", "ERROR"}}},
	}).SetDecimals(2).SetMin(0).SetMax(10).SetFilterable(true)

	clone := cfg.clone()
	require.Equal(t, cfg, clone)

	require.NotSame(t, cfg.Decimals, clone.Decimals)
	require.NotSame(t, cfg.Min, clone.Min)
	require.NotSame(t, cfg.Max, clone.Max)
	require.NotSame(t, cfg.Filterable, clone.Filterable)
	require.NotSame(t, cfg.Thresholds, clone.Thresholds)
	require.NotSame(t, cfg.TypeConfig, clone.TypeConfig)
	require.NotSame(t, cfg.TypeConfig.Enum, clone.TypeConfig.Enum)

	*clone.Decimals = 5
	clone.Thresholds.Steps[0].Color = "red"
	clone.Links[0].Title = "other"
	clone.Color["mode"] = "fixed"
	clone.Custom["drawStyle"] = "bars"
	clone.TypeConfig.Enum.Text[0] = "DEBUG"

	require.EqualValues(t, 2, *cfg.Decimals)
	require.Equal(t, "green", cfg.Thresholds.Steps[0].Color)
	require.Equal(t, "link", cfg.Links[0].Title)
	require.Equal(t, "palette-classic", cfg.Color["mode"])
	require.Equal(t, "line", cfg.Custom["drawStyle"])
	require.Equal(t, []string{"INFO", "ERROR"}, cfg.TypeConfig.Enum.Text)
}
