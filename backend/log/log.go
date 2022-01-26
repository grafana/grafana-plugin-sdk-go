// Package log provides a logging interface to send logs from plugins to Grafana server.
package log

import (
	hclog "github.com/hashicorp/go-hclog"
)

type Level int32

const (
	NoLevel Level = iota
	Trace
	Debug
	Info
	Warn
	Error
)

// Logger is the main Logger interface.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Level() Level
}

// New creates a new logger.
func New() Logger {
	return NewWithLevel(Debug)
}

// NewWithLevel creates a new logger at the Level defined.
func NewWithLevel(level Level) Logger {
	return &hclogWrapper{
		logger: hclog.New(&hclog.LoggerOptions{
			// Use debug as level since anything less severe is suppressed.
			Level: hclog.Level(level),
			// Use JSON format to make the output in Grafana format and work
			// when using multiple arguments such as Debug("message", "key", "value").
			JSONFormat: true,
		}),
	}
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

func (l *hclogWrapper) Level() Level {
	if l.logger.IsDebug() {
		return Debug
	}
	if l.logger.IsTrace() {
		return Trace
	}
	if l.logger.IsInfo() {
		return Info
	}
	if l.logger.IsWarn() {
		return Warn
	}
	if l.logger.IsError() {
		return Error
	}
	return NoLevel
}

// DefaultLogger is the default logger.
var DefaultLogger = New()
