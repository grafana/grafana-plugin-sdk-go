package resource

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseQueriesIntoQueryDataRequest(t *testing.T) {
	request := []byte(`{
		"queries": [
			{
				"refId": "A",
				"datasource": {
					"type": "grafana-googlesheets-datasource",
					"uid": "b1808c48-9fc9-4045-82d7-081781f8a553"
				},
				"cacheDurationSeconds": 300,
				"spreadsheet": "spreadsheetID",
				"datasourceId": 4,
				"intervalMs": 30000,
				"maxDataPoints": 794
			},
			{
				"refId": "Z",
				"datasource": "old",
				"maxDataPoints": 10,
				"timeRange": {
					"from": "100",
					"to": "200"
				}
			}
		],
		"from": "1692624667389",
		"to": "1692646267389"
	}`)

	req := &GenericQueryRequest{}
	err := json.Unmarshal(request, req)
	require.NoError(t, err)

	require.Len(t, req.Queries, 2)
	require.Equal(t, "b1808c48-9fc9-4045-82d7-081781f8a553", req.Queries[0].Datasource.UID)
	require.Equal(t, "spreadsheetID", req.Queries[0].GetString("spreadsheet"))

	// Write the query (with additional spreadsheetID) to JSON
	out, err := json.MarshalIndent(req.Queries[0], "", "  ")
	require.NoError(t, err)

	// And read it back with standard JSON marshal functions
	query := &GenericDataQuery{}
	err = json.Unmarshal(out, query)
	require.NoError(t, err)
	require.Equal(t, "spreadsheetID", query.GetString("spreadsheet"))

	// The second query has an explicit time range, and legacy datasource name
	out, err = json.MarshalIndent(req.Queries[1], "", "  ")
	require.NoError(t, err)
	// fmt.Printf("%s\n", string(out))
	require.JSONEq(t, `{
		"datasource": {
		  "type": "", ` /* NOTE! this implies legacy naming */ +`
		  "uid": "old"
		},
		"maxDataPoints": 10,
		"refId": "Z",
		"timeRange": {
		  "from": "100",
		  "to": "200"
		}
	  }`, string(out))
}
