package app

import (
	"fmt"
	"log"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder"
	"github.com/grafana/grafana-plugin-sdk-go/internal/automanagement"
	"github.com/grafana/grafana-plugin-sdk-go/internal/buildinfo"
)

// envPrintSchemaPath is the environment variable the SDK checks on startup
// to decide whether the plugin should dump its schema artifact and exit
// instead of entering the normal serve loop. The mage Schema:Gen target
// sets this; plugin authors don't see it.
const envPrintSchemaPath = "GF_PLUGIN_PRINT_SCHEMA"

// Schema groups a plugin's typed declarations for the schema artifact and
// runtime admission. When set on ManageOpts, the SDK:
//
//   - On normal startup, auto-derives an AdmissionHandler from declared
//     stored objects (unless ManageOpts.AdmissionHandler is also set, in
//     which case the explicit handler wins).
//   - On startup with GF_PLUGIN_PRINT_SCHEMA set, assembles the schema
//     artifact JSON from these declarations and writes it to the path the
//     env var points at, then exits before the serve loop runs.
//
// EXPERIMENTAL: shape may change as additional typed surfaces (settings,
// routes) gain reflection support.
type Schema struct {
	// APIVersion the artifact targets. Defaults to "v0alpha1".
	APIVersion string

	// ScanCode points at Go packages whose comments should be attached to
	// reflected schemas (matches the schemabuilder ScanCode option).
	ScanCode []schemabuilder.CodePaths

	// StoredObjects the plugin declares.
	StoredObjects []schemabuilder.StoredObjectInfo

	// Queries the plugin declares (datasource plugins).
	Queries []schemabuilder.QueryTypeInfo
}

// ManageOpts can modify Manage behavior.
type ManageOpts struct {
	// GRPCSettings settings for gPRC.
	GRPCSettings backend.GRPCSettings

	// TracingOpts contains settings for tracing setup.
	TracingOpts tracing.Opts

	// Stateless admission handler. If nil and Schema declares stored
	// objects, the SDK derives one automatically from Schema.StoredObjects.
	AdmissionHandler backend.AdmissionHandler

	// Stateless conversion handler
	ConversionHandler backend.ConversionHandler

	// Stateless stored object event handler. If nil and Schema declares
	// stored objects with Events, the SDK wires a default handler that feeds
	// the experimental/storedobjects event broker, so instances can consume
	// changes through Collection.Watch.
	StoredObjectEventHandler backend.StoredObjectEventHandler

	// Schema carries the plugin's typed declarations. Optional. When set,
	// the SDK uses it for both build-time schema artifact generation (via
	// GF_PLUGIN_PRINT_SCHEMA) and runtime admission auto-derivation.
	Schema *Schema
}

// Manage starts serving the app over gPRC with automatic instance management.
// pluginID should match the one from plugin.json.
func Manage(pluginID string, instanceFactory InstanceFactoryFunc, opts ManageOpts) error {
	// If we are running in build info mode, run that and exit
	if buildinfo.InfoModeEnabled() {
		if err := buildinfo.RunInfoMode(); err != nil {
			log.Fatalln(err)
			return err
		}
		os.Exit(0)
		return nil
	}

	// If we are running in print-schema mode, dump the artifact and exit
	// before any plugin process state is built up. Setup of the gRPC
	// server, tracing, and instance management is skipped.
	if path := os.Getenv(envPrintSchemaPath); path != "" {
		if err := printSchemaForOpts(pluginID, path, opts); err != nil {
			return err
		}
		os.Exit(0)
		return nil
	}

	// Auto-derive an admission handler from declared stored objects if the
	// caller didn't provide an explicit one. An explicit handler always
	// wins (lets plugins drop down to custom dispatch when needed).
	if opts.AdmissionHandler == nil && opts.Schema != nil && len(opts.Schema.StoredObjects) > 0 {
		opts.AdmissionHandler = admissionHandlerFromStoredObjects(opts.Schema.StoredObjects)
	}

	// Auto-wire the broker-backed event handler when any declared stored
	// object opts into events. An explicit handler always wins (same
	// precedence as AdmissionHandler).
	if opts.StoredObjectEventHandler == nil && opts.Schema != nil && anyStoredObjectDeclaresEvents(opts.Schema.StoredObjects) {
		opts.StoredObjectEventHandler = brokerStoredObjectEventHandler{}
	}

	backend.SetupPluginEnvironment(pluginID)
	if err := backend.SetupTracer(pluginID, opts.TracingOpts); err != nil {
		return fmt.Errorf("setup tracer: %w", err)
	}
	handler := automanagement.NewManager(NewInstanceManager(instanceFactory))
	return backend.Manage(pluginID, backend.ServeOpts{
		CheckHealthHandler:       handler,
		CallResourceHandler:      handler,
		QueryDataHandler:         handler,
		QueryChunkedDataHandler:  handler,
		StreamHandler:            handler,
		AdmissionHandler:         opts.AdmissionHandler,
		ConversionHandler:        opts.ConversionHandler,
		StoredObjectEventHandler: opts.StoredObjectEventHandler,
		GRPCSettings:             opts.GRPCSettings,
	})
}

// printSchemaForOpts assembles the schema artifact from opts.Schema and
// writes it to outPath. Returns nil (no-op) if no Schema is declared, so
// plugins that don't use the feature can safely be invoked in print-schema
// mode without erroring.
func printSchemaForOpts(pluginID, outPath string, opts ManageOpts) error {
	if opts.Schema == nil {
		return nil
	}
	return schemabuilder.PrintBundledArtifact(outPath, schemabuilder.PrintArtifactOpts{
		PluginID:      pluginID,
		APIVersion:    opts.Schema.APIVersion,
		ScanCode:      opts.Schema.ScanCode,
		StoredObjects: opts.Schema.StoredObjects,
		Queries:       opts.Schema.Queries,
	})
}

// admissionHandlerFromStoredObjects translates declared stored objects into
// the AdmissionEntry shape the schemabuilder dispatcher takes. Kept private
// to avoid plugin authors reaching for it directly; the entry point is
// declaring Schema.StoredObjects in ManageOpts.
func admissionHandlerFromStoredObjects(stored []schemabuilder.StoredObjectInfo) backend.AdmissionHandler {
	entries := make([]schemabuilder.AdmissionEntry, 0, len(stored))
	for _, s := range stored {
		entries = append(entries, schemabuilder.AdmissionEntry{
			Kind:     s.Name,
			SpecType: s.SpecType,
		})
	}
	return schemabuilder.AdmissionHandler(entries...)
}
