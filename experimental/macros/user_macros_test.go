package macros_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/macros"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserMacro(t *testing.T) {
	pc := backend.PluginContext{
		User: &backend.User{
			UserID: "4",
			Name:   "Foo",
			Email:  "foo@bar.com",
			Role:   "viewer",
		},
	}
	tests := []struct {
		name          string
		inputString   string
		pluginContext *backend.PluginContext
		want          string
		wantErr       error
	}{
		{inputString: "${__user}", want: "4"},
		{inputString: "${__user:id}", want: "4"},
		{inputString: "${__user:email}", want: "foo@bar.com"},
		{inputString: "${__user:name}", want: "Foo"},
		{inputString: "${__user:does-not-exist}", want: "4"},
		{inputString: "Foo${__user}Bar${__user:email}Baz", want: "Foo4Barfoo@bar.comBaz"},
		{inputString: "${__user}", pluginContext: &backend.PluginContext{}, wantErr: macros.ErrUserContextNotExit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pluginContext != nil {
				pc = *tt.pluginContext
			}
			got, err := macros.UserMacro(tt.inputString, pc)
			if tt.wantErr != nil {
				require.NotNil(t, err)
				assert.Equal(t, tt.wantErr, err)
				return
			}
			require.Nil(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
