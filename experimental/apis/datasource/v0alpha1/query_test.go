package v0alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSerializeAdditionalQueryFieldsOrdered(t *testing.T) {
	q := DataQuery{
		CommonQueryProperties: CommonQueryProperties{
			RefID:         "A",
			MaxDataPoints: 10,
			IntervalMS:    500,
		},
		additional: map[string]any{
			"utcOffsetSec": 3600,
			"exemplar":     false,
			"instant":      false,
			"range":        true,
			"editorMode":   "code",
			"legendFormat": "__auto",
		},
	}
	jsonBytes, err := json.Marshal(q)
	require.NoError(t, err)
	// NOTE: we cannot use require.JSONEq() here,
	// because we want to make sure the object-keys
	// are ordered
	expectedBytes := []byte(`{"refId":"A","maxDataPoints":10,"intervalMs":500,"editorMode":"code","exemplar":false,"instant":false,"legendFormat":"__auto","range":true,"utcOffsetSec":3600}`)
	require.Equal(t, expectedBytes, jsonBytes)
}

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
				"maxDataPoints": 794,
				"expr": "{filename=\"/var/log/docker/2e21a640173f1f3359e2a178905dad6b1b56e2da6e3d53dbdba0a1123cdcb4fe/json.log\"} | logfmt"
			},
			{
				"refId": "Z",
				"datasource": "old",
				"maxDataPoints": "10",
				"timeRange": {
					"from": "100",
					"to": "200"
				}
			}
		],
		"from": "1692624667389",
		"to": "1692646267389"
	}`)

	req := &QueryDataRequest{}
	err := json.Unmarshal(request, req)
	require.NoError(t, err)

	t.Run("verify raw unmarshal", func(t *testing.T) {
		require.Len(t, req.Queries, 2)
		require.Equal(t, "b1808c48-9fc9-4045-82d7-081781f8a553", req.Queries[0].Datasource.UID)
		require.Equal(t, "spreadsheetID", req.Queries[0].GetString("spreadsheet"))

		// Write the query (with additional spreadsheetID) to JSON
		out, err := json.MarshalIndent(req.Queries[0], "", "  ")
		require.NoError(t, err)

		// And read it back with standard JSON marshal functions
		query := &DataQuery{}
		err = json.Unmarshal(out, query)
		require.NoError(t, err)
		require.Equal(t, "spreadsheetID", query.GetString("spreadsheet"))
		require.Equal(t, int64(794), query.MaxDataPoints)         // input was a number
		require.Equal(t, int64(10), req.Queries[1].MaxDataPoints) // input was a string

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
	})

	t.Run("verify deep copy", func(t *testing.T) {
		cpy := req.DeepCopy()
		require.Equal(t, req, cpy)
	})

	t.Run("same results from either parser", func(t *testing.T) {
		typed := &QueryDataRequest{}
		err = json.Unmarshal(request, typed)
		require.NoError(t, err)

		out1, err := json.MarshalIndent(req, "", "  ")
		require.NoError(t, err)

		out2, err := json.MarshalIndent(typed, "", "  ")
		require.NoError(t, err)

		require.JSONEq(t, string(out1), string(out2))
	})
}

func TestLegacyDataSourceRef(t *testing.T) {
	type testWrapper struct {
		Ref DataSourceRef `json:"ref"`
	}

	wrap := &testWrapper{}
	err := json.Unmarshal([]byte(`{ "ref": {"type":"ttt", "uid":"UID", "apiVersion": "v2"}}`), wrap)
	require.NoError(t, err)
	require.Equal(t, "ttt", wrap.Ref.Type)
	require.Equal(t, "UID", wrap.Ref.UID)
	require.Equal(t, "v2", wrap.Ref.APIVersion)

	err = json.Unmarshal([]byte(`{ "ref": "name"}`), wrap)
	require.NoError(t, err)
	require.Equal(t, "", wrap.Ref.Type)
	require.Equal(t, "name", wrap.Ref.UID)

	ref := &DataSourceRef{}
	err = json.Unmarshal([]byte(`"aaa"`), ref) // string as reference
	require.NoError(t, err)
	require.Equal(t, "aaa", ref.UID)
}

func TestQueryBuilders(t *testing.T) {
	prop := "testkey"
	testQ1 := &DataQuery{}
	testQ1.Set(prop, "A")
	require.Equal(t, "A", testQ1.GetString(prop))

	testQ1.Set(prop, "B")
	require.Equal(t, "B", testQ1.GetString(prop))

	testQ2 := testQ1
	testQ2.Set(prop, "C")
	require.Equal(t, "C", testQ1.GetString(prop))
	require.Equal(t, "C", testQ2.GetString(prop))

	// Uses the official field when exists
	testQ2.Set("queryType", "D")
	require.Equal(t, "D", testQ2.QueryType)
	require.Equal(t, "D", testQ1.QueryType)
	require.Equal(t, "D", testQ2.GetString("queryType"))

	// Map constructor
	testQ3 := NewDataQuery(map[string]any{
		"queryType": "D",
		"extra":     "E",
	})
	require.Equal(t, "D", testQ3.QueryType)
	require.Equal(t, "E", testQ3.GetString("extra"))

	testQ3.Set("datasource", &DataSourceRef{Type: "TYPE", UID: "UID"})
	require.NotNil(t, testQ3.Datasource)
	require.Equal(t, "TYPE", testQ3.Datasource.Type)
	require.Equal(t, "UID", testQ3.Datasource.UID)

	testQ3.Set("datasource", map[string]any{"uid": "XYZ"})
	require.Equal(t, "XYZ", testQ3.Datasource.UID)

	testQ3.Set("maxDataPoints", 100)
	require.Equal(t, int64(100), testQ3.MaxDataPoints)
}
