package v0alpha1

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"k8s.io/kube-openapi/pkg/validation/strfmt"
	"k8s.io/kube-openapi/pkg/validation/validate"
)

func TestOpenAPI(t *testing.T) {
	defs := GetOpenAPIDefinitions(func(path string) spec.Ref {
		return spec.MustCreateRef(path)
	})

	def, ok := defs["github.com/grafana/grafana-plugin-sdk-go/backend.QueryDataResponse"]
	require.True(t, ok)
	require.Empty(t, def.Dependencies) // not yet supported!

	validator := validate.NewSchemaValidator(&def.Schema, nil, "data", strfmt.Default)

	body, err := os.ReadFile("./testdata/sample_query_results.json")
	require.NoError(t, err)
	unstructured := make(map[string]any)
	err = json.Unmarshal(body, &unstructured)
	require.NoError(t, err)

	result := validator.Validate(unstructured)
	for _, err := range result.Errors {
		assert.NoError(t, err, "validation error")
	}
	for _, err := range result.Warnings {
		assert.NoError(t, err, "validation warning")
	}
}
