package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdir changes to the given directory and registers a cleanup to restore the original.
func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(prev); err != nil {
			t.Logf("failed to restore working directory: %v", err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0750); err != nil {
		t.Fatal(err)
	}
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestGenerateManifest_SkipsNodeModules(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeFile(t, "main.go", []byte("package main"))
	mkdirAll(t, filepath.Join("node_modules", "somepackage"))
	writeFile(t, filepath.Join("node_modules", "somepackage", "file.go"), []byte("package somepackage"))

	manifest, err := GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(manifest, "main.go") {
		t.Error("manifest should contain main.go")
	}
	if strings.Contains(manifest, "node_modules") {
		t.Error("manifest should not contain files from node_modules")
	}
}

func TestGenerateManifest_OnlySkipsRootNodeModules(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeFile(t, "main.go", []byte("package main"))

	// Root node_modules — should be excluded.
	mkdirAll(t, filepath.Join("node_modules", "pkg"))
	writeFile(t, filepath.Join("node_modules", "pkg", "root.go"), []byte("package pkg"))

	// Nested node_modules inside a subdirectory — should be included.
	nestedNM := filepath.Join("vendor", "lib", "node_modules", "dep")
	mkdirAll(t, nestedNM)
	writeFile(t, filepath.Join(nestedNM, "nested.go"), []byte("package dep"))

	manifest, err := GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(manifest, "root.go") {
		t.Error("manifest should not contain Go files from root node_modules")
	}
	if !strings.Contains(manifest, "nested.go") {
		t.Error("manifest should contain Go files from nested node_modules")
	}
}

func TestGenerateManifest_OnlyIncludesGoFiles(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeFile(t, "main.go", []byte("package main"))
	writeFile(t, "readme.md", []byte("# readme"))
	writeFile(t, "config.json", []byte("{}"))
	writeFile(t, "script.sh", []byte("#!/bin/bash"))

	manifest, err := GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(manifest, "main.go") {
		t.Error("manifest should contain main.go")
	}
	for _, name := range []string{"readme.md", "config.json", "script.sh"} {
		if strings.Contains(manifest, name) {
			t.Errorf("manifest should not contain non-Go file %s", name)
		}
	}
}

func TestGenerateManifest_IncludesNestedGoFiles(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeFile(t, "main.go", []byte("package main"))
	mkdirAll(t, filepath.Join("pkg", "sub"))
	writeFile(t, filepath.Join("pkg", "sub", "helper.go"), []byte("package sub"))

	manifest, err := GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(manifest, "main.go") {
		t.Error("manifest should contain main.go")
	}
	if !strings.Contains(manifest, "pkg/sub/helper.go") {
		t.Error("manifest should contain nested Go file pkg/sub/helper.go")
	}
}

func TestGenerateManifest_EntryFormat(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	content := []byte("package main")
	writeFile(t, "main.go", content)

	manifest, err := GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	expectedHash := sha256Hex(content)
	expectedLine := fmt.Sprintf("%s:main.go", expectedHash)

	lines := strings.Split(strings.TrimSpace(manifest), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 manifest line, got %d", len(lines))
	}
	if lines[0] != expectedLine {
		t.Errorf("manifest line = %q, want %q", lines[0], expectedLine)
	}
}

func TestGenerateManifest_UsesForwardSlashes(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	mkdirAll(t, filepath.Join("a", "b"))
	writeFile(t, filepath.Join("a", "b", "c.go"), []byte("package b"))

	manifest, err := GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(manifest, "a/b/c.go") {
		t.Errorf("manifest should use forward slashes, got: %s", manifest)
	}
}

func TestGenerateManifest_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	manifest, err := GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	if manifest != "" {
		t.Errorf("expected empty manifest for directory with no .go files, got: %q", manifest)
	}
}
