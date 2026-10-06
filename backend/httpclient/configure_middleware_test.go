package httpclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigureBasicAuthAfterContextualMiddleware(t *testing.T) {
	t.Run("Moves BasicAuthenticationMiddleware to right after ContextualMiddleware", func(t *testing.T) {
		existing := []Middleware{
			BasicAuthenticationMiddleware(),
			CustomHeadersMiddleware(),
			ContextualMiddleware(),
			ErrorSourceMiddleware(),
		}
		reordered := ConfigureBasicAuthAfterContextualMiddleware(Options{}, existing)
		names := make([]string, 0, len(reordered))
		for _, m := range reordered {
			names = append(names, middlewareName(m))
		}
		assert.Equal(t, []string{
			CustomHeadersMiddlewareName,
			ContextualMiddlewareName,
			BasicAuthenticationMiddlewareName,
			ErrorSourceMiddlewareName,
		}, names)
	})

	t.Run("Returns existingMiddleware unmodified when BasicAuthenticationMiddleware is missing", func(t *testing.T) {
		existing := []Middleware{CustomHeadersMiddleware(), ContextualMiddleware(), ErrorSourceMiddleware()}
		reordered := ConfigureBasicAuthAfterContextualMiddleware(Options{}, existing)
		assert.Equal(t, existing, reordered)
	})

	t.Run("Returns existingMiddleware unmodified when ContextualMiddleware is missing", func(t *testing.T) {
		existing := []Middleware{BasicAuthenticationMiddleware(), CustomHeadersMiddleware(), ErrorSourceMiddleware()}
		reordered := ConfigureBasicAuthAfterContextualMiddleware(Options{}, existing)
		assert.Equal(t, existing, reordered)
	})

	t.Run("Returns existingMiddleware unmodified when BasicAuthenticationMiddleware and ContextualMiddleware are missing", func(t *testing.T) {
		existing := []Middleware{CustomHeadersMiddleware(), ErrorSourceMiddleware()}
		reordered := ConfigureBasicAuthAfterContextualMiddleware(Options{}, existing)
		assert.Equal(t, existing, reordered)
	})
}
