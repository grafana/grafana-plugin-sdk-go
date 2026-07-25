package grpcplugin

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// TestPluginSet pins the go-plugin entries that pluginSet registers. It focuses
// on the V3 behavior added by this contract: V3 is opt-in (no v3-* entries
// unless V3Server is set) and additive (it is served alongside, not instead of,
// the legacy services), with one entry per V3 service.
func TestPluginSet(t *testing.T) {
	v3Keys := []string{"v3-route", "v3-admission-control", "v3-resource-conversion"}

	t.Run("no V3Server registers no v3 entries", func(t *testing.T) {
		ps := pluginSet(ServeOpts{})
		for _, k := range v3Keys {
			if _, ok := ps[k]; ok {
				t.Errorf("did not expect entry %q", k)
			}
		}
	})

	t.Run("V3Server registers one entry per v3 service", func(t *testing.T) {
		ps := pluginSet(ServeOpts{V3Server: testV3Server{}})
		for _, k := range v3Keys {
			if _, ok := ps[k]; !ok {
				t.Errorf("expected entry %q", k)
			}
		}
	})

	t.Run("V3 is served alongside the legacy services", func(t *testing.T) {
		ps := pluginSet(ServeOpts{
			DataServer: pluginv2.UnimplementedDataServer{},
			V3Server:   testV3Server{},
		})
		for _, k := range append([]string{"data"}, v3Keys...) {
			if _, ok := ps[k]; !ok {
				t.Errorf("expected entry %q", k)
			}
		}
	})
}
