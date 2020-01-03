package dataframe

// Matches:
// https://github.com/grafana/grafana/blob/master/packages/grafana-data/src/types/data.ts#L11
// NOTE -- in javascript this can accept any `[key: string]: any;` however
// this interface only exposes the values we want to be exposed
type QueryResultMeta struct {
	// Used in Explore for highlighting
	SearchWords []string `json:"searchWords,omitempty"`

	// Used in Explore to show limit applied to search result
	Limit int64 `json:"limit,omitempty"`
}
