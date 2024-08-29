package v0alpha1

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"k8s.io/kube-openapi/pkg/validation/strfmt"
	"k8s.io/kube-openapi/pkg/validation/validate"
)

func TestOpenAPI(t *testing.T) {
	//nolint:gocritic
	defs := GetOpenAPIDefinitions(func(path string) spec.Ref { //  (unlambda: replace ¯\_(ツ)_/¯)
		return spec.MustCreateRef(path) // placeholder for tests
	})

	def, ok := defs["github.com/grafana/grafana-plugin-sdk-go/backend.DataResponse"]
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

	// Ensure DataSourceRef exists and has three properties
	def, ok = defs["github.com/grafana/grafana-plugin-sdk-go/experimental/apis/data/v0alpha1.DataSourceRef"]
	require.True(t, ok)
	require.ElementsMatch(t, []string{"type", "uid", "apiVersion"}, maps.Keys(def.Schema.Properties))
}
