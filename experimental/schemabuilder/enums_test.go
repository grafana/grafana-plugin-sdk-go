package schemabuilder

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindEnums(t *testing.T) {
	t.Run("data", func(t *testing.T) {
		fields, err := findEnumFields(
			"github.com/grafana/grafana-plugin-sdk-go/data",
			"../../data")
		require.NoError(t, err)

		out, err := json.MarshalIndent(fields, "", "  ")
		require.NoError(t, err)
		fmt.Printf("%s", string(out))

		require.Equal(t, 1, len(fields))
	})

	t.Run("example", func(t *testing.T) {
		fields, err := findEnumFields(
			"github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder/example",
			"./example")
		require.NoError(t, err)

		out, err := json.MarshalIndent(fields, "", "  ")
		require.NoError(t, err)
		fmt.Printf("%s", string(out))

		require.Equal(t, 3, len(fields))
	})
}
