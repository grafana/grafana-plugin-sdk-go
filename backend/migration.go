package backend

import (
	"context"
)

const (
	// EndpointMigration friendly name for the query migration endpoint/handler.
	EndpointQueryMigration Endpoint = "queryMigration"
)

// QueryMigrationHandler is an EXPERIMENTAL service that allows migrating queries
type QueryMigrationHandler interface {
	// MigrateQuery migrates a query request
	MigrateQuery(ctx context.Context, req *QueryMigrationRequest) (*QueryMigrationResponse, error)
}

type MigrateQueryFunc func(context.Context, *QueryMigrationRequest) (*QueryMigrationResponse, error)

type QueryMigrationRequest struct {
	// PluginContext the contextual information for the request.
	PluginContext PluginContext `json:"pluginContext,omitempty"`

	// Queries to migrate.
	Queries []DataQuery `json:"queries"`
}

type QueryMigrationResponse struct {
	// Migrated queries.
	Queries []DataQuery `json:"queries"`
}
