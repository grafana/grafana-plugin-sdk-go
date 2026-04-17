package v0alpha1

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

func TestSchemaSupport(t *testing.T) {
	val := JSONSchema{
		Spec: &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "hello",
				Schema:      draft04,
				ID:          "something",
			},
		},
	}
	jj, err := json.MarshalIndent(val, "", "")
	require.NoError(t, err)

	fmt.Printf("%s\n", string(jj))

	cpy := &JSONSchema{}
	err = cpy.UnmarshalJSON(jj)
	require.NoError(t, err)
	require.Equal(t, val.Spec.Description, cpy.Spec.Description)
}
