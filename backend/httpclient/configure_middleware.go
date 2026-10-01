package httpclient

// ConfigureBasicAuthBeforeContextualMiddleware is a ConfigureMiddlewareFunc that moves
// BasicAuthenticationMiddleware to run before ContextualMiddleware in the middleware chain.
// This is provided to restore the original auth precedence of DefaultMiddlewares before
// https://github.com/grafana/grafana-plugin-sdk-go/pull/1735 in case of unwanted side effects.
// If existingMiddleware doesn't contain both BasicAuthenticationMiddleware and
// ContextualMiddleware, it's returned unmodified.
//
// DefaultMiddlewares applies ContextualMiddleware before BasicAuthenticationMiddleware, so a
// forwarded Authorization header (e.g. a bearer token forwarded because of Forward OAuth Identity)
// wins over configured basic authentication credentials. Assigning this function to
// Options.ConfigureMiddleware restores the original precedence: configured basic authentication
// credentials win over a forwarded Authorization header.
//
// This function does not affect the precedence of an Authorization header provided by
// CustomHeadersMiddleware, since that middleware overwrites the header unconditionally. A custom
// Authorization header always wins, regardless of the middleware order.
func ConfigureBasicAuthBeforeContextualMiddleware(_ Options, existingMiddleware []Middleware) []Middleware {
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
	insertAt := contextualIdx
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
