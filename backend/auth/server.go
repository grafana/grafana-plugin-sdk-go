package auth

import (
	"context"
	"strings"

	"github.com/grafana/authlib/authn"
	"github.com/grafana/authlib/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// v3MethodPrefix starts the gRPC method name of every plugin protocol v3
// request. The interceptors only check these: v3 sends the token in gRPC
// metadata, while v2 sends it in the request headers, which the backend
// package checks in a handler middleware.
const v3MethodPrefix = "/grafana.plugin.v3."

// RequestAuthenticator checks the access tokens on the requests a plugin
// serves.
type RequestAuthenticator struct {
	// authenticator reads tokens. It is nil when none is configured.
	authenticator authn.Authenticator
	// verifies is false when tokens are only parsed, for
	// InsecureSkipAuthentication.
	verifies bool
}

// NewRequestAuthenticator returns a RequestAuthenticator for s, which should
// already be resolved (see Settings.Resolve).
func NewRequestAuthenticator(s Settings) *RequestAuthenticator {
	switch {
	case s.Authenticator != nil:
		return &RequestAuthenticator{authenticator: s.Authenticator, verifies: true}
	case s.InsecureSkipAuthentication:
		return &RequestAuthenticator{authenticator: authn.NewAccessTokenAuthenticator(authn.NewUnsafeAccessTokenVerifier(authn.VerifierConfig{}))}
	default:
		return &RequestAuthenticator{}
	}
}

// AuthenticateHeader checks the token in a plugin protocol v2 call's
// AccessTokenHeader, with or without a "Bearer " prefix, and returns ctx with
// the caller's AuthInfo.
//
//   - No token: the call has no identity. Grafana only sends tokens on calls
//     made for a user, such as an app's settings health and resource calls.
//   - A valid token: it must cover namespace (or have a namespace, if
//     namespace is empty).
//   - An invalid token: the call fails.
//   - A token but no authenticator: a warning is logged and the call has no
//     identity.
func (a *RequestAuthenticator) AuthenticateHeader(ctx context.Context, namespace, header string) (context.Context, error) {
	token := trimBearer(header)
	if token == "" {
		return ctx, nil
	}
	if a.authenticator == nil {
		// TODO: fail the call once every plugin that gets tokens has signing keys.
		log.DefaultLogger.FromContext(ctx).Warn("Ignoring the access token: no authenticator is configured; set "+EnvVarGrafanaJWKSURL+" or "+EnvVarGrafanaJWKS,
			"namespace", namespace)
		return ctx, nil
	}
	ctx, err := AuthenticateAccessToken(ctx, a.authenticator, token)
	if err != nil {
		return nil, err
	}
	if a.verifies {
		info, _ := types.AuthInfoFrom(ctx)
		tokenNamespace := info.GetNamespace()
		if (namespace == "" && tokenNamespace == "") || (namespace != "" && !types.NamespaceMatches(tokenNamespace, namespace)) {
			return nil, status.Error(codes.PermissionDenied, "access token does not cover the requested namespace")
		}
	}
	return ctx, nil
}

// AuthenticateMetadata checks the token in a plugin protocol v3 request's
// gRPC metadata, and returns ctx with the caller's AuthInfo.
//
// The request needs exactly one access token and no ID token. Without an
// authenticator, every request fails. With InsecureSkipAuthentication, a
// request without tokens has no identity.
//
// The v3 services still need to check the request's API group and
// namespaces, which only they can read.
func (a *RequestAuthenticator) AuthenticateMetadata(ctx context.Context) (context.Context, error) {
	if a.authenticator == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin has no authenticator configured; set "+EnvVarGrafanaJWKSURL+" or "+EnvVarGrafanaJWKS)
	}
	md, _ := metadata.FromIncomingContext(ctx)
	tokens := md.Get(AccessTokenMetadataKey)
	idTokens := md.Get(idTokenMetadataKey)
	if !a.verifies && len(tokens) == 0 && len(idTokens) == 0 {
		return ctx, nil
	}
	// The exchanged access token must be the only identity.
	if len(tokens) != 1 || tokens[0] == "" || len(idTokens) != 0 {
		return nil, status.Error(codes.Unauthenticated, "a single access token and no separate ID token are required")
	}
	return AuthenticateAccessToken(ctx, a.authenticator, tokens[0])
}

// UnaryServerInterceptor checks plugin protocol v3 unary requests with
// AuthenticateMetadata. Other requests pass through.
func (a *RequestAuthenticator) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !strings.HasPrefix(info.FullMethod, v3MethodPrefix) {
			return handler(ctx, req)
		}
		ctx, err := a.AuthenticateMetadata(ctx)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamServerInterceptor checks plugin protocol v3 streaming requests with
// AuthenticateMetadata. Other requests pass through.
func (a *RequestAuthenticator) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !strings.HasPrefix(info.FullMethod, v3MethodPrefix) {
			return handler(srv, stream)
		}
		ctx, err := a.AuthenticateMetadata(stream.Context())
		if err != nil {
			return err
		}
		return handler(srv, &authenticatedServerStream{ServerStream: stream, ctx: ctx})
	}
}

// authenticatedServerStream is a stream with the caller's AuthInfo in its context.
type authenticatedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedServerStream) Context() context.Context {
	return s.ctx
}

// trimBearer returns the token in header, without spaces or a "Bearer " prefix
// (in any case).
func trimBearer(header string) string {
	token := strings.TrimSpace(header)
	const prefix = "Bearer "
	if len(token) > len(prefix) && strings.EqualFold(token[:len(prefix)], prefix) {
		token = strings.TrimSpace(token[len(prefix):])
	}
	return token
}
