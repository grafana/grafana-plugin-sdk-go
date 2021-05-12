package data_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadMappings(t *testing.T) {
	jsonText := `{
		"description": "turn on/off system. write 1 to turn on the system and write 0 to turn off the system",
		"writeable": true,
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
			  "type": "special",
			  "options": {
				"match": "nan",
				"result": {
				  "text": "Batman!",
				  "color": "dark-red",
				  "index": 3
				}
			  }
			}
		  ]
	}`

	cfg := &data.FieldConfig{}
	err := json.Unmarshal([]byte(jsonText), &cfg)
	require.NoError(t, err, "error parsing json")

	require.True(t, *cfg.Writeable)
	require.Len(t, cfg.Mappings, 3)

	out, err := json.MarshalIndent(cfg, "\t", "\t")
	require.NoError(t, err, "error parsing json")
	str := string(out)

	fmt.Printf("%s", str)

	// Same text after export
	assert.JSONEq(t, jsonText, str)
}
