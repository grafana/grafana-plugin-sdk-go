package v0alpha1

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConversionsDefaults(t *testing.T) {
	res, err := toBackendDataQuery(DataQuery{}, nil)

	require.NoError(t, err)

	// we used to default the refId to "A",
	// we do not do that anymore,
	// we verify the new behavior
	require.Equal(t, "", res.RefID)

	require.Equal(t, int64(100), res.MaxDataPoints)
	require.Equal(t, time.Second, res.Interval)
}

func TestToBackendDataQueryJSON(t *testing.T) {
	q := DataQuery{
		CommonQueryProperties: CommonQueryProperties{
			RefID: "A",
			TimeRange: &TimeRange{
				From: "12345678",
				To:   "87654321",
			},
			Datasource: &DataSourceRef{
				Type: "prometheus",
				UID:  "hello-world",
			},
			QueryType:     "interesting",
			MaxDataPoints: 42,
			IntervalMS:    15.0,
		},
	}

	q.Set("key1", "value1")
	q.Set("key2", "value2")

	bq, err := toBackendDataQuery(q, nil)
	require.NoError(t, err)

	require.Equal(t, "A", bq.RefID)
	require.Equal(t, "interesting", bq.QueryType)
	require.Equal(t, int64(42), bq.MaxDataPoints)
	require.Equal(t, time.Millisecond*15, bq.Interval)

	require.NotNil(t, bq.TimeRange)
	require.Equal(t, time.UnixMilli(12345678).UTC(), bq.TimeRange.From)
	require.Equal(t, time.UnixMilli(87654321).UTC(), bq.TimeRange.To)

	jsonData := `{` +
		`"datasource":{"type":"prometheus","uid":"hello-world"},` +
		`"intervalMs":15,` +
		`"key1":"value1",` +
		`"key2":"value2",` +
		`"maxDataPoints":42,` +
		`"queryType":"interesting",` +
		`"refId":"A"` +
		`}`

	require.Equal(t, jsonData, string(bq.JSON))
}

func TestToDataSourceQueriesTimeRangeHandling(t *testing.T) {
	data := `
		{
		"queries": [
			{
				"datasource": {
					"type": "prometheus",
					"uid": "prom1"
				},
				"expr": "111",
				"refId": "A"
			},
			{
				"datasource": {
					"type": "prometheus",
					"uid": "prom1"
				},
				"expr": "222",
				"refId": "B",
				"timeRange": {
					"from": "1763114120000",
					"to": "1763114130000"
				}
			}
		],
		"from": "1763114100000",
		"to": "1763114110000"
	}
	`

	var req QueryDataRequest

	err := json.Unmarshal([]byte(data), &req)
	require.NoError(t, err)

	queries, _, err := ToDataSourceQueries(req)
	require.NoError(t, err)

	require.Len(t, queries, 2)

	a := queries[0]
	require.Equal(t, "A", a.RefID)
	b := queries[1]
	require.Equal(t, "B", b.RefID)

	require.Equal(t, time.UnixMilli(1763114100000).UTC(), a.TimeRange.From)
	require.Equal(t, time.UnixMilli(1763114110000).UTC(), a.TimeRange.To)
	jsonA := `{` +
		`"refId":"A",` +
		`"datasource":{"type":"prometheus","uid":"prom1"},` +
		`"expr":"111"` +
		`}`
	require.Equal(t, jsonA, string(a.JSON))

	require.Equal(t, time.UnixMilli(1763114120000).UTC(), b.TimeRange.From)
	require.Equal(t, time.UnixMilli(1763114130000).UTC(), b.TimeRange.To)
	jsonB := `{` +
		`"datasource":{"type":"prometheus","uid":"prom1"},` +
		`"expr":"222",` +
		`"refId":"B"` +
		`}`
	require.Equal(t, jsonB, string(b.JSON))
}

func TestDeleteTimeRangeFromQueryJSON(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		expected      []byte
		expectedError bool
	}{
		{
			name:          "invalid json",
			data:          []byte("hello world"),
			expectedError: true,
		},
		{
			name:          "with time range",
			data:          []byte(`{"f1":{"f2":42},"timeRange":{"from":"111","to":"222"},"f3":"v3"}`),
			expected:      []byte(`{"f1":{"f2":42},"f3":"v3"}`),
			expectedError: false,
		},
		{
			name:          "without time range",
			data:          []byte(`{"f1":{"f2":42},"f3":"v3"}`),
			expected:      []byte(`{"f1":{"f2":42},"f3":"v3"}`),
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := deleteTimeRangeFromQueryJSON(tt.data)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}
