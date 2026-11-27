// Package auth checks the access tokens that Grafana sends with a request on
// behalf of a user.
//
// The plugin SDK checks every plugin protocol v2 call that has an
// X-Access-Token header, and every plugin protocol v3 request. A handler can
// then get the caller from its context with types.AuthInfoFrom, and use the
// caller's token to act on their behalf.
//
// Settings configure this. backend.Manage reads them from the environment
// variables Grafana sets.
package auth
