package build

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateOpenAPIWritesWarningsAndProviderFilenameToPluginRoot(t *testing.T) {
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

type DataSourceInstanceSettings struct {
	JSONData []byte
}
`,
		"plugin/plugin.go": `
package plugin

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Settings struct {
	Name string ` + "`json:\"name\"`" + `
}

type OtherSettings struct {
	Enabled bool ` + "`json:\"enabled\"`" + `
}

func DecodePrimary(cfg backend.DataSourceInstanceSettings) error {
	var settings Settings
	return json.Unmarshal(cfg.JSONData, &settings)
}

func DecodeSecondary(cfg backend.DataSourceInstanceSettings) error {
	var settings OtherSettings
	return json.Unmarshal(cfg.JSONData, &settings)
}
`,
	})

	stderr := captureFile(t, &os.Stderr)

	if err := (Datasource{}).GenerateOpenAPI(dir); err != nil {
		t.Fatalf("generate openapi failed: %v", err)
	}

	errOut := stderr.read()

	if !strings.Contains(errOut, "warning: datasource_multiple_types:") {
		t.Fatalf("expected warning output on stderr, got %q", errOut)
	}

	body, err := os.ReadFile(filepath.Join(dir, openAPIFilename))
	if err != nil {
		t.Fatalf("read generated openapi file failed: %v", err)
	}
	if !strings.Contains(string(body), `"settings"`) {
		t.Fatalf("expected pluginspec JSON in output file, got %q", string(body))
	}
}

func TestGenerateQueryTypesWritesDefinitionsToPluginRoot(t *testing.T) {
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

type DataQuery struct {
	JSON []byte
}
`,
		"plugin/plugin.go": `
package plugin

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Query struct {
	QueryType string ` + "`json:\"queryType\"`" + `
}

func LoadQuery(q backend.DataQuery) error {
	var query Query
	return json.Unmarshal(q.JSON, &query)
}
`,
	})

	stderr := captureFile(t, &os.Stderr)
	if err := (Datasource{}).GenerateQueryTypes(dir); err != nil {
		t.Fatalf("generate query types file failed: %v", err)
	}
	errOut := stderr.read()
	if errOut != "" {
		t.Fatalf("did not expect warning output on stderr, got %q", errOut)
	}

	body, err := os.ReadFile(filepath.Join(dir, queryTypesFilename))
	if err != nil {
		t.Fatalf("read generated query types file failed: %v", err)
	}
	if !strings.Contains(string(body), `"kind": "QueryTypeDefinitionList"`) {
		t.Fatalf("expected query definitions JSON in output file, got %q", string(body))
	}
}

func TestGenerateTargetsDefaultToCurrentDirectory(t *testing.T) {
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

type DataSourceInstanceSettings struct {
	JSONData []byte
}

type DataQuery struct {
	JSON []byte
}
`,
		"plugin/plugin.go": `
package plugin

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Settings struct {
	Name string ` + "`json:\"name\"`" + `
}

type Query struct {
	QueryType string ` + "`json:\"queryType\"`" + `
}

func LoadSettings(cfg backend.DataSourceInstanceSettings) error {
	var settings Settings
	return json.Unmarshal(cfg.JSONData, &settings)
}

func LoadQuery(q backend.DataQuery) error {
	var query Query
	return json.Unmarshal(q.JSON, &query)
}
`,
	})

	stderr := captureFile(t, &os.Stderr)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	if err := (Datasource{}).GenerateOpenAPI(""); err != nil {
		t.Fatalf("generate openapi with current dir failed: %v", err)
	}
	if err := (Datasource{}).GenerateQueryTypes(""); err != nil {
		t.Fatalf("generate query types with current dir failed: %v", err)
	}
	_ = stderr.read()

	for _, name := range []string{openAPIFilename, queryTypesFilename} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s in current directory: %v", name, err)
		}
	}
}

type capturedFile struct {
	t      *testing.T
	target **os.File
	old    *os.File
	reader *os.File
	writer *os.File
}

func captureFile(t *testing.T, target **os.File) *capturedFile {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}

	c := &capturedFile{
		t:      t,
		target: target,
		old:    *target,
		reader: reader,
		writer: writer,
	}

	*target = writer
	t.Cleanup(func() {
		*target = c.old
		_ = c.writer.Close()
		_ = c.reader.Close()
	})

	return c
}

func (c *capturedFile) read() string {
	c.t.Helper()

	*c.target = c.old
	if err := c.writer.Close(); err != nil {
		c.t.Fatalf("close failed: %v", err)
	}

	body, err := io.ReadAll(c.reader)
	if err != nil {
		c.t.Fatalf("read failed: %v", err)
	}

	return string(body)
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
