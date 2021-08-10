package datasource_test

import (
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/stretchr/testify/assert"
)

func TestMigrator_Up(t *testing.T) {
	versions := []string{"0.2.0", "1.0.0", "1.2.0", "2.0.0"}
	migrator := genMigrations(versions)
	t.Run("it should migrate up", func(t *testing.T) {
		got, err := migrator.Up(datasource.DataSourceQuery, "0.1.0", "2.0.0", genData([]string{"0.1.0"}))
		assert.Nil(t, err)
		assert.Equal(t, []string{"0.1.0", "0.2.0", "1.0.0", "1.2.0", "2.0.0"}, got)
	})

	t.Run("it should not re-migrate up", func(t *testing.T) {
		got, err := migrator.Up(datasource.DataSourceQuery, "0.2.0", "2.0.0", genData([]string{"0.2.0"}))
		assert.Nil(t, err)
		assert.Equal(t, genData(versions), got)
		assert.Equal(t, []string{"0.2.0", "1.0.0", "1.2.0", "2.0.0"}, got)
	})

	t.Run("it should not migrate beyond next version", func(t *testing.T) {
		got, err := migrator.Up(datasource.DataSourceQuery, "0.2.0", "1.0.0", genData([]string{"0.2.0"}))
		assert.Nil(t, err)
		assert.Equal(t, genData([]string{"0.2.0", "1.0.0"}), got)
	})

	t.Run("it should handle a next version outside the range", func(t *testing.T) {
		got, err := migrator.Up(datasource.DataSourceQuery, "2.0.0", "3.0.0", genData([]string{"2.0.0"}))
		assert.Nil(t, err)
		assert.Equal(t, genData(versions), got)
		assert.Equal(t, genData([]string{"2.0.0"}), got)
	})
}

func TestMigrator_Down(t *testing.T) {
	versions := []string{"0.2.0", "1.0.0", "1.2.0", "2.0.0"}
	migrator := genMigrations(versions)
	t.Run("it should migrate down", func(t *testing.T) {
		got, err := migrator.Down(datasource.DataSourceQuery, "2.1.0", "0.2.0", genData([]string{"2.1.0"}))
		assert.Nil(t, err)
		assert.Equal(t, genData([]string{"2.1.0", "2.0.0", "1.2.0", "1.0.0", "0.2.0"}), got)
	})

	t.Run("it should not re-migrate down", func(t *testing.T) {
		got, err := migrator.Down(datasource.DataSourceQuery, "2.0.0", "0.2.0", genData([]string{"2.0.0"}))
		assert.Nil(t, err)
		assert.Equal(t, genData([]string{"2.0.0", "1.2.0", "1.0.0", "0.2.0"}), got)
	})

	t.Run("it should not migrate beyond the next version", func(t *testing.T) {
		got, err := migrator.Down(datasource.DataSourceQuery, "2.0.0", "1.2.0", genData([]string{"2.0.0"}))
		assert.Nil(t, err)
		assert.Equal(t, genData([]string{"2.0.0", "1.2.0"}), got)
	})

	t.Run("it should handle a next version outside of the range", func(t *testing.T) {
		got, err := migrator.Down(datasource.DataSourceQuery, "0.3.0", "0.1.0", genData([]string{"0.3.0"}))
		assert.Nil(t, err)
		assert.Equal(t, genData([]string{"0.3.0", "0.2.0"}), got)
	})
}

func genData(versions []string) json.RawMessage {
	data, _ := json.Marshal(versions)
	return data
}

func genMigrations(versions []string) *datasource.Migrator {
	migrator := datasource.NewMigrator()
	for _, version := range versions {
		migrator.Register(&datasource.Migration{
			Type:    datasource.DataSourceQuery,
			Version: version,
			Up:      migrate(version),
			Down:    migrate(version),
		})
	}
	return migrator
}

func migrate(version string) datasource.MigrationFunc {
	return func(d json.RawMessage) (json.RawMessage, error) {
		var data []string
		err := json.Unmarshal(d, &data)
		if err != nil {
			return nil, err
		}
		data = append(data, version)
		return json.Marshal(data)
	}
}
