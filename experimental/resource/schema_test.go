package resource

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

func TestSchemaSupport(t *testing.T) {
	val := JSONSchema{
		Spec: &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "hello",
			},
		},
	}
	jj, err := json.Marshal(val)
	require.NoError(t, err)

	copy := &JSONSchema{}
	err = copy.UnmarshalJSON(jj)
	require.NoError(t, err)
	require.Equal(t, val.Spec.Description, copy.Spec.Description)
}
