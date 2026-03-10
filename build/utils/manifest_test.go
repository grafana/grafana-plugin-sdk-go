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
	t.Cleanup(func() { os.Chdir(prev) })
	if err := os.Chdir(dir); err != nil {
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

	os.WriteFile("main.go", []byte("package main"), 0644)
	os.MkdirAll(filepath.Join("node_modules", "somepackage"), 0755)
	os.WriteFile(filepath.Join("node_modules", "somepackage", "file.go"), []byte("package somepackage"), 0644)

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

	os.WriteFile("main.go", []byte("package main"), 0644)

	// Root node_modules — should be excluded.
	os.MkdirAll(filepath.Join("node_modules", "pkg"), 0755)
	os.WriteFile(filepath.Join("node_modules", "pkg", "root.go"), []byte("package pkg"), 0644)

	// Nested node_modules inside a subdirectory — should be included.
	nestedNM := filepath.Join("vendor", "lib", "node_modules", "dep")
	os.MkdirAll(nestedNM, 0755)
	os.WriteFile(filepath.Join(nestedNM, "nested.go"), []byte("package dep"), 0644)

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

	os.WriteFile("main.go", []byte("package main"), 0644)
	os.WriteFile("readme.md", []byte("# readme"), 0644)
	os.WriteFile("config.json", []byte("{}"), 0644)
	os.WriteFile("script.sh", []byte("#!/bin/bash"), 0644)

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

	os.WriteFile("main.go", []byte("package main"), 0644)
	os.MkdirAll(filepath.Join("pkg", "sub"), 0755)
	os.WriteFile(filepath.Join("pkg", "sub", "helper.go"), []byte("package sub"), 0644)

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
	os.WriteFile("main.go", content, 0644)

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

	os.MkdirAll(filepath.Join("a", "b"), 0755)
	os.WriteFile(filepath.Join("a", "b", "c.go"), []byte("package b"), 0644)

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

