package build

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testBuildDirPerm  = 0o750
	testBuildFilePerm = 0o600
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

	if err := (Datasource{}).GenerateOpenAPI(stringPtr(dir)); err != nil {
		t.Fatalf("generate openapi failed: %v", err)
	}

	errOut := stderr.read()

	if !strings.Contains(errOut, "warning: datasource_multiple_types:") {
		t.Fatalf("expected warning output on stderr, got %q", errOut)
	}

	//nolint:gosec // test reads a generated file at a fixed filename within the temp fixture directory.
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
	if err := (Datasource{}).GenerateQueryTypes(stringPtr(dir)); err != nil {
		t.Fatalf("generate query types file failed: %v", err)
	}
	errOut := stderr.read()
	if errOut != "" {
		t.Fatalf("did not expect warning output on stderr, got %q", errOut)
	}

	//nolint:gosec // test reads a generated file at a fixed filename within the temp fixture directory.
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

	if err := (Datasource{}).GenerateOpenAPI(nil); err != nil {
		t.Fatalf("generate openapi with current dir failed: %v", err)
	}
	if err := (Datasource{}).GenerateQueryTypes(nil); err != nil {
		t.Fatalf("generate query types with current dir failed: %v", err)
	}
	_ = stderr.read()

	for _, name := range []string{openAPIFilename, queryTypesFilename} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s in current directory: %v", name, err)
		}
	}
}

func TestMageTargetsDefaultToCurrentDirectory(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}

	repoRoot := filepath.Dir(wd)
	dir := writeFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1

require (
	github.com/grafana/grafana-plugin-sdk-go v0.0.0
	github.com/magefile/mage v1.17.0
)

replace github.com/grafana/grafana-plugin-sdk-go => ` + repoRoot + `
`,
		"Magefile.go": `
//go:build mage

package main

import (
	// mage:import
	build "github.com/grafana/grafana-plugin-sdk-go/build"
)

var Default = build.BuildAll
`,
		"pkg/plugin.go": `
package pkg

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

	runGoCommand(t, dir, "mod", "tidy")
	mageBin := buildMageBinary(t)
	runCommand(t, dir, mageBin, "datasource:generateOpenAPI")
	runCommand(t, dir, mageBin, "datasource:generateQueryTypes")

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
		if err := os.MkdirAll(filepath.Dir(fullPath), testBuildDirPerm); err != nil {
			t.Fatalf("mkdir failed for %s: %v", fullPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(strings.TrimLeft(content, "\n")), testBuildFilePerm); err != nil {
			t.Fatalf("write failed for %s: %v", fullPath, err)
		}
	}

	return dir
}

func runGoCommand(t *testing.T, dir string, args ...string) string {
	t.Helper()

	return runCommand(t, dir, "go", args...)
}

func buildMageBinary(t *testing.T) string {
	t.Helper()

	bin := filepath.Join(t.TempDir(), "mage")
	runGoCommand(t, "", "build", "-o", bin, "github.com/magefile/mage")
	return bin
}

func stringPtr(s string) *string {
	return &s
}

func runCommand(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()

	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "MAGEFILE_CACHE="+t.TempDir())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %s failed: %v\nstderr:\n%s", name, strings.Join(args, " "), err, stderr.String())
	}

	return string(out)
}
