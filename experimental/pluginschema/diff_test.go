package pluginschema

import (
	"testing"

	"github.com/go-openapi/jsonreference"
	"github.com/google/go-cmp/cmp"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		name          string
		x             any
		y             any
		expectEmpty   bool
		expectContent string
	}{
		{
			name:        "both nil",
			x:           nil,
			y:           nil,
			expectEmpty: true,
		},
		{
			name:          "x nil y not nil",
			x:             nil,
			y:             "value",
			expectContent: "y is a new value",
		},
		{
			name:          "x not nil y nil",
			x:             "value",
			y:             nil,
			expectContent: "y does not exist",
		},
		{
			name:        "identical strings",
			x:           "hello",
			y:           "hello",
			expectEmpty: true,
		},
		{
			name:        "different strings",
			x:           "hello",
			y:           "world",
			expectEmpty: false,
		},
		{
			name:        "identical ints",
			x:           42,
			y:           42,
			expectEmpty: true,
		},
		{
			name:        "different ints",
			x:           42,
			y:           43,
			expectEmpty: false,
		},
		{
			name:        "int and int64 same value",
			x:           42,
			y:           int64(42),
			expectEmpty: true,
		},
		{
			name:        "int and float64 same value",
			x:           42,
			y:           42.0,
			expectEmpty: true,
		},
		{
			name:        "same refs",
			x:           spec.MustCreateRef("#a"),
			y:           spec.MustCreateRef("#a"),
			expectEmpty: true,
		},
		{
			name:        "same jsonreference",
			x:           jsonreference.MustCreateRef("#a"),
			y:           jsonreference.MustCreateRef("#a"),
			expectEmpty: true,
		},
		{
			name:        "identical maps",
			x:           map[string]int{"a": 1, "b": 2},
			y:           map[string]int{"a": 1, "b": 2},
			expectEmpty: true,
		},
		{
			name:        "different maps",
			x:           map[string]int{"a": 1},
			y:           map[string]int{"a": 2},
			expectEmpty: false,
		},
		{
			name:        "identical slices",
			x:           []string{"a", "b"},
			y:           []string{"a", "b"},
			expectEmpty: true,
		},
		{
			name:        "different slices",
			x:           []string{"a", "b"},
			y:           []string{"a", "c"},
			expectEmpty: false,
		},
		{
			name:        "identical structs",
			x:           struct{ Name string }{Name: "test"},
			y:           struct{ Name string }{Name: "test"},
			expectEmpty: true,
		},
		{
			name:        "different structs",
			x:           struct{ Name string }{Name: "test1"},
			y:           struct{ Name string }{Name: "test2"},
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Diff(tt.x, tt.y)
			if tt.expectEmpty && result != "" {
				t.Errorf("expected empty diff, got: %q", result)
			}
			if !tt.expectEmpty && result == "" {
				t.Errorf("expected non-empty diff, got empty")
			}
			if tt.expectContent != "" && !cmp.Equal(result, tt.expectContent) {
				if !contains(result, tt.expectContent) {
					t.Errorf("expected diff to contain %q, got: %q", tt.expectContent, result)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0)
}
