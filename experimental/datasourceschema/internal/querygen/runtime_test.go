package querygen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	v0alpha1 "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
)

func TestBuildDefinitionsInModule(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
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
		TypeName:       "Query",
		Name:           "Pull_Requests",
		Description:    "GitHub pull request query",
		Discriminators: []v0alpha1.DiscriminatorFieldValue{{Field: "queryType", Value: "Pull_Requests"}},
		Examples: []v0alpha1.QueryExample{{
			Name: "simple",
			SaveModel: v0alpha1.AsUnstructured(map[string]any{
				"queryType": "Pull_Requests",
				"owner":     "grafana",
				"repo":      "grafana",
			}),
		}},
	}})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	if len(definitions.Items) != 1 {
		t.Fatalf("expected one item, got %d", len(definitions.Items))
	}
	item := definitions.Items[0]
	if item.Name != "Pull_Requests" {
		t.Fatalf("unexpected item name: %s", item.Name)
	}
	if item.Spec.Description != "GitHub pull request query" {
		t.Fatalf("unexpected description: %q", item.Spec.Description)
	}

	properties := item.Spec.Schema.Spec.Properties
	if _, ok := properties["owner"]; !ok {
		t.Fatalf("expected owner property in schema, got %#v", properties)
	}
}

func TestBuildDefinitionsFromFindingsSupportsUnexportedTypes(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(definitions.Items) != 1 {
		t.Fatalf("expected one item, got %#v", definitions.Items)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
}

func TestBuildDefinitionsFromFindingsSkipsUntaggedRuntimeOnlyQueryFields(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
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
			TypeName:    "Query",
		},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(definitions.Items) != 1 {
		t.Fatalf("expected one item, got %#v", definitions.Items)
	}

	properties := definitions.Items[0].Spec.Schema.Spec.Properties
	if _, ok := properties["query"]; !ok {
		t.Fatalf("expected tagged query property, got %#v", properties)
	}
	if _, ok := properties["runtimeOnly"]; ok {
		t.Fatalf("did not expect untagged runtime-only field in schema, got %#v", properties)
	}
}

func TestBuildDefinitionsFromFindingsBuildsSQLDSQuerySchemaFromFinding(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
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
			TypeName:    "Query",
		},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(definitions.Items) != 1 {
		t.Fatalf("expected one query definition, got %#v", definitions.Items)
	}
	item := definitions.Items[0]
	if item.Name != "Query" {
		t.Fatalf("expected query name, got %#v", item.Name)
	}
	if _, ok := item.Spec.Schema.Spec.Properties["rawSql"]; !ok {
		t.Fatalf("expected rawSql property in schema, got %#v", item.Spec.Schema.Spec.Properties)
	}
}

func TestBuildDefinitionsFromFindingsNormalizesInternalQueryTypeName(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(definitions.Items) != 1 {
		t.Fatalf("expected one query definition, got %#v", definitions.Items)
	}
	if definitions.Items[0].Name != "Query" {
		t.Fatalf("expected normalized query name, got %#v", definitions.Items[0].Name)
	}
	if _, ok := definitions.Items[0].Spec.Schema.Spec.Properties["expr"]; !ok {
		t.Fatalf("expected expr property in schema, got %#v", definitions.Items[0].Spec.Schema.Spec.Properties)
	}
}

func TestBuildDefinitionsFromFindingsInfersDiscriminatorFromQueryTypeMuxWrapper(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(definitions.Items) != 1 {
		t.Fatalf("expected one item, got %#v", definitions.Items)
	}
	if definitions.Items[0].Name != "Commits" {
		t.Fatalf("expected inferred item name, got %#v", definitions.Items[0].Name)
	}
	if len(definitions.Items[0].Spec.Discriminators) != 1 {
		t.Fatalf("expected one discriminator, got %#v", definitions.Items[0].Spec.Discriminators)
	}
	discriminator := definitions.Items[0].Spec.Discriminators[0]
	if discriminator.Field != "queryType" || discriminator.Value != "Commits" {
		t.Fatalf("unexpected discriminator: %#v", discriminator)
	}
}

func TestBuildDefinitionsFromFindingsInfersMultipleDiscriminatorsForSharedType(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
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
				TypeName:    "Query",
			},
		},
		{
			Source:       model.SourceKindQueryJSON,
			FunctionName: "queryHandlerStats",
			Target: &model.TargetRef{
				PackagePath: "fixture/pkg/models",
				TypeName:    "Query",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(definitions.Items) != 2 {
		t.Fatalf("expected one item per discriminator, got %#v", definitions.Items)
	}

	got := map[string][]v0alpha1.DiscriminatorFieldValue{}
	for _, item := range definitions.Items {
		got[item.Name] = item.Spec.Discriminators
	}

	if len(got["AggregateAPI"]) != 1 || got["AggregateAPI"][0].Value != "AggregateAPI" {
		t.Fatalf("unexpected AggregateAPI discriminators: %#v", got["AggregateAPI"])
	}
	if len(got["TableAPI"]) != 1 || got["TableAPI"][0].Value != "TableAPI" {
		t.Fatalf("unexpected TableAPI discriminators: %#v", got["TableAPI"])
	}
}

func TestBuildDefinitionsFromFindingsInfersDiscriminatorsFromEnumField(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
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
			TypeName:    "Query",
		},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(definitions.Items) != 3 {
		t.Fatalf("expected one item per enum value, got %#v", definitions.Items)
	}

	got := map[string][]v0alpha1.DiscriminatorFieldValue{}
	for _, item := range definitions.Items {
		got[item.Name] = item.Spec.Discriminators
	}

	for _, name := range []string{"metrics", "raw", "slo"} {
		discriminators := got[name]
		if len(discriminators) != 1 {
			t.Fatalf("expected one discriminator for %s, got %#v", name, discriminators)
		}
		if discriminators[0].Field != "queryType" || discriminators[0].Value != name {
			t.Fatalf("unexpected discriminator for %s: %#v", name, discriminators[0])
		}
	}
}

func TestBuildDefinitionsInModuleSuppressesRequiredFields(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
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
		TypeName:    "Query",
	}})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	if len(definitions.Items) != 1 {
		t.Fatalf("expected one item, got %#v", definitions.Items)
	}

	if len(definitions.Items[0].Spec.Schema.Spec.Required) > 0 {
		t.Fatalf("did not expect required list, got %#v", definitions.Items[0].Spec.Schema.Spec.Required)
	}

	logSearch, ok := definitions.Items[0].Spec.Schema.Spec.Properties["logSearch"]
	if !ok {
		t.Fatalf("expected logSearch property, got %#v", definitions.Items[0].Spec.Schema.Spec.Properties)
	}
	if len(logSearch.Required) > 0 {
		t.Fatalf("did not expect nested required list, got %#v", logSearch.Required)
	}
}

func TestAsJSONSchemaRejectsInvalidSchema(t *testing.T) {
	_, err := asJSONSchema(map[string]any{
		"type": 123,
	})
	if err == nil {
		t.Fatal("expected schema conversion to fail")
	}
	if !strings.Contains(err.Error(), "expected string or array") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeRuntimeFixture(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		fullPath := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir failed for %s: %v", fullPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(strings.TrimLeft(content, "\n")), 0o644); err != nil {
			t.Fatalf("write failed for %s: %v", fullPath, err)
		}
	}

	return dir
}
