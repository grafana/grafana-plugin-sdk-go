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

		require.Equal(t, 1, len(fields))
		require.Equal(t, "FrameType", fields[0].Name)
	})

	t.Run("verify enum extraction", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			fields, err := findEnumFields(
				"github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder/example",
				"./example")
			require.NoError(t, err)
			require.Equal(t, 3, len(fields))

			var reduceMode EnumField
			for _, f := range fields {
				if f.Name == "ReduceMode" {
					reduceMode = f
					break
				}
			}
			require.NotNil(t, reduceMode)

			out, err := json.MarshalIndent(reduceMode, "", "  ")
			require.NoError(t, err)
			fmt.Printf("%s", string(out))

			require.JSONEq(t, `{
				"Package": "github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder/example",
				"Name": "ReduceMode",
				"Comment": "Non-Number behavior mode",
				"Values": [
				  {
					"Value": "dropNN",
					"Comment": "Drop non-numbers"
				  },
				  {
					"Value": "replaceNN",
					"Comment": "Replace non-numbers"
				  }
				]
			  } `, string(out))
		}
	})
}
