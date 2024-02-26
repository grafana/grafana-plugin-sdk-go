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
	require.Equal(t, "spreadsheetID", req.Queries[0].MustString("spreadsheet"))

	// Write the query (with additional spreadsheetID) to JSON
	out, err := json.MarshalIndent(req.Queries[0], "", "  ")
	require.NoError(t, err)

	// And read it back with standard JSON marshal functions
	query := &GenericDataQuery{}
	err = json.Unmarshal(out, query)
	require.NoError(t, err)
	require.Equal(t, "spreadsheetID", query.MustString("spreadsheet"))

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

func TestQueryBuilders(t *testing.T) {
	prop := "testkey"
	testQ1 := &GenericDataQuery{}
	testQ1.Set(prop, "A")
	require.Equal(t, "A", testQ1.MustString(prop))

	testQ1.Set(prop, "B")
	require.Equal(t, "B", testQ1.MustString(prop))

	testQ2 := testQ1
	testQ2.Set(prop, "C")
	require.Equal(t, "C", testQ1.MustString(prop))
	require.Equal(t, "C", testQ2.MustString(prop))

	// Uses the official field when exists
	testQ2.Set("queryType", "D")
	require.Equal(t, "D", testQ2.QueryType)
	require.Equal(t, "D", testQ1.QueryType)
	require.Equal(t, "D", testQ2.MustString("queryType"))

	// Map constructor
	testQ3 := NewGenericDataQuery(map[string]any{
		"queryType": "D",
		"extra":     "E",
	})
	require.Equal(t, "D", testQ3.QueryType)
	require.Equal(t, "E", testQ3.MustString("extra"))
}
