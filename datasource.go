package grafana

import (
	"context"
	"encoding/json"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/datasource"
	plugin "github.com/hashicorp/go-plugin"
)

// TimeRange represents a time range for a query.
type TimeRange struct {
	From time.Time
	To   time.Time
}

// DataSourceInfo holds metadata for the queried data source.
type DataSourceInfo struct {
	ID       int64
	OrgID    int64
	Name     string
	Type     string
	URL      string
	JSONData json.RawMessage
}

// Query represents the query as sent from the frontend.
type Query struct {
	RefID         string
	MaxDataPoints int64
	Interval      time.Duration
	ModelJSON     json.RawMessage
}

// QueryResult holds the results for a given query.
type QueryResult struct {
	Error      string
	RefID      string
	MetaJSON   string
	DataFrames []*dataframe.Frame
}

// DataSourceHandler handles data source queries.
type DataSourceHandler interface {
	Query(ctx context.Context, tr TimeRange, ds DataSourceInfo, queries []Query, api GrafanaAPIHandler) ([]QueryResult, error)
}

// datasourcePluginWrapper converts to and from protobuf types.
type datasourcePluginWrapper struct {
	plugin.NetRPCUnsupportedPlugin

	handler DataSourceHandler
}

func (p *datasourcePluginWrapper) Query(ctx context.Context, req *datasource.DatasourceRequest, api GrafanaAPI) (*datasource.DatasourceResponse, error) {
	tr := TimeRange{
		From: time.Unix(0, req.TimeRange.FromEpochMs*int64(time.Millisecond)),
		To:   time.Unix(0, req.TimeRange.FromEpochMs*int64(time.Millisecond)),
	}

	dsi := DataSourceInfo{
		ID:       req.Datasource.Id,
		OrgID:    req.Datasource.OrgId,
		Name:     req.Datasource.Name,
		Type:     req.Datasource.Type,
		URL:      req.Datasource.Url,
		JSONData: json.RawMessage(req.Datasource.JsonData),
	}

	var queries []Query
	for _, q := range req.Queries {
		queries = append(queries, Query{
			RefID:         q.RefId,
			MaxDataPoints: q.MaxDataPoints,
			Interval:      time.Duration(q.IntervalMs) * time.Millisecond,
			ModelJSON:     []byte(q.ModelJson),
		})
	}

	results, err := p.handler.Query(ctx, tr, dsi, queries, &grafanaAPIWrapper{api: api})
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &datasource.DatasourceResponse{
			Results: []*datasource.QueryResult{},
		}, nil
	}

	var respResults []*datasource.QueryResult

	for _, res := range results {
		// var tss []*datasource.TimeSeries
		// var tbs []*datasource.Table

		// for _, df := range res.DataFrames {
		// 	if len(df.Fields) == 0 {
		// 		continue
		// 	}

		// 	// Attempt to convert data frame to time series.
		// 	// Otherwise convert it to a table.
		// 	ts, err := asTimeSeries(df)
		// 	if err != nil {
		// 		tbs = append(tbs, asTable(df))
		// 	} else {
		// 		tss = append(tss, ts)
		// 	}
		// }

		queryResult := &datasource.QueryResult{
			Error:    res.Error,
			RefId:    res.RefID,
			MetaJson: res.MetaJSON,
			// Series:   tss,
			// Tables:   tbs,
		}

		respResults = append(respResults, queryResult)
	}

	return &datasource.DatasourceResponse{
		Results: respResults,
	}, nil
}

// asTimeSeries converts the data frame into a protobuf time series.
//
// It will use the first time field found as the timestamp, and the first
// number field as the value.
// func asTimeSeries(df *dataframe.Frame) (*datasource.TimeSeries, error) {
// 	timeIdx := indexOfFieldType(df, dataframe.FieldTypeTime)
// 	timeVec := df.Fields[timeIdx].Vector

// 	valueIdx := indexOfFieldType(df, dataframe.FieldTypeNumber)
// 	valueVec := df.Fields[valueIdx].Vector

// 	if timeIdx < 0 || valueIdx < 0 {
// 		return nil, errors.New("invalid time series")
// 	}

// 	pts := []*datasource.Point{}

// 	for i := 0; i < timeVec.Len(); i++ {
// 		t := timeVec.At(i).(*time.Time)
// 		v := valueVec.At(i).(*float64)

// 		pts = append(pts, &datasource.Point{
// 			Timestamp: int64(t.UnixNano()) / int64(time.Millisecond),
// 			Value:     *v,
// 		})
// 	}

// 	return &datasource.TimeSeries{
// 		Name:   df.Name,
// 		Tags:   df.Labels,
// 		Points: pts,
// 	}, nil
// }

// asTable converts the data frame into a protobuf table.
// func asTable(df *dataframe.Frame) *datasource.Table {
// 	if len(df.Fields) == 0 {
// 		return &datasource.Table{}
// 	}

// 	rows := make([]*datasource.TableRow, df.Fields[0].Len())

// 	for i := 0; i < len(rows); i++ {
// 		rowvals := make([]*datasource.RowValue, len(df.Fields))

// 		for j, f := range df.Fields {
// 			switch f.Type {
// 			case dataframe.FieldTypeTime:
// 				rowvals[j] = &datasource.RowValue{
// 					Type:        datasource.RowValue_TYPE_DOUBLE,
// 					DoubleValue: float64(f.Vector.At(i).(*time.Time).UnixNano()),
// 				}
// 			case dataframe.FieldTypeNumber:
// 				v := f.Vector.At(i).(*float64)
// 				rowvals[j] = &datasource.RowValue{
// 					Type:        datasource.RowValue_TYPE_DOUBLE,
// 					DoubleValue: *v,
// 				}
// 			case dataframe.FieldTypeString:
// 				v := f.Vector.At(i).(*string)
// 				rowvals[j] = &datasource.RowValue{
// 					Type:        datasource.RowValue_TYPE_STRING,
// 					StringValue: *v,
// 				}
// 			case dataframe.FieldTypeBoolean:
// 				v := f.Vector.At(i).(*bool)
// 				rowvals[j] = &datasource.RowValue{
// 					Type:      datasource.RowValue_TYPE_BOOL,
// 					BoolValue: *v,
// 				}
// 			}
// 		}

// 		rows[i] = &datasource.TableRow{
// 			Values: rowvals,
// 		}
// 	}

// 	var cols []*datasource.TableColumn
// 	for _, f := range df.Fields {
// 		cols = append(cols, &datasource.TableColumn{Name: f.Name})
// 	}

// 	return &datasource.Table{
// 		Columns: cols,
// 		Rows:    rows,
// 	}
// }

func indexOfFieldType(df *dataframe.Frame, t dataframe.FieldType) int {
	for idx, f := range df.Fields {
		if f.Type == t {
			return idx
		}
	}
	return -1
}

// DatasourceQueryResult holds the results for a given query.
type DatasourceQueryResult struct {
	Error      string
	RefID      string
	MetaJSON   string
	DataFrames []*dataframe.Frame
}

// GrafanaAPIHandler handles data source queries.
type GrafanaAPIHandler interface {
	QueryDatasource(ctx context.Context, orgID int64, datasourceID int64, tr TimeRange, queries []Query) ([]DatasourceQueryResult, error)
}

// grafanaAPIWrapper converts to and from Grafana types for calls from a datasource.
type grafanaAPIWrapper struct {
	api GrafanaAPI
}

func (w *grafanaAPIWrapper) QueryDatasource(ctx context.Context, orgID int64, datasourceID int64, tr TimeRange, queries []Query) ([]DatasourceQueryResult, error) {
	rawQueries := make([]*datasource.Query, 0, len(queries))

	for _, q := range queries {
		rawQueries = append(rawQueries, &datasource.Query{
			RefId:         q.RefID,
			MaxDataPoints: q.MaxDataPoints,
			IntervalMs:    q.Interval.Milliseconds(),
			ModelJson:     string(q.ModelJSON),
		})
	}

	rawResp, err := w.api.QueryDatasource(ctx, &datasource.QueryDatasourceRequest{
		OrgId:        orgID,
		DatasourceId: datasourceID,
		TimeRange: &datasource.TimeRange{
			FromEpochMs: int64(tr.From.UnixNano()) / int64(time.Millisecond),
			ToEpochMs:   int64(tr.To.UnixNano()) / int64(time.Millisecond),
		},
		Queries: rawQueries,
	})
	if err != nil {
		return nil, err
	}

	_ = rawResp

	//vals := make([]dataframe.Frame, 0)
	// for _, dsRes := range rawResp.Results {
	// 	vals = append(vals, FromGRPC(dsRes.GetSeries()).Values...)
	// }

	resp := []DatasourceQueryResult{}
	// TODO: Convert to dataframes?

	return resp, nil
}
