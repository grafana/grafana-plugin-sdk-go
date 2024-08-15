package log

import (
	"context"
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

	// join the key/value pairs with a colon, and separate the pairs with a comma
	var res strings.Builder
	for i := 0; i < len(logParams); i += 2 {
		if i > 0 {
			res.WriteString(",")
		}
		res.WriteString(logParams[i].(string))
		res.WriteString(":")
		res.WriteString(logParams[i+1].(string))
	}

	return metadata.AppendToOutgoingContext(ctx, loggerParamsCtxMetadataKey, res.String())
}

// ContextualAttributesFromIncomingContext returns the contextual key/value log parameters from the given context.
// If no contextual log parameters are set, it returns nil.
func ContextualAttributesFromIncomingContext(ctx context.Context) []any {
	logParams := metadata.ValueFromIncomingContext(ctx, loggerParamsCtxMetadataKey)
	if len(logParams) == 0 {
		return nil
	}

	kvs := strings.Split(logParams[0], ",")

	var res []any
	for i := 0; i < len(kvs); i++ {
		kv := strings.Split(kvs[i], ":")
		res = append(res, kv[0], kv[1])
	}
	return res
}
