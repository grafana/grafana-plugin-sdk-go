package openapigen

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/testutil"
	"github.com/stretchr/testify/require"
)

var expectedGenericSettingsProperties = []string{
	"basicAuth",
	"basicAuthUser",
	"dialTimeout",
	"enableSecureSocksProxy",
	"httpExpectContinueTimeout",
	"httpIdleConnTimeout",
	"httpKeepAlive",
	"httpMaxConnsPerHost",
	"httpMaxIdleConns",
	"httpMaxIdleConnsPerHost",
	"httpTLSHandshakeTimeout",
	"secureSocksProxyUsername",
	"serverName",
	"sigV4AssumeRoleArn",
	"sigV4Auth",
	"sigV4AuthType",
	"sigV4ExternalId",
	"sigV4Profile",
	"sigV4Region",
	"timeout",
	"tlsAuth",
	"tlsAuthWithCACert",
	"tlsSkipVerify",
	"url",
	"user",
}

var expectedGenericSecureValues = []string{
	"basicAuthPassword",
	"httpHeaderValue{dynamic}",
	"password",
	"secureSocksProxyPassword",
	"sigV4AccessKey",
	"sigV4SecretKey",
	"sigV4SessionToken",
	"tlsCACert",
	"tlsClientCert",
	"tlsClientKey",
}

func TestBuildAssemblesExtension(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/models.go": `
package models

type Settings struct {
	Name string ` + "`json:\"name\"`" + `
	Token string
	APIKey string ` + "`json:\"apiKey\"`" + `
	LegacyAPIKey string ` + "`json:\"api_key\"`" + `
}

type Query struct {
	QueryType string ` + "`json:\"queryType\"`" + `
	Owner string ` + "`json:\"owner\"`" + `
}
`,
	})

	result, err := Build(Options{
		Dir: dir,
		Report: model.Report{
			Findings: []model.Finding{
				{
					Source: model.SourceKindDatasourceJSON,
					Target: &model.TargetRef{PackagePath: "fixture/pkg/models", TypeName: "Settings"},
				},
				{
					Source: model.SourceKindDatasourceSecure,
					Key:    "apiKey",
				},
				{
					Source: model.SourceKindDatasourceSecure,
					Key:    "token",
				},
				{
					Source:  model.SourceKindDatasourceSecure,
					Pattern: "auth.{dynamic}.token",
				},
				{
					Source: model.SourceKindQueryJSON,
					Target: &model.TargetRef{PackagePath: "fixture/pkg/models", TypeName: "Query"},
				},
			},
		},
		GenerateSpec:    true,
		GenerateQueries: true,
	})
	require.NoError(t, err, "build failed")
	require.NotNil(t, result.OpenAPI, "expected spec to be generated")
	require.NotNil(t, result.OpenAPI.Settings.Spec, "expected spec to be generated")
	properties := result.OpenAPI.Settings.Spec.Properties
	require.Contains(t, properties, "name", "expected non-secure property to remain")
	require.NotContains(t, properties, "apiKey", "did not expect secure-backed property in spec")
	require.NotContains(t, properties, "api_key", "did not expect normalized secure-backed legacy property in spec")
	require.NotContains(t, properties, "Token", "did not expect case-mismatched secure-backed property in spec")
	require.Len(t, result.OpenAPI.Settings.SecureValues, 3, "expected three secure values")
	require.NotNil(t, result.QueryTypes, "expected query definitions")
	require.Len(t, result.QueryTypes.Items, 1, "expected query definitions")
}

func TestBuildSynthesizesGenericDatasourceSettingsAndSecureValues(t *testing.T) {
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
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

type DataSourceInstanceSettings struct {
	URL string
}

func (s DataSourceInstanceSettings) HTTPClientOptions(context.Context) (httpclient.Options, error) {
	return httpclient.Options{}, nil
}
`,
		"stubs/grafana-plugin-sdk-go/backend/httpclient/options.go": `
package httpclient

type Options struct {
	ForwardHTTPHeaders bool
}
`,
		"plugin/plugin.go": `
package plugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) error {
	_, err := settings.HTTPClientOptions(ctx)
	if err != nil {
		return err
	}
	_ = settings.URL
	return nil
}
`,
	})

	result, err := Build(Options{
		Dir:          dir,
		Report:       model.Report{},
		GenerateSpec: true,
	})
	require.NoError(t, err, "build failed")

	properties := result.OpenAPI.Settings.Spec.Properties
	require.ElementsMatch(t, expectedGenericSettingsProperties, testutil.KeysOfMap(properties), "expected exact generic properties")

	secureNames := map[string]struct{}{}
	for _, value := range result.OpenAPI.Settings.SecureValues {
		secureNames[value.Key] = struct{}{}
	}
	require.ElementsMatch(t, expectedGenericSecureValues, testutil.KeysOfMap(secureNames), "expected exact generic secure values")
}

func TestBuildSynthesizesGenericDatasourceSettingsViaFrameworkUsage(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1

require (
	github.com/example/framework v0.0.0
	github.com/grafana/grafana-plugin-sdk-go v0.0.0
)

replace github.com/example/framework => ./stubs/framework
replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.26.1
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

type DataSourceInstanceSettings struct {
	URL string
}

func (s DataSourceInstanceSettings) HTTPClientOptions(context.Context) (httpclient.Options, error) {
	return httpclient.Options{}, nil
}
`,
		"stubs/grafana-plugin-sdk-go/backend/httpclient/options.go": `
package httpclient

type Options struct {
	ForwardHTTPHeaders bool
}
`,
		"stubs/framework/go.mod": `
module github.com/example/framework

go 1.26.1

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ../grafana-plugin-sdk-go
`,
		"stubs/framework/framework/framework.go": `
package framework

import (
	"context"

	"github.com/example/framework/client"
	"github.com/example/framework/resource"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func NewService() any {
	return newInstanceSettings()
}

func newInstanceSettings() func(context.Context, backend.DataSourceInstanceSettings) error {
	return func(ctx context.Context, settings backend.DataSourceInstanceSettings) error {
		if _, err := client.CreateTransportOptions(ctx, settings); err != nil {
			return err
		}
		resource.UseURL(settings)
		return nil
	}
}
`,
		"stubs/framework/client/client.go": `
package client

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

func CreateTransportOptions(ctx context.Context, settings backend.DataSourceInstanceSettings) (httpclient.Options, error) {
	return settings.HTTPClientOptions(ctx)
}
`,
		"stubs/framework/resource/resource.go": `
package resource

import "github.com/grafana/grafana-plugin-sdk-go/backend"

func UseURL(settings backend.DataSourceInstanceSettings) string {
	return settings.URL
}
`,
		"plugin/plugin.go": `
package plugin

import "github.com/example/framework/framework"

func NewDatasource() any {
	return framework.NewService()
}
`,
	})

	result, err := Build(Options{
		Dir:          dir,
		Report:       model.Report{},
		GenerateSpec: true,
	})
	require.NoError(t, err, "build failed")

	properties := result.OpenAPI.Settings.Spec.Properties
	require.ElementsMatch(t, expectedGenericSettingsProperties, testutil.KeysOfMap(properties), "expected exact framework-derived generic properties")

	secureNames := map[string]struct{}{}
	for _, value := range result.OpenAPI.Settings.SecureValues {
		secureNames[value.Key] = struct{}{}
	}
	require.ElementsMatch(t, expectedGenericSecureValues, testutil.KeysOfMap(secureNames), "expected exact framework-derived generic secure values")
}
