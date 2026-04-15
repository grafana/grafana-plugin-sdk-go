package pluginschema

import (
	"fmt"
	"strings"

	"k8s.io/kube-openapi/pkg/spec3"
)

// Holds the OpenAPI routes required for /resources and /proxy
type RouteOpenAPI struct {
	spec3.OpenAPI
}

func (r *RouteOpenAPI) Register(path string, props spec3.PathProps) {
	if r.Paths == nil {
		r.Paths = &spec3.Paths{}
	}
	if r.Paths.Paths == nil {
		r.Paths.Paths = make(map[string]*spec3.Path)
	}
	r.Paths.Paths[path] = &spec3.Path{PathProps: props}
}

// Make sure the paths start with registered values
func (r *RouteOpenAPI) AssertPrefixes(prefix ...string) error {
	for k := range r.Paths.Paths {
		for _, p := range prefix {
			if strings.HasPrefix(k, p) {
				continue
			}
			return fmt.Errorf("invalid path: %s, must start with [%v]", k, prefix)
		}
	}
	return nil
}
