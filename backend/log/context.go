package log

import "context"

type loggerCtxKeyType struct{}

var loggerCtxKey = loggerCtxKeyType{}

// FromContext returns a logger from the context if one is set, otherwise it returns [DefaultLogger].
func FromContext(ctx context.Context) Logger {
	logger, ok := ctx.Value(loggerCtxKey).(Logger)
	if !ok {
		return DefaultLogger
	}
	return logger
}

// WithContext returns a new context with the logger set to the provided value.
// The logger can be retrieved later using [FromContext].
func WithContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey, logger)
}
