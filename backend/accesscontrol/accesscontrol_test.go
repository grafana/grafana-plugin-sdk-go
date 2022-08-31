package accesscontrol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_WildcardsFromPrefixes(t *testing.T) {
	tests := []struct {
		name     string
		prefixes []string
		want     Wildcards
	}{
		{
			name:     "no prefix",
			prefixes: []string{},
			want:     []string{},
		},
		{
			name:     "empty prefix",
			prefixes: []string{""},
			want:     []string{"*"},
		},
		{
			name:     "one prefix",
			prefixes: []string{"plugin_id.a:uid:"},
			want:     []string{"*", "plugin_id.a:*", "plugin_id.a:uid:*"},
		},
		{
			name:     "two prefixes",
			prefixes: []string{"plugin_id.a:uid:", "plugin_id.b:uid:"},
			want:     []string{"*", "plugin_id.a:*", "plugin_id.a:uid:*", "plugin_id.b:*", "plugin_id.b:uid:*"},
		},
		{
			name:     "long prefix",
			prefixes: []string{"plugin_id.a:sub:uid:"},
			want:     []string{"*", "plugin_id.a:*", "plugin_id.a:sub:*", "plugin_id.a:sub:uid:*"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, WildcardsFromPrefixes(tt.prefixes...), tt.want)
		})
	}
}

func TestGenerateChecker(t *testing.T) {
	userPermissions := Permissions{
		"pid.dashboards:create": []string{},                                                                                         // no scope
		"pid.dashboards:read":   []string{"pid.dashboards:uid:*", "pid.folders:uid:*"},                                              // wildcards
		"pid.dashboards:write":  []string{"pid.dashboards:uid:1", "pid.dashboards:uid:2", "pid.folders:uid:3", "pid.folders:uid:4"}, // folders or dashboards
		"pid.dashboards:delete": []string{"pid.folders:uid:3"},                                                                      // should have no scope
	}

	type match struct {
		scopes    []string
		hasAccess bool
	}
	tests := []struct {
		name        string
		permissions Permissions
		action      string
		prefixes    []string
		want        match
	}{
		{
			name:        "no match user has no permission",
			permissions: Permissions{},
			action:      "pid.dashboards:read",
			prefixes:    []string{"pid.dashboards:uid"},
			want:        match{scopes: []string{"pid.dashboards:uid:1"}, hasAccess: false},
		},
		{
			name:        "no match user does not have the permission",
			permissions: userPermissions,
			action:      "pid.folders:read",
			prefixes:    []string{"pid.folders:uid"},
			want:        match{scopes: []string{"pid.folders:uid:2"}, hasAccess: false},
		},
		{
			name:        "match on action only",
			permissions: userPermissions,
			action:      "pid.dashboards:create",
			prefixes:    []string{},
			want:        match{scopes: []string{}, hasAccess: true},
		},
		{
			name:        "no match on action only",
			permissions: userPermissions,
			action:      "pid.dashboards:print",
			prefixes:    []string{},
			want:        match{scopes: []string{}, hasAccess: false},
		},
		{
			name:        "no match on action only when user action is scoped",
			permissions: userPermissions,
			action:      "pid.dashboards:delete",
			prefixes:    []string{},
			want:        match{scopes: []string{}, hasAccess: false},
		},
		{
			name:        "match user has specific permission",
			permissions: userPermissions,
			action:      "pid.dashboards:write",
			prefixes:    []string{"pid.dashboards:uid"},
			want:        match{scopes: []string{"pid.dashboards:uid:2"}, hasAccess: true},
		},
		{
			name:        "match user has wildcard permission",
			permissions: userPermissions,
			action:      "pid.dashboards:read",
			prefixes:    []string{"pid.dashboards:uid"},
			want:        match{scopes: []string{"pid.dashboards:uid:1"}, hasAccess: true},
		},
		{
			name:        "no match user has action but on none of the desired",
			permissions: userPermissions,
			action:      "pid.dashboards:write",
			prefixes:    []string{"pid.dashboards:uid"},
			want:        match{scopes: []string{"pid.dashboards:uid:55", "pid.dashboards:uid:56"}, hasAccess: false},
		},
		{
			name:        "match checker built with multiple prefixes",
			permissions: userPermissions,
			action:      "pid.dashboards:write",
			prefixes:    []string{"pid.dashboards:uid", "pid.folders:uid"},
			want:        match{scopes: []string{"pid.dashboard:uid:55", "pid.folders:uid:3"}, hasAccess: true},
		},
		{
			name:        "match when at least one scope is found",
			permissions: userPermissions,
			action:      "pid.dashboards:write",
			prefixes:    []string{"pid.dashboards:uid"},
			want:        match{scopes: []string{"pid.dashboards:uid:55", "pid.dashboards:uid:2"}, hasAccess: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := GenerateChecker(context.Background(), tt.permissions, tt.action, tt.prefixes...)
			got := checker(tt.want.scopes...)
			require.Equal(t, tt.want.hasAccess, got)
		})
	}
}

func TestCheckerExamples(t *testing.T) {
	type dashboard struct {
		UID       string
		parentUID string
	}

	userPermissions := Permissions{
		"pid.dashboards:create": []string{},
		"pid.dashboards:read":   []string{"pid.dashboards:uid:*", "pid.folders:uid:*"},
		"pid.dashboards:write":  []string{"pid.dashboards:uid:dash1", "pid.dashboards:uid:dash2", "pid.folders:uid:fold1", "pid.folders:uid:fold2"},
	}

	dashboards := []dashboard{
		{UID: "dash1", parentUID: "fold1"}, // Can write dash directly and through folder
		{UID: "dash2", parentUID: "fold3"}, // Can write dash directly
		{UID: "dash3", parentUID: "fold2"}, // Can write dash through folder
		{UID: "dash4", parentUID: "fold3"}, // Cannot write dash
	}

	// Check on action only
	canCreateDashboards := GenerateChecker(context.Background(), userPermissions, "pid.dashboards:create")
	require.True(t, canCreateDashboards())
	canDeleteDashboards := GenerateChecker(context.Background(), userPermissions, "pid.dashboards:delete")
	require.False(t, canDeleteDashboards())

	// Check on either dashboard or folder
	canReadDashboards := GenerateChecker(context.Background(), userPermissions, "pid.dashboards:read", "pid.dashboards:uid", "pid.folders:uid")
	require.True(t, canReadDashboards("pid.dashboards:uid:dash2"), "should be allowed to read dashboard")
	require.True(t, canReadDashboards("pid.folders:uid:fold2"), "should be allowed to read dashboard in the folder")
	require.True(t, canReadDashboards("pid.dashboards:uid:dash4", "pid.folders:uid:fold3"), "should be allowed to read dashboards in the folder")

	// Filter resources
	canWriteDashboards := GenerateChecker(context.Background(), userPermissions, "pid.dashboards:write", "pid.dashboards:uid", "pid.folders:uid")
	writeOK := []string{}
	for _, dash := range dashboards {
		if canWriteDashboards("pid.dashboards:uid:"+dash.UID, "pid.folders:uid:"+dash.parentUID) {
			writeOK = append(writeOK, dash.UID)
		}
	}
	require.EqualValues(t, []string{"dash1", "dash2", "dash3"}, writeOK)
}
