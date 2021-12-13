package log_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/stretchr/testify/assert"
)

func TestLogLevel(t *testing.T) {
	logger := log.New()
	level := logger.Level()
	assert.Equal(t, level, log.Level(log.Debug))
}
