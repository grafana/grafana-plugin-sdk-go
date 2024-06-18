package backend

import (
	"errors"
	"runtime/debug"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

func panicGuard[R *QueryDataResponse |
	*CheckHealthResult |
	*SubscribeStreamResponse |
	*PublishStreamResponse |
	interface{}](f func() (R, error)) (res R, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Error("panic recovered", "error", r, "stack", string(debug.Stack()))
			err = errors.New("internal server error")
			return
		}
	}()

	res, err = f()
	return
}
