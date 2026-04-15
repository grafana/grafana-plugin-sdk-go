package pluginschema

import (
	"github.com/go-openapi/jsonreference"
	"github.com/go-openapi/spec"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"sigs.k8s.io/yaml"
)

// Load yaml or json into a settings object
func Load(jsonOrYaml []byte, obj any) error {
	return yaml.Unmarshal(jsonOrYaml, obj)
}

// Write settings objects as yaml (k8s compatible flavor)
func ToYAML(obj any) ([]byte, error) {
	return yaml.Marshal(obj) // ensure a k8s compatible format
}

// Diff returns a human-readable report of the differences between two settings objects
func Diff(x, y any) string {
	return cmp.Diff(x, y,
		alwaysCompareNumeric,
		cmpopts.EquateApprox(0.001, 0.0001),
		cmp.Comparer(func(a, b spec.Ref) bool {
			return a.String() == b.String()
		}),
		cmp.Comparer(func(a, b jsonreference.Ref) bool {
			return a.String() == b.String()
		}))
}

// alwaysCompareNumeric transforms all ints and floats to float64 for comparison
var alwaysCompareNumeric = cmp.FilterValues(func(x, y any) bool {
	return isNumeric(x) && isNumeric(y)
}, cmp.Transformer("NormalizeNumeric", func(v any) float64 {
	switch t := v.(type) {
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case float64:
		return t
	default:
		return 0 // Should be filtered by isNumeric
	}
}))

func isNumeric(v any) bool {
	switch v.(type) {
	case int, int64, float64:
		return true
	default:
		return false
	}
}
