package sqlutil_test

import (
	"database/sql"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestMakeScanRow(t *testing.T) {
	t.Run("Should return converters for every column type", func(t *testing.T) {
		colTypes := make([]*sql.ColumnType, 32)
	})
}
