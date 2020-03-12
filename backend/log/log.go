package log

import (
	hclog "github.com/hashicorp/go-hclog"
)

// Logger the main Logger interface.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// NewLoggerWithName creates a named logger
func NewLoggerWithName(name string) Logger {
	return &hclogWrapper{
		logger: hclog.New(&hclog.LoggerOptions{
			Name: name,
			// Use debug as level since anything less severe is supressed.
			Level: hclog.Debug,
			// Use JSON format to make the output in Grafana format and work
			// when using multiple arguments such as Debug("message", "key", "value").
			JSONFormat: true,
		}),
	}
}

// New creates a logger without a name
func New() Logger {
	return NewLoggerWithName("")
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
