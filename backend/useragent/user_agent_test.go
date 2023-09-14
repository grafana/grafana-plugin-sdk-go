package useragent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromString(t *testing.T) {
	tcs := []struct {
		name      string
		userAgent string
		expected  *UserAgent
		err       error
	}{
		{
			name:      "valid",
			userAgent: "Grafana/7.0.0-beta1 (darwin; amd64)",
			expected: &UserAgent{
				grafanaVersion: "7.0.0-beta1",
				arch:           "amd64",
				os:             "darwin",
			},
		},
		{
			name:      "invalid (missing os + arch)",
			userAgent: "Grafana/7.0.0-beta1",
			err:       errInvalidFormat,
		},
		{
			name:      "invalid (missing arch)",
			userAgent: "Grafana/7.0.0-beta1 (darwin)",
			err:       errInvalidFormat,
		},
		{
			name:      "invalid (missing os)",
			userAgent: "Grafana/7.0.0-beta1 (; amd64)",
			err:       errInvalidFormat,
		},
		{
			name:      "invalid (missing semicolon)",
			userAgent: "Grafana/7.0.0-beta1 (darwin amd64)",
			err:       errInvalidFormat,
		},
		{
			name:      "invalid (not semver)",
			userAgent: "Grafana/10.0 (darwin; amd64)",
			err:       errInvalidFormat,
		},
		{
			name:      "invalid (extra param)",
			userAgent: "Grafana/7.0.0-beta1 (darwin; amd64; linux)",
			err:       errInvalidFormat,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			res, err := NewFromString(tc.userAgent)
			require.ErrorIs(t, err, tc.err)
			require.Equalf(t, tc.expected, res, "NewFromString(%v)", tc.userAgent)
		})
	}
}
