package schemabuilder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PrintArtifactOpts carries the typed declarations the SDK needs to assemble
// a bundled schema artifact for a plugin running in print-schema mode.
//
// The SDK's app.Manage and datasource.Manage entry points check for the
// GF_PLUGIN_PRINT_SCHEMA environment variable on startup; if it is set, the
// SDK builds the artifact from the plugin's declared ManageOpts.Schema and
// calls PrintBundledArtifact to write it to the path the env var points at.
// The plugin process then exits without entering its serve loop.
type PrintArtifactOpts struct {
	PluginID      string
	APIVersion    string
	ScanCode      []CodePaths
	StoredObjects []StoredObjectInfo
	Queries       []QueryTypeInfo
}

// PrintBundledArtifact assembles the schema artifact for the given options
// and writes it to outPath as a single bundled JSON file. Intended for use
// from the print-schema startup mode in app.Manage / datasource.Manage. The
// parent directory is created if it doesn't exist.
//
// If the plugin declares no typed surfaces (no stored objects, no queries),
// the function returns nil without writing anything. Mage targets running
// this against plugins that don't use the feature are safe no-ops.
func PrintBundledArtifact(outPath string, opts PrintArtifactOpts) error {
	if outPath == "" {
		return fmt.Errorf("output path is empty")
	}
	if opts.PluginID == "" {
		return fmt.Errorf("PluginID is required")
	}
	apiVersion := opts.APIVersion
	if apiVersion == "" {
		apiVersion = "v0alpha1"
	}

	if len(opts.StoredObjects) == 0 && len(opts.Queries) == 0 {
		return nil
	}

	b, err := NewSchemaBuilder(BuilderOptions{
		PluginID: []string{opts.PluginID},
		ScanCode: opts.ScanCode,
	})
	if err != nil {
		return fmt.Errorf("creating schema builder: %w", err)
	}
	if len(opts.Queries) > 0 {
		if err := b.AddQueries(opts.Queries); err != nil {
			return fmt.Errorf("declaring query types: %w", err)
		}
	}
	if len(opts.StoredObjects) > 0 {
		if err := b.AddStoredObjects(opts.StoredObjects); err != nil {
			return fmt.Errorf("declaring stored objects: %w", err)
		}
	}

	artifact, err := b.BuildArtifact(apiVersion)
	if err != nil {
		return fmt.Errorf("building artifact: %w", err)
	}
	raw, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling artifact: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0750); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	if err := os.WriteFile(outPath, raw, 0600); err != nil {
		return fmt.Errorf("writing artifact: %w", err)
	}
	return nil
}
