package marketplace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLicensePluginID(t *testing.T) {
	tests := []struct {
		name     string
		opts     Opts
		expected string
	}{
		{"no parent uses own ID", Opts{}, "child-plugin"},
		{"parent overrides own ID", Opts{ParentPluginID: "parent-app"}, "parent-app"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.opts.licensePluginID("child-plugin"))
		})
	}
}

func TestInvalidLicenseError(t *testing.T) {
	for _, id := range []string{"parent-app", "own-plugin"} {
		t.Run(id, func(t *testing.T) {
			msg := invalidLicenseError(id).Error()
			require.Contains(t, msg, "The Marketplace plugin "+id+" is not")
			require.Contains(t, msg, "https://grafana.com/grafana/plugins/"+id)
		})
	}
}
