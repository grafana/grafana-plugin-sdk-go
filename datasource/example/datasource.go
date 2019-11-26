package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	"github.com/grafana/grafana-plugin-sdk-go/datasource"
)

const pluginID = "myorg-custom-datasource"

type MyDatasource struct {
	logger *log.Logger
}

func (d *MyDatasource) Query(ctx context.Context, tr datasource.TimeRange, ds datasource.DataSourceInfo, queries []datasource.Query) ([]datasource.QueryResult, error) {
	return []datasource.QueryResult{
		{
			RefID: "A",
			DataFrames: []*dataframe.Frame{
				dataframe.New("http_requests_total",
					dataframe.NewField("timestamp", nil, []time.Time{time.Now(), time.Now(), time.Now()}),
					dataframe.NewField("value", dataframe.Labels{"service": "auth", "env": "prod"}, []float64{45.0, 49.0, 29.0}),
				),
				dataframe.New("go_goroutines",
					dataframe.NewField("timestamp", nil, []int64{123238426, 123238456, 123238486}),
					dataframe.NewField("value", nil, []float64{45.0, 49.0, 29.0}),
				),
			},
		},
		{
			RefID: "B",
			DataFrames: []*dataframe.Frame{
				dataframe.New("organization",
					dataframe.NewField("department", dataframe.Labels{"component": "business"}, []string{"engineering", "sales"}),
					dataframe.NewField("num_employees", dataframe.Labels{"component": "business"}, []int64{20, 15}),
				),
			},
		},
	}, nil
}

func main() {
	logger := log.New(os.Stderr, "", 0)

	if err := datasource.Serve(pluginID, &MyDatasource{logger: logger}); err != nil {
		logger.Fatal(err)
	}
}
