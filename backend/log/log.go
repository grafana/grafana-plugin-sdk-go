package log

import (
	hclog "github.com/hashicorp/go-hclog"
)

var defaultLogger Logger

func init() {
	defaultLogger = New()
}

// Logger the main Logger interface.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// New creates a new logger.
func New() Logger {
	return &hclogWrapper{
		logger: hclog.New(&hclog.LoggerOptions{
			Level:      hclog.LevelFromString("DEBUG"),
			JSONFormat: true,
		}),
	}
}

// Default returns the default logger.
func Default() Logger {
	return defaultLogger
}

type hclogWrapper struct {
	logger hclog.Logger
}

func (l *hclogWrapper) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}

func (l *hclogWrapper) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

func (l *hclogWrapper) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}

func (l *hclogWrapper) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
}
