package schemabuilder

import (
	"github.com/grafana/grafana-plugin-sdk-go/experimental/resource"
)

func exampleRequest(defs resource.QueryTypeDefinitionList) resource.DataQueryRequest {
	rsp := resource.DataQueryRequest{
		TimeRange: resource.TimeRange{
			From: "now-1h",
			To:   "now",
		},
		Queries: []resource.DataQuery{},
	}

	for _, def := range defs.Items {
		for _, sample := range def.Spec.Examples {
			if sample.SaveModel.Object != nil {
				q := resource.NewDataQuery(sample.SaveModel.Object)
				q.RefID = string(rune('A' + len(rsp.Queries)))
				for _, dis := range def.Spec.Discriminators {
					_ = q.Set(dis.Field, dis.Value)
				}

				if q.MaxDataPoints < 1 {
					q.MaxDataPoints = 1000
				}
				if q.IntervalMS < 1 {
					q.IntervalMS = 5
				}

				rsp.Queries = append(rsp.Queries, q)
			}
		}
	}
	return rsp
}

func examplePanelTargets(ds *resource.DataSourceRef, defs resource.QueryTypeDefinitionList) []resource.DataQuery {
	targets := []resource.DataQuery{}

	for _, def := range defs.Items {
		for _, sample := range def.Spec.Examples {
			if sample.SaveModel.Object != nil {
				q := resource.NewDataQuery(sample.SaveModel.Object)
				q.Datasource = ds
				q.RefID = string(rune('A' + len(targets)))
				for _, dis := range def.Spec.Discriminators {
					_ = q.Set(dis.Field, dis.Value)
				}
				targets = append(targets, q)
			}
		}
	}
	return targets
}
