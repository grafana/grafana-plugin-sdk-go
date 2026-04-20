package v0alpha1

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func benchQuery(withPerQueryTimeRange bool) DataQuery {
	q := DataQuery{
		CommonQueryProperties: CommonQueryProperties{
			RefID: "A",
			Datasource: &DataSourceRef{
				Type: "prometheus",
				UID:  "hello-world",
			},
			QueryType:     "interesting",
			MaxDataPoints: 1000,
			IntervalMS:    15,
		},
	}
	if withPerQueryTimeRange {
		q.TimeRange = &TimeRange{From: "12345678", To: "87654321"}
	}
	q.Set("expr", "up{job=\"prometheus\"}")
	q.Set("legendFormat", "{{ instance }}")
	return q
}

func BenchmarkToBackendDataQuery(b *testing.B) {
	defaultTR := &backend.TimeRange{}

	b.Run("no_per_query_timerange", func(b *testing.B) {
		q := benchQuery(false)
		b.ReportAllocs()
		for b.Loop() {
			if _, err := toBackendDataQuery(q, defaultTR); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("with_per_query_timerange", func(b *testing.B) {
		q := benchQuery(true)
		b.ReportAllocs()
		for b.Loop() {
			if _, err := toBackendDataQuery(q, defaultTR); err != nil {
				b.Fatal(err)
			}
		}
	})
}
