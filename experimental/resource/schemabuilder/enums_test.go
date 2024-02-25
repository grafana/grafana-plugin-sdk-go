package schemabuilder

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindEnums(t *testing.T) {
	fields, err := findEnumFields(
		"github.com/grafana/grafana-plugin-sdk-go/experimental/resource/schemabuilder",
		"./example")
	require.NoError(t, err)

	out, err := json.MarshalIndent(fields, "", "  ")
	require.NoError(t, err)
	fmt.Printf("%s", string(out))

	require.Equal(t, 3, len(fields))
}
