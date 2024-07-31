package data_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
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

	out, err := json.MarshalIndent(cfg, "\t", "\t")
	require.NoError(t, err, "error parsing json")
	str := string(out)

	fmt.Printf("%s", str)

	// Same text after export
	assert.JSONEq(t, jsonText, str)
}

func TestInternalDataLinks(t *testing.T) {
	jsonText := `{
		"links": [
			{
				"internal": {
					"query": "test 1",
					"range": {
						"from": "2000-01-01T00:00:00Z",
						"to": "2000-01-02T00:00:00Z",
						"raw": {
							"from": "2000-01-01T00:00:00Z",
							"to": "2000-01-02T00:00:00Z"
						}
					}
				}
			},
			{
				"internal": {
					"query": "test 2",
					"range": {
						"from": "2000-01-01T00:00:00Z",
						"to": "2000-01-02T00:00:00Z",
						"raw": {
							"from": "now-1h",
							"to": "now"
						}
					}
				}
			},
			{
				"internal": {
					"query": "test 3",
					"range": {
						"from": "2000-01-01T00:00:00Z",
						"to": "2000-01-02T00:00:00Z"
					}
				}
			}
		]
	}`

	cfg := data.FieldConfig{}
	err := json.Unmarshal([]byte(jsonText), &cfg)

	require.NoError(t, err, "error parsing json")

	out, err := json.MarshalIndent(cfg, "\t", "\t")
	require.NoError(t, err, "error parsing json")
	str := string(out)
	assert.JSONEq(t, jsonText, str)
}

func TestCreateInternalDataLinks(t *testing.T) {
	linkTime, _ := time.Parse("2006-01-02 15:04:05.000", "2000-01-01 00:00:00.000")
	cfg := data.FieldConfig{
		Links: []data.DataLink{
			{
				Internal: &data.InternalDataLink{
					Query: "test 1",
					Range: &data.TimeRange{
						From: linkTime,
						To:   linkTime,
						Raw: &data.RawTimeRangeTime{
							From: linkTime,
							To:   linkTime,
						},
					},
				},
			},
			{
				Internal: &data.InternalDataLink{
					Query: "test 2",
					Range: &data.TimeRange{
						From: linkTime,
						To:   linkTime,
						Raw: &data.RawTimeRangeString{
							From: "now-1h",
							To:   "now",
						},
					},
				},
			},
			{
				Internal: &data.InternalDataLink{
					Query: "test 3",
					Range: &data.TimeRange{
						From: linkTime,
						To:   linkTime,
					},
				},
			},
		},
	}

	expectedJson := `{
		"links": [
			{
				"internal": {
					"query": "test 1",
					"range": {
						"from": "2000-01-01T00:00:00Z",
						"to": "2000-01-01T00:00:00Z",
						"raw": {
							"from": "2000-01-01T00:00:00Z",
							"to": "2000-01-01T00:00:00Z"
						}
					}
				}
			},
			{
				"internal": {
					"query": "test 2",
					"range": {
						"from": "2000-01-01T00:00:00Z",
						"to": "2000-01-01T00:00:00Z",
						"raw": {
							"from": "now-1h",
							"to": "now"
						}
					}
				}
			},
			{
				"internal": {
					"query": "test 3",
					"range": {
						"from": "2000-01-01T00:00:00Z",
						"to": "2000-01-01T00:00:00Z"
					}
				}
			}
		]
	}`

	out, err := json.MarshalIndent(cfg, "\t", "\t")
	require.NoError(t, err, "error marshalling json")

	str := string(out)
	assert.JSONEq(t, expectedJson, str)
}
