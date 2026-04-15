package pluginschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/kube-openapi/pkg/spec3"
)

func TestAssertPrefixes(t *testing.T) {
	tests := []struct {
		name    string
		paths   map[string]*spec3.Path
		wantErr string
	}{
		{
			name: "valid paths",
			paths: map[string]*spec3.Path{
				"/resources/test":  {},
				"/resources/other": {},
				"/proxy/xyz":       {},
			},
		},
		{
			name: "invalid path",
			paths: map[string]*spec3.Path{
				"/invalid/path": {},
			},
			wantErr: "invalid path: /invalid/path",
		},
		{
			name: "invalid resource path",
			paths: map[string]*spec3.Path{
				"/resource": {},
			},
			wantErr: "invalid path: /resource",
		},
		{
			name:  "empty paths",
			paths: map[string]*spec3.Path{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Routes{Paths: tt.paths}
			err := r.AssertPrefixes("/resources", "/proxy")
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
