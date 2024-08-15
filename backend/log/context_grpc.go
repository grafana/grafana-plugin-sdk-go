package log

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	loggerParamsCtxMetadataKey = "loggerParamsCtxMetadata"
	logParamSeparator          = ":"
)

// WithContextualAttributesForOutgoingContext will append the given key/value log parameters to the outgoing context.
func WithContextualAttributesForOutgoingContext(ctx context.Context, logParams []any) context.Context {
	if len(logParams) == 0 || len(logParams)%2 != 0 {
		return ctx
	}

	for i := 0; i < len(logParams); i += 2 {
		k := logParams[i].(string)
		v := logParams[i+1].(string)
		ctx = metadata.AppendToOutgoingContext(ctx, loggerParamsCtxMetadataKey, fmt.Sprintf("%s%s%s", k, logParamSeparator, v))
	}

	return ctx
}

// ContextualAttributesFromIncomingContext returns the contextual key/value log parameters from the given incoming context.
func ContextualAttributesFromIncomingContext(ctx context.Context) []any {
	logParams := metadata.ValueFromIncomingContext(ctx, loggerParamsCtxMetadataKey)
	if len(logParams) == 0 {
		return nil
	}

	var attrs []any
	for _, param := range logParams {
		kv := strings.Split(param, logParamSeparator)
		attrs = append(attrs, kv[0], kv[1])
	}
	return attrs
}
