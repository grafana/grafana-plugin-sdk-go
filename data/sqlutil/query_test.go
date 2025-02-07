package sqlutil_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
)

func TestGetQuery(t *testing.T) {
	t.Run("returns correct query", func(t *testing.T) {
		timeRange := backend.TimeRange{From: time.Now().Add(-time.Hour), To: time.Now()}
		dataQuery := backend.DataQuery{
			RefID:         "foo",
			MaxDataPoints: 10,
			Interval:      time.Second,
			TimeRange:     timeRange,
			JSON: json.RawMessage(`{
			"rawSql":"abc",
			"format":1,
			"connectionArgs":"options",
			"fillMode":{"mode":1},
			"schema":"x",
			"table":"y",
			"column":"z"
		}`),
		}

		parsedQuery, err := sqlutil.GetQuery(dataQuery)
		assert.NoError(t, err)
		assert.Equal(t, parsedQuery.RawSQL, "abc")
		assert.Equal(t, parsedQuery.Format, sqlutil.FormatOptionTable)
		assert.Equal(t, parsedQuery.ConnectionArgs, json.RawMessage(`"options"`))
		assert.Equal(t, parsedQuery.RefID, "foo")
		assert.Equal(t, parsedQuery.Interval, time.Second)
		assert.Equal(t, parsedQuery.TimeRange, timeRange)
		assert.Equal(t, parsedQuery.MaxDataPoints, int64(10))
		assert.Equal(t, parsedQuery.FillMissing.Mode, data.FillModeNull)
		assert.Equal(t, parsedQuery.Schema, "x")
		assert.Equal(t, parsedQuery.Table, "y")
		assert.Equal(t, parsedQuery.Column, "z")
	})

	t.Run("returns error if invalid query", func(t *testing.T) {
		timeRange := backend.TimeRange{From: time.Now().Add(-time.Hour), To: time.Now()}
		dataQuery := backend.DataQuery{
			RefID:         "foo",
			MaxDataPoints: 10,
			Interval:      time.Second,
			TimeRange:     timeRange,
			// invalid JSON, rawSql should be a string
			JSON: json.RawMessage(`{
			"rawSql": 1,
		}`),
		}

		_, err := sqlutil.GetQuery(dataQuery)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "error unmarshaling query JSON to the Query Model")
		assert.True(t, backend.IsDownstreamError(err))
	})
}
