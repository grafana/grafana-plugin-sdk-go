package querygen

import (
	"testing"

	v0alpha1 "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/testutil"
	"github.com/stretchr/testify/require"
)

const queryModelName = "Query"

func TestBuildDefinitionsInModule(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/query.go": `
package models

type Query struct {
	QueryType string ` + "`json:\"queryType\"`" + `
	Owner     string ` + "`json:\"owner\"`" + `
	Repo      string ` + "`json:\"repo\"`" + `
}
`,
	})

	definitions, err := BuildDefinitionsInModule(RuntimeOptions{
		Dir:      dir,
		PluginID: []string{"github-datasource"},
	}, []RuntimeRegistration{{
		PackagePath:    "fixture/pkg/models",
		TypeName:       queryModelName,
		Name:           "Pull_Requests",
		Description:    "GitHub pull request query",
		Discriminators: []v0alpha1.DiscriminatorFieldValue{{Field: queryTypeFieldName, Value: "Pull_Requests"}},
		Examples: []v0alpha1.QueryExample{{
			Name: "simple",
			SaveModel: v0alpha1.AsUnstructured(map[string]any{
				queryTypeFieldName: "Pull_Requests",
				"owner":            "grafana",
				"repo":             "grafana",
			}),
		}},
	}})
	require.NoError(t, err, "build failed")

	require.Len(t, definitions.Items, 1, "expected one item")
	item := definitions.Items[0]
	require.Equal(t, "Pull_Requests", item.Name, "unexpected item name")
	require.Equal(t, "GitHub pull request query", item.Spec.Description, "unexpected description")

	properties := item.Spec.Schema.Spec.Properties
	require.ElementsMatch(t, []string{"owner", "queryType", "repo"}, testutil.KeysOfMap(properties), "expected exact query schema property set")
}

func TestBuildDefinitionsFromFindingsSupportsUnexportedTypes(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/query.go": `
package models

type query struct {
	QueryType string ` + "`json:\"queryType\"`" + `
}
`,
	})

	definitions, warnings, err := BuildDefinitionsFromFindings(RuntimeOptions{
		Dir:      dir,
		PluginID: []string{"example"},
	}, []model.Finding{{
		Source: model.SourceKindQueryJSON,
		Target: &model.TargetRef{
			PackagePath: "fixture/pkg/models",
			TypeName:    "query",
		},
	}})
	require.NoError(t, err, "unexpected error")
	require.Len(t, definitions.Items, 1, "expected one item")
	require.Empty(t, warnings, "expected no warnings")
}

func TestBuildDefinitionsFromFindingsSkipsUntaggedRuntimeOnlyQueryFields(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/query.go": `
package models

type Query struct {
	QueryType string ` + "`json:\"queryType\"`" + `
	Query     string ` + "`json:\"query\"`" + `
	RuntimeOnly string
}
`,
	})

	definitions, warnings, err := BuildDefinitionsFromFindings(RuntimeOptions{
		Dir: dir,
	}, []model.Finding{{
		Source: model.SourceKindQueryJSON,
		Target: &model.TargetRef{
			PackagePath: "fixture/pkg/models",
			TypeName:    queryModelName,
		},
	}})
	require.NoError(t, err, "unexpected error")
	require.Empty(t, warnings, "expected no warnings")
	require.Len(t, definitions.Items, 1, "expected one item")

	properties := definitions.Items[0].Spec.Schema.Spec.Properties
	require.ElementsMatch(t, []string{"query", "queryType"}, testutil.KeysOfMap(properties), "expected exact tagged query property set")
}

func TestBuildDefinitionsFromFindingsBuildsSQLDSQuerySchemaFromFinding(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1

require (
	github.com/grafana/grafana-plugin-sdk-go v0.0.0
)

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.26.1
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

import "encoding/json"

type DataSourceInstanceSettings struct{}

type DataQuery struct {
	RefID string
	JSON  json.RawMessage
}

type TimeRange struct{}
`,
		"stubs/grafana-plugin-sdk-go/data/data.go": `
package data

type FillMissing struct{}
`,
		"stubs/grafana-plugin-sdk-go/data/sqlutil/query.go": `
package sqlutil

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type FormatQueryOption uint32

type Query struct {
	RawSQL         string            ` + "`json:\"rawSql\"`" + `
	Format         FormatQueryOption ` + "`json:\"format\"`" + `
	ConnectionArgs json.RawMessage   ` + "`json:\"connectionArgs\"`" + `
	RefID          string            ` + "`json:\"-\"`" + `
	TimeRange      backend.TimeRange ` + "`json:\"-\"`" + `
	FillMissing    *data.FillMissing ` + "`json:\"fillMode,omitempty\"`" + `
}
`,
		"pkg/plugin/plugin.go": `
package plugin

import "github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"

var _ sqlutil.Query
`,
	})

	definitions, warnings, err := BuildDefinitionsFromFindings(RuntimeOptions{
		Dir: dir,
	}, []model.Finding{{
		Source: model.SourceKindQueryJSON,
		Target: &model.TargetRef{
			PackagePath: "github.com/grafana/grafana-plugin-sdk-go/data/sqlutil",
			TypeName:    queryModelName,
		},
	}})
	require.NoError(t, err, "unexpected error")
	require.Empty(t, warnings, "expected no warnings")
	require.Len(t, definitions.Items, 1, "expected one query definition")
	item := definitions.Items[0]
	require.Equal(t, queryModelName, item.Name, "expected query name")
	require.Contains(t, item.Spec.Schema.Spec.Properties, "rawSql", "expected rawSql property in schema")
}

func TestBuildDefinitionsFromFindingsNormalizesInternalQueryTypeName(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/query.go": `
package models

type internalQueryModel struct {
	Expr string ` + "`json:\"expr\"`" + `
}
`,
	})

	definitions, warnings, err := BuildDefinitionsFromFindings(RuntimeOptions{
		Dir: dir,
	}, []model.Finding{{
		Source: model.SourceKindQueryJSON,
		Target: &model.TargetRef{
			PackagePath: "fixture/pkg/models",
			TypeName:    "internalQueryModel",
		},
	}})
	require.NoError(t, err, "unexpected error")
	require.Empty(t, warnings, "expected no warnings")
	require.Len(t, definitions.Items, 1, "expected one query definition")
	require.Equal(t, queryModelName, definitions.Items[0].Name, "expected normalized query name")
	require.Contains(t, definitions.Items[0].Spec.Schema.Spec.Properties, "expr", "expected expr property in schema")
}

func TestBuildDefinitionsFromFindingsInfersDiscriminatorFromQueryTypeMuxWrapper(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.26.1
`,
		"stubs/grafana-plugin-sdk-go/backend/data.go": `
package backend

import "encoding/json"

type DataQuery struct {
	RefID string
	JSON  json.RawMessage
}

type DataResponse struct{}

type Responses map[string]DataResponse

type QueryDataRequest struct {
	Queries []DataQuery
}

type QueryDataResponse struct {
	Responses Responses
}

func NewQueryDataResponse() *QueryDataResponse {
	return &QueryDataResponse{Responses: Responses{}}
}
`,
		"stubs/grafana-plugin-sdk-go/backend/datasource/query_type_mux.go": `
package datasource

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type QueryTypeMux struct{}

func NewQueryTypeMux() *QueryTypeMux { return &QueryTypeMux{} }

func (mux *QueryTypeMux) HandleFunc(queryType string, handler func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error)) {}
`,
		"pkg/models/query.go": `
package models

const QueryTypeCommits = "Commits"

type CommitsQuery struct {
	QueryType string ` + "`json:\"queryType\"`" + `
}
`,
		"pkg/gitlab/query_handler.go": `
package gitlab

import (
	"context"

	"fixture/pkg/models"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
)

type QueryHandler struct{}

type QueryHandlerFunc func(context.Context, backend.DataQuery) backend.DataResponse

func processQueries(ctx context.Context, req *backend.QueryDataRequest, handler QueryHandlerFunc) backend.Responses {
	res := backend.Responses{}
	for _, v := range req.Queries {
		res[v.RefID] = handler(ctx, v)
	}
	return res
}

func GetQueryHandlers(s *QueryHandler) *datasource.QueryTypeMux {
	mux := datasource.NewQueryTypeMux()
	mux.HandleFunc(models.QueryTypeCommits, s.HandleCommits)
	return mux
}
`,
		"pkg/gitlab/commits_handler.go": `
package gitlab

import (
	"context"
	"encoding/json"

	"fixture/pkg/dfutil"
	"fixture/pkg/models"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func (s *QueryHandler) handleCommitsQuery(_ context.Context, q backend.DataQuery) backend.DataResponse {
	query := &models.CommitsQuery{}
	_ = json.Unmarshal(q.JSON, query)
	return dfutil.FrameResponse()
}

func (s *QueryHandler) HandleCommits(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return &backend.QueryDataResponse{
		Responses: processQueries(ctx, req, s.handleCommitsQuery),
	}, nil
}
`,
		"pkg/dfutil/dfutil.go": `
package dfutil

import "github.com/grafana/grafana-plugin-sdk-go/backend"

func FrameResponse() backend.DataResponse {
	return backend.DataResponse{}
}
`,
	})

	definitions, warnings, err := BuildDefinitionsFromFindings(RuntimeOptions{Dir: dir}, []model.Finding{{
		Source:       model.SourceKindQueryJSON,
		FunctionName: "(*QueryHandler).handleCommitsQuery",
		Target: &model.TargetRef{
			PackagePath: "fixture/pkg/models",
			TypeName:    "CommitsQuery",
		},
	}})
	require.NoError(t, err, "unexpected error")
	require.Empty(t, warnings, "expected no warnings")
	require.Len(t, definitions.Items, 1, "expected one item")
	require.Equal(t, "Commits", definitions.Items[0].Name, "expected inferred item name")
	require.Len(t, definitions.Items[0].Spec.Discriminators, 1, "expected one discriminator")
	discriminator := definitions.Items[0].Spec.Discriminators[0]
	require.Equal(t, queryTypeFieldName, discriminator.Field, "unexpected discriminator field")
	require.Equal(t, "Commits", discriminator.Value, "unexpected discriminator value")
}

func TestBuildDefinitionsFromFindingsInfersMultipleDiscriminatorsForSharedType(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.26.1
`,
		"stubs/grafana-plugin-sdk-go/backend/data.go": `
package backend

import (
	"context"
	"encoding/json"
)

type DataQuery struct {
	RefID string
	JSON  json.RawMessage
}

type DataResponse struct{}

type Responses map[string]DataResponse

type QueryDataRequest struct {
	Queries []DataQuery
}

type QueryDataResponse struct {
	Responses Responses
}

func NewQueryDataResponse() *QueryDataResponse {
	return &QueryDataResponse{Responses: Responses{}}
}

type QueryDataHandlerFunc func(context.Context, *QueryDataRequest) (*QueryDataResponse, error)
`,
		"stubs/grafana-plugin-sdk-go/backend/datasource/query_type_mux.go": `
package datasource

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type QueryTypeMux struct{}

func NewQueryTypeMux() *QueryTypeMux { return &QueryTypeMux{} }

func (mux *QueryTypeMux) HandleFunc(queryType string, handler func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error)) {}
`,
		"pkg/models/query.go": `
package models

type QueryType string

const (
	QueryTypeStats QueryType = "AggregateAPI"
	QueryTypeTable QueryType = "TableAPI"
)

type Query struct {
	QueryType string ` + "`json:\"queryType\"`" + `
}
`,
		"pkg/query_handlers.go": `
package main

import (
	"context"
	"encoding/json"

	"fixture/pkg/models"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
)

func getQueryTypeMux(opts *HandlerOpts) *datasource.QueryTypeMux {
	m := datasource.NewQueryTypeMux()
	m.HandleFunc(string(models.QueryTypeTable), HandleTableQuery(opts))
	m.HandleFunc(string(models.QueryTypeStats), HandleStatsQuery(opts))
	return m
}

type HandlerOpts struct{}

type QueryHandlerFunc func(context.Context, *HandlerOpts, backend.DataQuery) (backend.Responses, error)

func queryHandler(opts *HandlerOpts, handler QueryHandlerFunc) func(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return func(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		res := backend.NewQueryDataResponse()
		for _, v := range req.Queries {
			_, _ = handler(ctx, opts, v)
			res.Responses[v.RefID] = backend.DataResponse{}
		}
		return res, nil
	}
}

func queryHandlerTable(_ context.Context, _ *HandlerOpts, req backend.DataQuery) (backend.Responses, error) {
	query := models.Query{}
	if err := json.Unmarshal(req.JSON, &query); err != nil {
		return nil, err
	}
	return backend.Responses{}, nil
}

func queryHandlerStats(_ context.Context, _ *HandlerOpts, req backend.DataQuery) (backend.Responses, error) {
	query := models.Query{}
	if err := json.Unmarshal(req.JSON, &query); err != nil {
		return nil, err
	}
	return backend.Responses{}, nil
}

func HandleTableQuery(opts *HandlerOpts) func(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return queryHandler(opts, queryHandlerTable)
}

func HandleStatsQuery(opts *HandlerOpts) func(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return queryHandler(opts, queryHandlerStats)
}
`,
	})

	definitions, warnings, err := BuildDefinitionsFromFindings(RuntimeOptions{Dir: dir}, []model.Finding{
		{
			Source:       model.SourceKindQueryJSON,
			FunctionName: "queryHandlerTable",
			Target: &model.TargetRef{
				PackagePath: "fixture/pkg/models",
				TypeName:    queryModelName,
			},
		},
		{
			Source:       model.SourceKindQueryJSON,
			FunctionName: "queryHandlerStats",
			Target: &model.TargetRef{
				PackagePath: "fixture/pkg/models",
				TypeName:    queryModelName,
			},
		},
	})
	require.NoError(t, err, "unexpected error")
	require.Empty(t, warnings, "expected no warnings")
	require.Len(t, definitions.Items, 2, "expected one item per discriminator")
	require.ElementsMatch(t, []string{"AggregateAPI", "TableAPI"}, queryDefinitionNames(definitions.Items), "expected exact item names")

	got := map[string][]v0alpha1.DiscriminatorFieldValue{}
	for _, item := range definitions.Items {
		got[item.Name] = item.Spec.Discriminators
	}

	require.Len(t, got["AggregateAPI"], 1, "unexpected AggregateAPI discriminators")
	require.Equal(t, "AggregateAPI", got["AggregateAPI"][0].Value, "unexpected AggregateAPI discriminator value")
	require.Len(t, got["TableAPI"], 1, "unexpected TableAPI discriminators")
	require.Equal(t, "TableAPI", got["TableAPI"][0].Value, "unexpected TableAPI discriminator value")
}

func TestBuildDefinitionsFromFindingsInfersDiscriminatorsFromEnumField(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/query.go": `
package models

type QueryType string

const (
	QueryTypeMetrics QueryType = "metrics"
	QueryTypeSLO     QueryType = "slo"
	QueryTypeRaw     QueryType = "raw"
)

type Query struct {
	QueryType QueryType ` + "`json:\"queryType\"`" + `
	Query     string    ` + "`json:\"query\"`" + `
}
`,
	})

	definitions, warnings, err := BuildDefinitionsFromFindings(RuntimeOptions{Dir: dir}, []model.Finding{{
		Source: model.SourceKindQueryJSON,
		Target: &model.TargetRef{
			PackagePath: "fixture/pkg/models",
			TypeName:    queryModelName,
		},
	}})
	require.NoError(t, err, "unexpected error")
	require.Empty(t, warnings, "expected no warnings")
	require.Len(t, definitions.Items, 3, "expected one item per enum value")
	require.ElementsMatch(t, []string{"metrics", "raw", "slo"}, queryDefinitionNames(definitions.Items), "expected exact item names")

	got := map[string][]v0alpha1.DiscriminatorFieldValue{}
	for _, item := range definitions.Items {
		got[item.Name] = item.Spec.Discriminators
	}

	for _, name := range []string{"metrics", "raw", "slo"} {
		discriminators := got[name]
		require.Len(t, discriminators, 1, "expected one discriminator for %s", name)
		require.Equal(t, queryTypeFieldName, discriminators[0].Field, "unexpected discriminator field for %s", name)
		require.Equal(t, name, discriminators[0].Value, "unexpected discriminator value for %s", name)
	}
}

func TestBuildDefinitionsInModuleSuppressesRequiredFields(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/query.go": `
package models

type Query struct {
	QueryType string ` + "`json:\"queryType\"`" + `
	LogSearch struct {
		Query string ` + "`json:\"query\"`" + `
	} ` + "`json:\"logSearch\"`" + `
}
`,
	})

	definitions, err := BuildDefinitionsInModule(RuntimeOptions{Dir: dir}, []RuntimeRegistration{{
		PackagePath: "fixture/pkg/models",
		TypeName:    queryModelName,
	}})
	require.NoError(t, err, "build failed")

	require.Len(t, definitions.Items, 1, "expected one item")
	require.Empty(t, definitions.Items[0].Spec.Schema.Spec.Required, "did not expect required list")

	logSearch, ok := definitions.Items[0].Spec.Schema.Spec.Properties["logSearch"]
	require.True(t, ok, "expected logSearch property, got %#v", definitions.Items[0].Spec.Schema.Spec.Properties)
	require.Empty(t, logSearch.Required, "did not expect nested required list")
}

func TestAsJSONSchemaRejectsInvalidSchema(t *testing.T) {
	_, err := asJSONSchema(map[string]any{
		"type": 123,
	})
	require.Error(t, err, "expected schema conversion to fail")
	require.Contains(t, err.Error(), "expected string or array", "unexpected error")
}

func queryDefinitionNames(items []v0alpha1.QueryTypeDefinition) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}
