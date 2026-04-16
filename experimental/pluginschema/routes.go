package pluginschema

import (
	"fmt"
	"strings"

	"k8s.io/kube-openapi/pkg/spec3"
)

// Holds the OpenAPI routes required for /resources and /proxy
type Routes struct {
	// Routes added below the configured plugin
	Paths map[string]*spec3.Path `json:"paths"`

	// Components includes additional re-usable elements that can be referenced in the full spec
	Components *spec3.Components `json:"components,omitempty"`
}

func (r *Routes) Register(path string, props spec3.PathProps) {
	if r.Paths == nil {
		r.Paths = make(map[string]*spec3.Path)
	}
	r.Paths[path] = &spec3.Path{PathProps: props}
}

// Make sure the paths start with registered values
func (r *Routes) AssertPrefixes(prefix ...string) error {
	for k := range r.Paths {
		found := false
		for _, p := range prefix {
			if strings.HasPrefix(k, p) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid path: %s, must start with: %v", k, prefix)
		}
	}
	return nil
}
