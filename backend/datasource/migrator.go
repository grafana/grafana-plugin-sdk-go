package datasource

import (
	"encoding/json"
	"sort"

	"github.com/Masterminds/semver/v3"
)

type MigrationType int

const (
	DataSourceQuery MigrationType = iota
)

func (mt MigrationType) String() string {
	switch mt {
	case 0:
		return "DataSourceQuery"
	default:
		panic("datasource: unknown migration type")

	}
}

type MigrationFunc func(d json.RawMessage) (json.RawMessage, error)
type MigrationsMap map[string]map[MigrationType]*Migration

type Migration struct {
	Type    MigrationType
	Version string
	Up      MigrationFunc
	Down    MigrationFunc
}

type Migrator struct {
	migrations MigrationsMap
}

// Migrator allocates and returns a new Migrator
func NewMigrator() *Migrator {
	return &Migrator{
		migrations: make(MigrationsMap),
	}
}

// GetMigration returns a migration for the matching migration type and version.
func (m *Migrator) GetMigration(mt MigrationType, version string) *Migration {
	v, ok := m.migrations[version]
	if !ok {
		return nil
	}

	return v[mt]
}

// Up runs the up migrations on the provided data until it reaches the expected version.
func (m *Migrator) Up(mt MigrationType, current, next string, data json.RawMessage) (json.RawMessage, error) {
	var (
		versions = m.getVersions()
		migrated = data
		err      error
	)

	sort.Sort(versions)

	currentVersion, err := semver.StrictNewVersion(current)
	if err != nil {
		return nil, err
	}

	nextVersion, err := semver.StrictNewVersion(next)
	if err != nil {
		return nil, err
	}

	for _, v := range versions {
		// we can skip this version
		if currentVersion.GreaterThan(v) || currentVersion.Equal(v) {
			continue
		}

		// make sure we don't migrate beyond the expected new version
		if nextVersion.LessThan(v) {
			return migrated, nil
		}

		migration := m.GetMigration(mt, v.String())
		// there is no migration for the requested migration type
		if migration == nil {
			continue
		}

		migrated, err = migration.Up(migrated)

		if err != nil {
			return nil, err
		}

	}

	return migrated, nil
}

// Down runs the down migrations on the provided data until it reaches the expected version.
func (m *Migrator) Down(mt MigrationType, current, next string, data json.RawMessage) (json.RawMessage, error) {
	var (
		versions                 = m.getVersions()
		migrated json.RawMessage = data
		err      error
	)

	sort.Sort(sort.Reverse(versions))

	currentVersion, err := semver.StrictNewVersion(current)
	if err != nil {
		return nil, err
	}

	nextVersion, err := semver.StrictNewVersion(next)
	if err != nil {
		return nil, err
	}

	for _, v := range versions {
		// we can skip this version
		if currentVersion.LessThan(v) || currentVersion.Equal(v) {
			continue
		}

		// make sure we don't migrate beyond the expected new version
		if nextVersion.GreaterThan(v) {
			return migrated, nil
		}

		migration := m.GetMigration(mt, v.String())
		// there is no migration for the requested migration type
		if migration == nil {
			continue
		}

		migrated, err = migration.Down(migrated)

		if err != nil {
			return nil, err
		}

	}

	return migrated, nil
}

// Register adds a new migration to the list of migrations.
func (m *Migrator) Register(migration *Migration) {
	if migration.Up == nil {
		panic("datasource: nil up migration handler")
	}

	if migration.Down == nil {
		panic("datasource: nil down migration handler")
	}

	if _, err := semver.NewVersion(migration.Version); err != nil {
		panic("datasource: invalid migration version: " + migration.Version)
	}

	typeMap, ok := m.migrations[migration.Version]
	if !ok {
		typeMap = make(map[MigrationType]*Migration)
	}

	if _, exist := typeMap[migration.Type]; exist {
		panic("datasource: multiple registrations for " + migration.Version + " " + migration.Type.String())
	}

	typeMap[migration.Type] = migration
	m.migrations[migration.Version] = typeMap
}

func (m *Migrator) getVersions() semver.Collection {
	vs := make([]*semver.Version, 0)
	for version := range m.migrations {
		v, err := semver.NewVersion(version)
		if err != nil {
			panic("datasource: invalid migration version: " + version)
		}

		vs = append(vs, v)
	}

	return semver.Collection(vs)
}
