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
