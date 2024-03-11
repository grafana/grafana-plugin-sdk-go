package v0alpha1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/apis/data/v0alpha1"
	"github.com/stretchr/testify/require"
)

func TestClientWithBadURL(t *testing.T) {
	client := v0alpha1.NewQueryDataClient("http://localhostXYZ:998/api/ds/query", nil, nil)
	code, _, err := client.QueryData(context.Background(), v0alpha1.QueryDataRequest{})
	require.Error(t, err)
	require.Equal(t, 404, code)
}

func TestQueryClient(t *testing.T) {
	t.Skip()

	client := v0alpha1.NewQueryDataClient("http://localhost:3000/api/ds/query", nil,
		map[string]string{
			"Authorization": "Bearer XYZ",
		})
	body := `{
		"from": "",
		"to": "",
		"queries": [
			{
				"refId": "X",
				"scenarioId": "csv_content",
				"datasource": {
					"type": "grafana-testdata-datasource",
					"uid": "PD8C576611E62080A"
				},
				"csvContent": "a,b,c\n1,hello,true",
				"hide": false
			}
		]
	}`
	qdr := v0alpha1.QueryDataRequest{}
	err := json.Unmarshal([]byte(body), &qdr)
	require.NoError(t, err)

	code, rsp, err := client.QueryData(context.Background(), qdr)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code)

	r, ok := rsp.Responses["X"]
	require.True(t, ok)

	for _, frame := range r.Frames {
		txt, err := frame.StringTable(20, 10)
		require.NoError(t, err)
		fmt.Printf("%s\n", txt)
	}
}
