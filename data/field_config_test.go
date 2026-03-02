package data_test

import (
	"encoding/json"
	"math"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestFieldCOnfig(t *testing.T) {
	t.Run("field mappings", func(t *testing.T) {
		jsonText := `{
			"description": "turn on/off system. write 1 to turn on the system and write 0 to turn off the system",
			"writeable": true,
			"thresholds": {
				"mode": "absolute",
				"steps": [
					{ "value": null, "color": "red" },
					{ "value": 1, "color": "green" },
					{ "value": 5, "color": "blue" }
				]
			},
			"mappings": [
				{
					"type": "value",
					"options": {
					"0": {
						"text": "OFF",
						"color": "rgba(56, 56, 56, 1)"
					},
					"1": {
						"text": "ON",
						"color": "dark-green",
						"index": 1
					}
					}
				},
				{
					"type": "range",
					"options": {
					"from": 0,
					"to": 100,
					"result": {
						"text": "0-100",
						"color": "yellow",
						"index": 2
					}
					}
				},
				{
					"type": "range",
					"options": {
					"from": 25,
					"result": {
						"text": "25-Inf",
						"color": "yellow",
						"index": 3
					}
					}
				},
				{
					"type": "special",
					"options": {
					"match": "nan",
					"result": {
						"text": "Batman!",
						"color": "dark-red",
						"index": 4
					}
					}
				}
			]
		}`

		cfg := data.FieldConfig{}
		err := json.Unmarshal([]byte(jsonText), &cfg)
		require.NoError(t, err, "error parsing json")

		require.True(t, *cfg.Writeable)
		require.Len(t, cfg.Mappings, 4)
		require.Len(t, cfg.Thresholds.Steps, 3)
		require.Equal(t, data.Threshold{
			Value: data.ConfFloat64(math.Inf(-1)),
			Color: "red",
		}, cfg.Thresholds.Steps[0])

		out, err := json.MarshalIndent(cfg, "\t", "\t")
		require.NoError(t, err, "error encoding with encoding/json")
		str := string(out)

		// Same text after export
		assert.JSONEq(t, jsonText, str)

		out, err = jsoniter.Marshal(cfg)
		require.NoError(t, err, "error encoding with jsoniter")
		assert.JSONEq(t, jsonText, string(out))
	})

	t.Run("ConfFloat64 json", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    data.ConfFloat64
			expected string
		}{
			{
				name:     "normal float",
				input:    3.14,
				expected: "3.14",
			},
			{
				name:     "+inf",
				input:    data.ConfFloat64(math.Inf(+1)),
				expected: "null",
			},
			{
				name:     "-inf",
				input:    data.ConfFloat64(math.Inf(-1)),
				expected: "null",
			},
			{
				name:     "nan",
				input:    data.ConfFloat64(math.NaN()),
				expected: "null",
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cf := data.ConfFloat64(tc.input)
				out, err := cf.MarshalJSON()
				require.NoError(t, err)
				assert.Equal(t, tc.expected, string(out))

				// non pointer
				out, err = json.Marshal(cf)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, string(out))

				// pointer value
				out, err = json.Marshal(&cf)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, string(out))

				threshold := &data.Threshold{
					Value: tc.input,
					Color: "red",
					State: "x",
				}
				out, err = json.Marshal(threshold)
				require.NoError(t, err)

				out = append([]byte("["), out...)
				out = append(out, byte(']'))
				thresholds := []data.Threshold{}
				err = json.Unmarshal(out, &thresholds)
				require.NoError(t, err)
				require.Len(t, thresholds, 1)

				f := float64(tc.input)
				if math.IsNaN(f) || math.IsInf(f, 0) || math.IsInf(f, -1) {
					require.Equal(t, data.ConfFloat64(math.Inf(-1)), thresholds[0].Value)
				} else {
					require.Equal(t, tc.input, thresholds[0].Value)
				}
				require.Equal(t, "red", thresholds[0].Color)
				require.Equal(t, "x", thresholds[0].State)
			})
		}
	})

	t.Run("threshold Marshal/Unmarshal", func(t *testing.T) {
		testCases := []struct {
			name   string
			before data.Threshold
			after  data.Threshold
		}{
			{
				name: "empty value is zero",
				before: data.Threshold{
					Color: "red",
				},
				after: data.Threshold{
					Value: data.ConfFloat64(0), // default value
					Color: "red",
				},
			}, {
				name: "nan to -inf",
				before: data.Threshold{
					Value: data.ConfFloat64(math.NaN()), // json null
					Color: "orange",
				},
				after: data.Threshold{
					Value: data.ConfFloat64(math.Inf(-1)),
					Color: "orange",
				},
			}, {
				name: "+inf to -inf",
				before: data.Threshold{
					Value: data.ConfFloat64(math.Inf(+1)), // json null
					Color: "orange",
				},
				after: data.Threshold{
					Value: data.ConfFloat64(math.Inf(-1)),
					Color: "orange",
				},
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				out, err := json.Marshal([]data.Threshold{tc.before})
				require.NoError(t, err)

				arr := []data.Threshold{}
				err = json.Unmarshal(out, &arr)
				require.NoError(t, err)

				require.Equal(t, tc.after, arr[0])
			})
		}
	})
}
