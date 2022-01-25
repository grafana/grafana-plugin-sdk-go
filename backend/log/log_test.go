package log_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/stretchr/testify/assert"
)

func TestLogLevel(t *testing.T) {
	logger := log.New()
	level := logger.Level()
	assert.Equal(t, level, log.Debug)
}

func TestLogLevelWarn(t *testing.T) {
	logger := log.NewWithLevel(log.Warn)
	level := logger.Level()
	assert.Equal(t, level, log.Warn)
}
