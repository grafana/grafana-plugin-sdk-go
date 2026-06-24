package build

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Schema is a namespace for plugin schema artifact tasks. Plugins inherit
// these targets via the standard `// mage:import` of the SDK's build package
// in their Magefile.
type Schema mg.Namespace

// schemaOutputPath is the conventional location plugins write their schema
// artifact to. Stays under src/ so the frontend build copies it into dist/
// alongside the rest of the plugin's static assets.
const schemaOutputPath = "src/schema/v0alpha1.json"

// Gen regenerates the plugin schema artifact by invoking the plugin's own
// main package in "print-schema" mode. The plugin's app.Manage call sees the
// GF_PLUGIN_PRINT_SCHEMA env var, assembles the artifact from its declared
// Schema, writes it to the path the env var points at, and exits before
// entering the serve loop.
//
// Plugins that declare a typed Schema (StoredObjects, Queries, etc.) on
// their ManageOpts produce content; plugins that don't are a safe no-op.
//
// Convention: the plugin's main package lives at ./pkg.
func (Schema) Gen() error {
	return sh.RunWithV(map[string]string{
		"GF_PLUGIN_PRINT_SCHEMA": schemaOutputPath,
	}, "go", "run", "./pkg")
}
