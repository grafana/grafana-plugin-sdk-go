package backend

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// Logger is the default logger instance. This can be used directly to log from
// your plugin to grafana-server with calls like backend.Logger.Debug(...).
var Logger = log.DefaultLogger

// NewLoggerWith creates a new logger with the given arguments.
var NewLoggerWith = func(args ...interface{}) log.Logger {
	return log.New().With(args...)
}
