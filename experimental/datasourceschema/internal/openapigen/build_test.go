package openapigen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
)

func TestBuildAssemblesExtension(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
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
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	if result.OpenAPI == nil || result.OpenAPI.Settings.Spec == nil {
		t.Fatalf("expected spec to be generated")
	}
	properties := result.OpenAPI.Settings.Spec.Properties
	if _, ok := properties["name"]; !ok {
		t.Fatalf("expected non-secure property to remain, got %#v", properties)
	}
	if _, ok := properties["apiKey"]; ok {
		t.Fatalf("did not expect secure-backed property in spec, got %#v", properties)
	}
	if _, ok := properties["api_key"]; ok {
		t.Fatalf("did not expect normalized secure-backed legacy property in spec, got %#v", properties)
	}
	if _, ok := properties["Token"]; ok {
		t.Fatalf("did not expect case-mismatched secure-backed property in spec, got %#v", properties)
	}
	if len(result.OpenAPI.Settings.SecureValues) != 3 {
		t.Fatalf("expected three secure values, got %#v", result.OpenAPI.Settings.SecureValues)
	}
	if result.QueryTypes == nil || len(result.QueryTypes.Items) != 1 {
		t.Fatalf("expected query definitions, got %#v", result.QueryTypes)
	}
}

func TestBuildSynthesizesGenericDatasourceSettingsAndSecureValues(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
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
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	properties := result.OpenAPI.Settings.Spec.Properties
	for _, key := range []string{"url", "user", "basicAuth", "basicAuthUser", "timeout", "tlsAuth", "tlsAuthWithCACert", "tlsSkipVerify", "serverName", "enableSecureSocksProxy"} {
		if _, ok := properties[key]; !ok {
			t.Fatalf("expected generic property %q, got %#v", key, properties)
		}
	}

	secureNames := map[string]struct{}{}
	for _, value := range result.OpenAPI.Settings.SecureValues {
		secureNames[value.Key] = struct{}{}
	}
	for _, key := range []string{"basicAuthPassword", "password", "tlsCACert", "tlsClientCert", "tlsClientKey", "secureSocksProxyPassword", "httpHeaderValue{dynamic}"} {
		if _, ok := secureNames[key]; !ok {
			t.Fatalf("expected generic secure value %q, got %#v", key, result.OpenAPI.Settings.SecureValues)
		}
	}
}

func TestBuildSynthesizesGenericDatasourceSettingsViaFrameworkUsage(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
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
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	properties := result.OpenAPI.Settings.Spec.Properties
	for _, key := range []string{"url", "user", "basicAuth", "basicAuthUser", "timeout", "tlsAuth"} {
		if _, ok := properties[key]; !ok {
			t.Fatalf("expected framework-derived generic property %q, got %#v", key, properties)
		}
	}

	secureNames := map[string]struct{}{}
	for _, value := range result.OpenAPI.Settings.SecureValues {
		secureNames[value.Key] = struct{}{}
	}
	for _, key := range []string{"basicAuthPassword", "password", "tlsCACert", "tlsClientCert", "tlsClientKey"} {
		if _, ok := secureNames[key]; !ok {
			t.Fatalf("expected framework-derived generic secure value %q, got %#v", key, result.OpenAPI.Settings.SecureValues)
		}
	}
}

func writeFixtureModule(t *testing.T, files map[string]string) string {
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
