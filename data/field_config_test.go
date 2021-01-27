package data_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
)

func TestReadMappings(t *testing.T) {
	jsonText := `{
		"description": "turn on/off system. write 1 to turn on the system and write 0 to turn off the system",
		"writeable": true,
		"mappings": [
		  {
			"text": "OFF",
			"value": "0"
		  },
		  {
			"type": 1,
			"text": "ON",
			"value": "1"
		  },
		  {
			"type": 2,
			"text": "0-100",
			"from": "0",
			"to": "100"
		  }
		]
	}`

	cfg := &data.FieldConfig{}
	err := json.Unmarshal([]byte(jsonText), &cfg)
	assert.NoError(t, err, "error parsing json")

	assert.Len(t, cfg.Mappings, 3)

	out, err := json.MarshalIndent(cfg, "\t", "\t")
	assert.NoError(t, err, "error parsing json")
	str := string(out)

	fmt.Printf("%s", str)

	assert.JSONEq(t, `{
		"description": "turn on/off system. write 1 to turn on the system and write 0 to turn off the system",
		"mappings": [
			{
				"text": "OFF",
				"value": "0"
			},
			{
				"text": "ON",
				"type": 1,
				"value": "1"
			},
			{
				"text": "0-100",
				"type": 2,
				"from": "0",
				"to": "100"
			}
		]
	}`, str)
}
