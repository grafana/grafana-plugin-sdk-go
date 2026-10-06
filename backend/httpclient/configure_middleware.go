package httpclient

// ConfigureBasicAuthAfterContextualMiddleware is a ConfigureMiddlewareFunc that moves
// BasicAuthenticationMiddleware to run after ContextualMiddleware in the middleware chain.
// If existingMiddleware doesn't contain both BasicAuthenticationMiddleware and
// ContextualMiddleware, it's returned unmodified.
//
// DefaultMiddlewares applies BasicAuthenticationMiddleware before ContextualMiddleware, so a
// configured basic authentication credential always wins over an Authorization header forwarded
// via a contextual middleware, e.g. a bearer token forwarded because of Forward OAuth Identity.
// Assigning this function to Options.ConfigureMiddleware reverses that precedence: a forwarded
// Authorization header then wins, while basic authentication is still used as a fallback for
// requests that don't carry a forwarded header.
func ConfigureBasicAuthAfterContextualMiddleware(_ Options, existingMiddleware []Middleware) []Middleware {
	var basic Middleware
	contextualIdx := -1
	out := make([]Middleware, 0, len(existingMiddleware))
	for _, m := range existingMiddleware {
		switch middlewareName(m) {
		case BasicAuthenticationMiddlewareName:
			basic = m
			continue
		case ContextualMiddlewareName:
			contextualIdx = len(out)
		}
		out = append(out, m)
	}
	if basic == nil || contextualIdx == -1 {
		return existingMiddleware
	}
	insertAt := contextualIdx + 1
	reordered := make([]Middleware, 0, len(out)+1)
	reordered = append(reordered, out[:insertAt]...)
	reordered = append(reordered, basic)
	reordered = append(reordered, out[insertAt:]...)
	return reordered
}

func middlewareName(m Middleware) string {
	n, ok := m.(MiddlewareName)
	if !ok {
		return ""
	}
	return n.MiddlewareName()
}
