package macros

import (
	"errors"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

var (
	ErrUserContextNotExit error = errors.New("user context doesn't exist in the plugin context")
)

func UserMacro(inputString string, pluginContext backend.PluginContext) (string, error) {
	// ${__user} 			--> 4
	// ${__user.id} 		--> 4
	// ${__user.login} 		--> foo
	// ${__user.email}		--> foo@bar.com
	// ${__user.name}		--> Foo
	res, err := applyMacro("$$user", inputString, func(query string, args []string) (string, error) {
		if pluginContext.User == nil {
			return "", ErrUserContextNotExit
		}
		switch args[0] {
		case "id":
			return pluginContext.User.UserID, nil
		case "login":
			return pluginContext.User.Login, nil
		case "email":
			return pluginContext.User.Email, nil
		case "name":
			return pluginContext.User.Name, nil
		default:
			return pluginContext.User.UserID, nil
		}
	})
	return res, err
}
