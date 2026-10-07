package httpclient

import "slices"

// ConfigureBasicAuthAfterContextualMiddleware is a ConfigureMiddlewareFunc that moves
// BasicAuthenticationMiddleware to run after ContextualMiddleware in the middleware chain.
// If existingMiddleware doesn't contain both BasicAuthenticationMiddleware and
// ContextualMiddleware, or if BasicAuthenticationMiddleware is already after
// ContextualMiddleware, it's returned unmodified.
//
// DefaultMiddlewares applies BasicAuthenticationMiddleware before ContextualMiddleware, so a
// configured basic authentication credential always wins over an Authorization header forwarded
// via a contextual middleware, e.g. a bearer token forwarded because of Forward OAuth Identity.
// Assigning this function to Options.ConfigureMiddleware reverses that precedence: a forwarded
// Authorization header then wins, while basic authentication is still used as a fallback for
// requests that don't carry a forwarded header.
func ConfigureBasicAuthAfterContextualMiddleware(_ Options, existingMiddleware []Middleware) []Middleware {
	indexOf := func(name string) int {
		return slices.IndexFunc(existingMiddleware, func(m Middleware) bool {
			return middlewareName(m) == name
		})
	}
	basicIdx := indexOf(BasicAuthenticationMiddlewareName)
	contextualIdx := indexOf(ContextualMiddlewareName)
	if basicIdx == -1 || contextualIdx == -1 || basicIdx > contextualIdx {
		return existingMiddleware
	}

	reordered := slices.Clone(existingMiddleware)
	basic := reordered[basicIdx]
	reordered = slices.Delete(reordered, basicIdx, basicIdx+1)
	return slices.Insert(reordered, contextualIdx, basic)
}

func middlewareName(m Middleware) string {
	n, ok := m.(MiddlewareName)
	if !ok {
		return ""
	}
	return n.MiddlewareName()
}
