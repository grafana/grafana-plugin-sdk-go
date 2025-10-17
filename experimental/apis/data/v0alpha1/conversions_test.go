package v0alpha1

import (
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

	jsonData := `{"refId":"A","_timeRange":{"from":"12345678","to":"87654321"},"datasource":{"type":"prometheus","uid":"hello-world"},"queryType":"interesting","maxDataPoints":42,"intervalMs":15,"key1":"value1","key2":"value2"}`

	require.Equal(t, jsonData, string(bq.JSON))
}
