package log

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/metadata"
)

var loggerParamsCtxMetadataKey = "loggerParamsCtxMetadata"

// WithContextualAttributesForOutgoingContext returns a new context with the given key/value log parameters appended to the existing ones.
// It's possible to get a logger with those contextual parameters by using [FromContext].
func WithContextualAttributesForOutgoingContext(ctx context.Context, logParams []any) context.Context {
	if len(logParams) == 0 || len(logParams)%2 != 0 {
		return ctx
	}

	for i := 0; i < len(logParams); i += 2 {
		k := logParams[i].(string)
		v := logParams[i+1].(string)
		ctx = metadata.AppendToOutgoingContext(ctx, loggerParamsCtxMetadataKey, fmt.Sprintf("%s:%s", k, v))
	}

	return ctx
}

// ContextualAttributesFromIncomingContext returns the contextual key/value log parameters from the given context.
// If no contextual log parameters are set, it returns nil.
func ContextualAttributesFromIncomingContext(ctx context.Context) []any {
	logParams := metadata.ValueFromIncomingContext(ctx, loggerParamsCtxMetadataKey)
	if len(logParams) == 0 {
		return nil
	}

	var res []any
	for _, param := range logParams {
		kv := strings.Split(param, ":")
		res = append(res, kv[0], kv[1])
	}
	return res
}
