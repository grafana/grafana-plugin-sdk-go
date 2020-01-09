package dataframe

import "encoding/json"

// QueryResultMeta matches:
// https://github.com/grafana/grafana/blob/master/packages/grafana-data/src/types/data.ts#L11
// NOTE -- in javascript this can accept any `[key: string]: any;` however
// this interface only exposes the values we want to be exposed
type QueryResultMeta struct {
	// Used in Explore for highlighting
	SearchWords []string `json:"searchWords,omitempty"`

	// Used in Explore to show limit applied to search result
	Limit int64 `json:"limit,omitempty"`

	// Visualization is so a Grafana visualization can be suggested with
	// the response such as "Graph", "Singlestat", or Gauge
	Visualization string `json:"visualization,omitempty"`
}

// QueryResultMetaFromJSON creates a QueryResultMeta from json string
func QueryResultMetaFromJSON(jsonStr string) (*QueryResultMeta, error) {
	var m QueryResultMeta
	err := json.Unmarshal([]byte(jsonStr), &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

const (
	VisGraph      = "grafana/graph"
	VisTable      = "grafana/table"
	VisSingleStat = "grafana/singlestat"
	VisGauge      = "grafana/gauge"
	VisBarGauge   = "grafana/bar_gauge"
	VisText       = "grafana/text"
	VisHeatMap    = "grafana/heatmap"
	VisStat       = "grafana/stat"
	VisLogs       = "grafana/logs"
)
