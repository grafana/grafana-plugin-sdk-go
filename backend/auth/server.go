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

// V3MethodPrefix prefixes the full gRPC method names of the plugin protocol v3
// services, which a plugin serves through backend.ServeOpts.ExtraPlugins. The
// server interceptors authenticate these methods only.
const V3MethodPrefix = "/grafana.plugin.v3."

// RequestAuthenticator authenticates the requests a plugin serves, with
// resolved Settings (see Settings.Resolve). The plugin SDK makes one for every
// plugin it serves.
type RequestAuthenticator struct {
	// authenticator parses access tokens; nil when none can be.
	authenticator authn.Authenticator
	// verifies is set when authenticator verifies tokens, rather than only
	// parsing them for InsecureSkipAuthentication.
	verifies bool
}

// NewRequestAuthenticator returns a RequestAuthenticator for s. With
// s.Authenticator, tokens are verified. Otherwise, with
// s.InsecureSkipAuthentication, they are parsed without being verified.
// Without either, plugin protocol v3 requests are rejected, and plugin
// protocol v2 calls are served with no identity (see AuthenticateHeader).
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

// Verifies reports whether tokens are verified, rather than parsed without
// verification for InsecureSkipAuthentication. Only then should the services
// check a request's API group and namespaces against the caller's AuthInfo.
func (a *RequestAuthenticator) Verifies() bool {
	return a.verifies
}

// AuthenticateHeader returns ctx with the AuthInfo of the caller whose access
// token header carries, the value of a plugin protocol v2 call's
// AccessTokenHeader. A "Bearer " prefix is accepted.
//
// A call without a token is let through with no identity: Grafana sends a
// token only on the calls it makes on behalf of a caller, such as the health
// and resource calls of an app's settings. A token that does not verify fails
// the call. A verified token must cover namespace, or carry a namespace if
// namespace is empty.
//
// A token that cannot be verified, because the plugin has no authenticator,
// is ignored with a warning: the call is let through with no identity, as
// without a token, so that plugins Grafana sends tokens to keep working until
// they are given signing keys.
func (a *RequestAuthenticator) AuthenticateHeader(ctx context.Context, namespace, header string) (context.Context, error) {
	token := strings.TrimSpace(header)
	if len(token) > len("Bearer ") && strings.EqualFold(token[:len("Bearer ")], "Bearer ") {
		token = strings.TrimSpace(token[len("Bearer "):])
	}
	if token == "" {
		log.DefaultLogger.FromContext(ctx).Debug("plugin v2 call without an access token; it is served without the caller's identity", "namespace", namespace)
		return ctx, nil
	}
	if a.authenticator == nil {
		// TODO: reject the call once every plugin Grafana sends tokens to
		// has signing keys; its handler now acts as the plugin, not the caller.
		log.DefaultLogger.FromContext(ctx).Warn("plugin v2 call with an access token, but no authenticator configured to verify it; it is served without the caller's identity: set "+EnvVarGrafanaJWKSURL+" or "+EnvVarGrafanaJWKS,
			"namespace", namespace)
		return ctx, nil
	}
	ctx, err := AuthenticateAccessToken(ctx, a.authenticator, token)
	if err != nil {
		return nil, err
	}
	if a.verifies {
		info, _ := types.AuthInfoFrom(ctx)
		if namespace == "" {
			if info.GetNamespace() == "" {
				return nil, status.Error(codes.PermissionDenied, "access token does not cover the requested namespace")
			}
		} else if !types.NamespaceMatches(info.GetNamespace(), namespace) {
			return nil, status.Error(codes.PermissionDenied, "access token does not cover the requested namespace")
		}
	}
	return ctx, nil
}

// AuthenticateMetadata returns ctx with the AuthInfo of the caller whose
// access token the incoming gRPC metadata of ctx carries, as for a plugin
// protocol v3 request. The request needs a single access token and no
// separate ID token; without verification, a request with neither is served
// with no identity. Without an authenticator, every request is rejected.
//
// The services check the request's API group and namespaces against the
// AuthInfo, as only they can read the request.
func (a *RequestAuthenticator) AuthenticateMetadata(ctx context.Context) (context.Context, error) {
	if a.authenticator == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin has no authenticator configured; set "+EnvVarGrafanaJWKSURL+" or "+EnvVarGrafanaJWKS)
	}
	md, _ := metadata.FromIncomingContext(ctx)
	tokens := md.Get(AccessTokenMetadataKey)
	idTokens := md.Get(IDTokenMetadataKey)
	if !a.verifies && len(tokens) == 0 && len(idTokens) == 0 {
		// Without verification, a request without a token has no identity.
		return ctx, nil
	}
	// Delegation has one source of identity: the exchanged access token.
	// Reject ambiguous credentials before invoking even a custom authenticator.
	if len(tokens) != 1 || tokens[0] == "" || len(idTokens) != 0 {
		return nil, status.Error(codes.Unauthenticated, "a single access token and no separate ID token are required")
	}
	return AuthenticateAccessToken(ctx, a.authenticator, tokens[0])
}

// UnaryServerInterceptor authenticates plugin protocol v3 unary requests with
// AuthenticateMetadata. Other methods pass through.
func (a *RequestAuthenticator) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !strings.HasPrefix(info.FullMethod, V3MethodPrefix) {
			return handler(ctx, req)
		}
		ctx, err := a.AuthenticateMetadata(ctx)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamServerInterceptor authenticates plugin protocol v3 streaming requests
// with AuthenticateMetadata. Other methods pass through.
func (a *RequestAuthenticator) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !strings.HasPrefix(info.FullMethod, V3MethodPrefix) {
			return handler(srv, stream)
		}
		ctx, err := a.AuthenticateMetadata(stream.Context())
		if err != nil {
			return err
		}
		return handler(srv, &authenticatedServerStream{ServerStream: stream, ctx: ctx})
	}
}

type authenticatedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedServerStream) Context() context.Context {
	return s.ctx
}
