package analyze

import (
	"path/filepath"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestRunFindsDirectJSONTargets(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataSourceInstanceSettings struct {
	JSONData []byte
	DecryptedSecureJSONData map[string]string
}

type DataQuery struct {
	JSON []byte
}
`,
		"fixture.go": `
package fixture

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Settings struct {
	Name string ` + "`json:\"name\"`" + `
}

type Query struct {
	RefID string ` + "`json:\"refId\"`" + `
}

func LoadSettings(config backend.DataSourceInstanceSettings) error {
	var settings Settings
	return json.Unmarshal(config.JSONData, &settings)
}

func LoadQuery(q backend.DataQuery) error {
	var query Query
	return json.Unmarshal(q.JSON, &query)
}
`,
	})

	report, err := Run(Config{
		Dir:      dir,
		Patterns: []string{"./..."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.True(t, hasTarget(report.Findings, "fixture", "Settings", model.SourceKindDatasourceJSON), "expected datasource JSON target finding for Settings, got %#v", report.Findings)
	require.True(t, hasTarget(report.Findings, "fixture", "Query", model.SourceKindQueryJSON), "expected query JSON target finding for Query, got %#v", report.Findings)
}

func TestRunResolvesDecodeTargetViaSSA(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataSourceInstanceSettings struct {
	JSONData []byte
	DecryptedSecureJSONData map[string]string
}
`,
		"fixture.go": `
package fixture

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Settings struct {
	Name string ` + "`json:\"name\"`" + `
}

func LoadSettings(config backend.DataSourceInstanceSettings) error {
	var out any = &Settings{}
	return json.Unmarshal(config.JSONData, out)
}
`,
	})

	report, err := Run(Config{
		Dir:      dir,
		Patterns: []string{"./..."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.True(t, hasTarget(report.Findings, "fixture", "Settings", model.SourceKindDatasourceJSON), "expected SSA-resolved datasource JSON target finding for Settings, got %#v", report.Findings)
}

func TestRunResolvesSecureKeyPatternViaSSA(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataSourceInstanceSettings struct {
	JSONData []byte
	DecryptedSecureJSONData map[string]string
}
`,
		"fixture.go": `
package fixture

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func secureKey(authID string) string {
	return fmt.Sprintf("auth.%s.apiKey", authID)
}

func LoadSettings(config backend.DataSourceInstanceSettings) string {
	key := secureKey("primary")
	return config.DecryptedSecureJSONData[key]
}
`,
	})

	report, err := Run(Config{
		Dir:      dir,
		Patterns: []string{"./..."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.True(t, hasPattern(report.Findings, "auth.{dynamic}.apiKey"), "expected SSA-resolved secure key pattern, got %#v", report.Findings)
}

func TestRunIgnoresNonStructJSONTargetsAndLocalStringMaps(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataSourceInstanceSettings struct {
	JSONData []byte
	DecryptedSecureJSONData map[string]string
}
`,
		"fixture.go": `
package fixture

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func LoadSettings(config backend.DataSourceInstanceSettings) error {
	jsonData := struct {
		Name string ` + "`json:\"name\"`" + `
	}{}
	if err := json.Unmarshal(config.JSONData, &jsonData); err != nil {
		return err
	}

	settingsMap := make(map[string]any)
	if err := json.Unmarshal(config.JSONData, &settingsMap); err != nil {
		return err
	}

	labels := map[string]string{"name": "value"}
	_ = labels["name"]
	_ = config.DecryptedSecureJSONData["apiKey"]
	return nil
}
`,
	})

	report, err := Run(Config{
		Dir:      dir,
		Patterns: []string{"./..."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.Empty(t, report.Warnings, "expected no warnings")
	require.True(t, hasLiteralKey(report.Findings, "apiKey"), "expected secure key finding, got %#v", report.Findings)
}

func TestRunInfersGuardedReplaceSecureKeyPattern(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataSourceInstanceSettings struct {
	JSONData []byte
	DecryptedSecureJSONData map[string]string
}
`,
		"fixture.go": `
package fixture

import (
	"encoding/json"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func LoadSettings(config backend.DataSourceInstanceSettings) error {
	settingsMap := make(map[string]any)
	if err := json.Unmarshal(config.JSONData, &settingsMap); err != nil {
		return err
	}

	for key := range settingsMap {
		if strings.Contains(key, "httpHeaderName") {
			secureKey := strings.Replace(key, "Name", "Value", -1)
			_ = config.DecryptedSecureJSONData[secureKey]
		}
	}

	return nil
}
`,
	})

	report, err := Run(Config{
		Dir:      dir,
		Patterns: []string{"./..."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.Empty(t, report.Warnings, "expected no warnings")
	require.True(t, hasPattern(report.Findings, "httpHeaderValue{dynamic}"), "expected guarded replace pattern, got %#v", report.Findings)
}

func TestRunInfersGuardedDynamicSecureKeyPattern(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataSourceInstanceSettings struct {
	JSONData []byte
	DecryptedSecureJSONData map[string]string
}
`,
		"fixture.go": `
package fixture

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Setting struct {
	Name string ` + "`json:\"name\"`" + `
	Secure bool ` + "`json:\"secure\"`" + `
}

type Settings struct {
	Settings []*Setting ` + "`json:\"settings\"`" + `
}

func LoadSettings(config backend.DataSourceInstanceSettings) error {
	settings := Settings{}
	if err := json.Unmarshal(config.JSONData, &settings); err != nil {
		return err
	}

	for _, setting := range settings.Settings {
		if setting.Secure {
			_ = config.DecryptedSecureJSONData[setting.Name]
		}
	}

	return nil
}
`,
	})

	report, err := Run(Config{
		Dir:      dir,
		Patterns: []string{"./..."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.Empty(t, report.Warnings, "expected no warnings")
	require.True(t, hasPattern(report.Findings, "{dynamic}"), "expected dynamic secure key pattern, got %#v", report.Findings)
}

func TestRunFindsAliasedAndReencodedQueryTargets(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataQuery struct {
	JSON []byte
}
`,
		"fixture.go": `
package fixture

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Query struct {
	Query string ` + "`json:\"query\"`" + `
}

func LoadQuery(input backend.DataQuery) error {
	queryJSON := string(input.JSON)
	var raw map[string]any
	if err := json.Unmarshal([]byte(queryJSON), &raw); err != nil {
		return err
	}

	modifiedJSON, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	query := Query{}
	return json.Unmarshal(modifiedJSON, &query)
}
`,
	})

	report, err := Run(Config{
		Dir:      dir,
		Patterns: []string{"./..."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.True(t, hasTarget(report.Findings, "fixture", "Query", model.SourceKindQueryJSON), "expected re-encoded query JSON target finding, got %#v", report.Findings)
}

func TestRunInfersLocalWrapperQueryTargetViaSSA(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

import (
	"encoding/json"
	"time"
)

type DataQuery struct {
	JSON json.RawMessage
	TimeRange TimeRange
	Interval time.Duration
}

type TimeRange struct {
	From time.Time
	To time.Time
}

type QueryDataRequest struct {
	Queries []DataQuery
}
`,
		"fixture.go": `
package fixture

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type QueryJSONModel struct {
	Expr string ` + "`json:\"expr\"`" + `
}

func parseQueryModel(raw json.RawMessage) (*QueryJSONModel, error) {
	model := &QueryJSONModel{}
	if err := json.Unmarshal(raw, model); err != nil {
		return nil, err
	}
	return model, nil
}

func parseQuery(req *backend.QueryDataRequest) error {
	for _, query := range req.Queries {
		if _, err := parseQueryModel(query.JSON); err != nil {
			return err
		}
	}
	return nil
}
`,
	})

	report, err := Run(Config{
		Dir:      dir,
		Patterns: []string{"./..."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.True(t, hasTarget(report.Findings, "fixture", "QueryJSONModel", model.SourceKindQueryJSON), "expected local wrapper query target finding, got %#v", report.Findings)
}

func TestRunFindsPointerBackedDecodeTargets(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataSourceInstanceSettings struct {
	JSONData []byte
}
`,
		"fixture.go": `
package fixture

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Config struct {
	Name string ` + "`json:\"name\"`" + `
}

func LoadSettings(settings backend.DataSourceInstanceSettings) error {
	config := &Config{}
	return json.Unmarshal(settings.JSONData, &config)
}
`,
	})

	report, err := Run(Config{
		Dir:      dir,
		Patterns: []string{"./..."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.True(t, hasTarget(report.Findings, "fixture", "Config", model.SourceKindDatasourceJSON), "expected pointer-backed datasource target finding, got %#v", report.Findings)
}

func TestRunFindsSecurePatternsOnMirroredSecureFields(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataSourceInstanceSettings struct {
	DecryptedSecureJSONData map[string]string
}
`,
		"fixture.go": `
package fixture

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Options struct {
	DecryptedSecureJSONData map[string]string
}

func LoadSettings(config backend.DataSourceInstanceSettings) string {
	options := Options{
		DecryptedSecureJSONData: config.DecryptedSecureJSONData,
	}

	return options.DecryptedSecureJSONData[fmt.Sprintf("auth.%s.apiKey", "primary")]
}
`,
	})

	report, err := Run(Config{
		Dir:      dir,
		Patterns: []string{"./..."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.True(t, hasPattern(report.Findings, "auth.{dynamic}.apiKey"), "expected mirrored secure key pattern, got %#v", report.Findings)
}

func TestRunInfersFrameworkQueryTargetViaSSA(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require (
	github.com/grafana/grafana-plugin-sdk-go v0.0.0
	github.com/grafana/sqlds/v5 v5.0.0
)

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
replace github.com/grafana/sqlds/v5 => ./stubs/sqlds
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

import "encoding/json"

type DataSourceInstanceSettings struct{}

type DataQuery struct {
	JSON json.RawMessage
}
`,
		"stubs/grafana-plugin-sdk-go/backend/instancemgmt/instancemgmt.go": `
package instancemgmt

type Instance interface{}
`,
		"stubs/grafana-plugin-sdk-go/data/sqlutil/query.go": `
package sqlutil

import "encoding/json"

type Query struct {
	RawSQL         string          ` + "`json:\"rawSql\"`" + `
	ConnectionArgs json.RawMessage ` + "`json:\"connectionArgs\"`" + `
}
`,
		"stubs/sqlds/go.mod": `
module github.com/grafana/sqlds/v5

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ../grafana-plugin-sdk-go
`,
		"stubs/sqlds/datasource.go": `
package sqlds

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
)

type Driver interface{}

type SQLDatasource struct{}

func NewDatasource(Driver) *SQLDatasource {
	return &SQLDatasource{}
}

func (ds *SQLDatasource) NewDatasource(context.Context, backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return ds, nil
}

type Query = sqlutil.Query

func GetQuery(query backend.DataQuery) (*Query, error) {
	q := &Query{}
	_ = query
	return q, nil
}
`,
		"pkg/plugin/datasource.go": `
package plugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/sqlds/v5"
)

type Driver struct{}

func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	ds := sqlds.NewDatasource(&Driver{})
	return ds.NewDatasource(ctx, settings)
}
`,
	})

	report, err := Run(Config{
		Dir:      dir,
		Patterns: []string{"./..."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.True(t, hasTarget(report.Findings, "github.com/grafana/grafana-plugin-sdk-go/data/sqlutil", "Query", model.SourceKindQueryJSON), "expected SSA-inferred framework query target finding, got %#v", report.Findings)
}

func TestRunInfersDelegatedFrameworkQueryTargetViaSSA(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require (
	github.com/grafana/grafana-plugin-sdk-go v0.0.0
	github.com/grafana/grafana-prometheus-datasource v0.0.0
)

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
replace github.com/grafana/grafana-prometheus-datasource => ./stubs/grafana-prometheus-datasource
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

import "encoding/json"

type DataQuery struct {
	JSON json.RawMessage
}

type QueryDataRequest struct {
	Queries []DataQuery
}

type QueryDataResponse struct{}
`,
		"stubs/grafana-prometheus-datasource/go.mod": `
module github.com/grafana/grafana-prometheus-datasource

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ../grafana-plugin-sdk-go
`,
		"stubs/grafana-prometheus-datasource/pkg/promlib/service.go": `
package promlib

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-prometheus-datasource/pkg/promlib/querydata"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return querydata.Handle(ctx, req)
}
`,
		"stubs/grafana-prometheus-datasource/pkg/promlib/querydata/request.go": `
package querydata

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-prometheus-datasource/pkg/promlib/models"
)

func Handle(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	for _, query := range req.Queries {
		if _, err := models.Parse(query); err != nil {
			return nil, err
		}
	}
	return &backend.QueryDataResponse{}, nil
}
`,
		"stubs/grafana-prometheus-datasource/pkg/promlib/models/query.go": `
package models

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Query struct {
	Expr string ` + "`json:\"expr\"`" + `
}

func Parse(query backend.DataQuery) (*Query, error) {
	model := &Query{}
	if err := json.Unmarshal(query.JSON, model); err != nil {
		return nil, err
	}
	return model, nil
}
`,
		"pkg/prometheus/prometheus.go": `
package prometheus

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-prometheus-datasource/pkg/promlib"
)

type Service struct {
	lib *promlib.Service
}

func ProvideService() *Service {
	return &Service{lib: promlib.NewService()}
}

func (s *Service) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return s.lib.QueryData(ctx, req)
}
`,
	})

	report, err := Run(Config{
		Dir:      dir,
		Patterns: []string{"./..."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.True(t, hasTarget(report.Findings, "github.com/grafana/grafana-prometheus-datasource/pkg/promlib/models", "Query", model.SourceKindQueryJSON), "expected SSA-inferred delegated framework query target finding, got %#v", report.Findings)
}

func TestRunInfersDelegatedFrameworkQueryTargetViaSSAFromSubdirectory(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require (
	github.com/grafana/grafana-plugin-sdk-go v0.0.0
	github.com/grafana/grafana-prometheus-datasource v0.0.0
)

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
replace github.com/grafana/grafana-prometheus-datasource => ./stubs/grafana-prometheus-datasource
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

import "encoding/json"

type DataQuery struct {
	JSON json.RawMessage
}

type QueryDataRequest struct {
	Queries []DataQuery
}

type QueryDataResponse struct{}
`,
		"stubs/grafana-prometheus-datasource/go.mod": `
module github.com/grafana/grafana-prometheus-datasource

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ../grafana-plugin-sdk-go
`,
		"stubs/grafana-prometheus-datasource/pkg/promlib/service.go": `
package promlib

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-prometheus-datasource/pkg/promlib/querydata"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return querydata.Handle(ctx, req)
}
`,
		"stubs/grafana-prometheus-datasource/pkg/promlib/querydata/request.go": `
package querydata

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-prometheus-datasource/pkg/promlib/models"
)

func Handle(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	for _, query := range req.Queries {
		if _, err := models.Parse(query); err != nil {
			return nil, err
		}
	}
	return &backend.QueryDataResponse{}, nil
}
`,
		"stubs/grafana-prometheus-datasource/pkg/promlib/models/query.go": `
package models

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Query struct {
	Expr string
}

type internalQueryModel struct {
	Expr string ` + "`json:\"expr\"`" + `
}

func Parse(query backend.DataQuery) (*Query, error) {
	model := &internalQueryModel{}
	if err := json.Unmarshal(query.JSON, model); err != nil {
		return nil, err
	}
	return &Query{Expr: model.Expr}, nil
}
`,
		"pkg/tsdb/prometheus/prometheus.go": `
package prometheus

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-prometheus-datasource/pkg/promlib"
)

type Service struct {
	lib *promlib.Service
}

func ProvideService() *Service {
	return &Service{lib: promlib.NewService()}
}

func (s *Service) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return s.lib.QueryData(ctx, req)
}
`,
	})

	report, err := Run(Config{
		Dir:      filepath.Join(dir, "pkg/tsdb/prometheus"),
		Patterns: []string{"."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.True(t, hasTarget(report.Findings, "github.com/grafana/grafana-prometheus-datasource/pkg/promlib/models", "internalQueryModel", model.SourceKindQueryJSON), "expected SSA-inferred delegated framework decode target finding from subdirectory, got %#v", report.Findings)
}

func TestRunInfersFrameworkQueryTargetViaSSAThroughCallbackClosure(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.25.5

require (
	github.com/grafana/grafana-plugin-sdk-go v0.0.0
	github.com/grafana/grafana-prometheus-datasource v0.0.0
)

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
replace github.com/grafana/grafana-prometheus-datasource => ./stubs/grafana-prometheus-datasource
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.25.5
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

import "encoding/json"

type DataQuery struct {
	JSON json.RawMessage
}

type QueryDataRequest struct {
	Queries []DataQuery
}

type QueryDataResponse struct{}
`,
		"stubs/grafana-prometheus-datasource/go.mod": `
module github.com/grafana/grafana-prometheus-datasource

go 1.25.5

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ../grafana-plugin-sdk-go
`,
		"stubs/grafana-prometheus-datasource/pkg/promlib/service.go": `
package promlib

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-prometheus-datasource/pkg/promlib/querydata"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return querydata.Handle(ctx, req)
}
`,
		"stubs/grafana-prometheus-datasource/pkg/promlib/querydata/request.go": `
package querydata

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-prometheus-datasource/pkg/promlib/internal/iter"
	"github.com/grafana/grafana-prometheus-datasource/pkg/promlib/models"
)

func Handle(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	err := iter.ForEach(req, func(query backend.DataQuery) error {
		_, err := models.Parse(query)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &backend.QueryDataResponse{}, nil
}
`,
		"stubs/grafana-prometheus-datasource/pkg/promlib/internal/iter/iter.go": `
package iter

import "github.com/grafana/grafana-plugin-sdk-go/backend"

func ForEach(req *backend.QueryDataRequest, fn func(backend.DataQuery) error) error {
	for _, query := range req.Queries {
		if err := fn(query); err != nil {
			return err
		}
	}
	return nil
}
`,
		"stubs/grafana-prometheus-datasource/pkg/promlib/models/query.go": `
package models

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Query struct {
	Expr string
}

type internalQueryModel struct {
	Expr string ` + "`json:\"expr\"`" + `
}

func Parse(query backend.DataQuery) (*Query, error) {
	model := &internalQueryModel{}
	if err := json.Unmarshal(query.JSON, model); err != nil {
		return nil, err
	}
	return &Query{Expr: model.Expr}, nil
}
`,
		"pkg/prometheus/prometheus.go": `
package prometheus

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-prometheus-datasource/pkg/promlib"
)

type Service struct {
	lib *promlib.Service
}

func ProvideService() *Service {
	return &Service{lib: promlib.NewService()}
}

func (s *Service) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return s.lib.QueryData(ctx, req)
}
`,
	})

	report, err := Run(Config{
		Dir:      dir,
		Patterns: []string{"./..."},
		UseSSA:   true,
	})
	require.NoError(t, err, "run failed")
	require.True(t, hasTarget(report.Findings, "github.com/grafana/grafana-prometheus-datasource/pkg/promlib/models", "internalQueryModel", model.SourceKindQueryJSON), "expected SSA-inferred framework decode target finding through callback closure, got %#v", report.Findings)
}

func hasTarget(findings []model.Finding, packagePath string, typeName string, source model.SourceKind) bool {
	for _, finding := range findings {
		if finding.Source != source || finding.Target == nil {
			continue
		}
		if finding.Target.PackagePath == packagePath && finding.Target.TypeName == typeName {
			return true
		}
	}
	return false
}

func hasPattern(findings []model.Finding, pattern string) bool {
	for _, finding := range findings {
		if finding.Pattern == pattern {
			return true
		}
	}
	return false
}

func hasLiteralKey(findings []model.Finding, key string) bool {
	for _, finding := range findings {
		if finding.Source == model.SourceKindDatasourceSecure && finding.Key == key {
			return true
		}
	}
	return false
}
