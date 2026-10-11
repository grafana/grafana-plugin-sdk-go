package backend

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync/atomic"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/status"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

const (
	errorSourceMetadataKey = "errorSource"

	// ResponseSizeBytesTrailerKey is the gRPC trailer QueryChunkedData sets to the total
	// size in bytes of the encoded frames it sent.
	ResponseSizeBytesTrailerKey = "response-size-bytes"
)

// dataSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type dataSDKAdapter struct {
	queryDataHandler        QueryDataHandler
	queryChunkedDataHandler QueryChunkedDataHandler
}

// newDataSDKAdapter creates a new adapter between the plugin protocol and SDK interfaces.
// It handles both query data and chunked query data operations.
func newDataSDKAdapter(queryDataHandler QueryDataHandler, queryChunkedDataHandler QueryChunkedDataHandler) *dataSDKAdapter {
	return &dataSDKAdapter{
		queryDataHandler:        queryDataHandler,
		queryChunkedDataHandler: queryChunkedDataHandler,
	}
}

// QueryData handles incoming gRPC data requests by converting them to SDK format
// and passing them to the registered QueryDataHandler.
func (a *dataSDKAdapter) QueryData(ctx context.Context, req *pluginv2.QueryDataRequest) (*pluginv2.QueryDataResponse, error) {
	parsedReq := FromProto().QueryDataRequest(req)
	resp, err := a.queryDataHandler.QueryData(ctx, parsedReq)
	if err != nil {
		return nil, enrichWithErrorSourceInfo(err)
	}

	if resp == nil {
		return nil, errors.New("both response and error are nil, but one must be provided")
	}

	return ToProto().QueryDataResponse(parsedReq.Format, resp)
}

// QueryChunkedData handles incoming gRPC stream data requests by converting them to SDK format
// and passing them to the registered QueryChunkedDataHandler.
func (a *dataSDKAdapter) QueryChunkedData(req *pluginv2.QueryChunkedDataRequest, stream grpc.ServerStreamingServer[pluginv2.QueryChunkedDataResponse]) error {
	if a.queryChunkedDataHandler == nil {
		return stream.Send(&pluginv2.QueryChunkedDataResponse{
			Status: http.StatusNotImplemented,
		})
	}

	ctx := stream.Context()
	parsedReq := FromProto().QueryChunkedDataRequest(req)

	var sentBytes atomic.Int64
	writer := NewChunkedDataWriter(parsedReq.Format, func(chunk *pluginv2.QueryChunkedDataResponse) error {
		if err := stream.Send(chunk); err != nil {
			return err
		}
		sentBytes.Add(int64(len(chunk.Frame)))
		return nil
	})
	// The total is only known once the stream ends, so it goes out as a trailer.
	defer func() {
		stream.SetTrailer(metadata.Pairs(ResponseSizeBytesTrailerKey, strconv.FormatInt(sentBytes.Load(), 10)))
	}()

	err := a.queryChunkedDataHandler.QueryChunkedData(ctx, parsedReq, writer)
	if err != nil {
		return enrichWithErrorSourceInfo(err)
	}

	return nil
}

// enrichWithErrorSourceInfo returns a gRPC status error with error source info as metadata.
func enrichWithErrorSourceInfo(err error) error {
	var errorSource status.Source
	if IsDownstreamError(err) {
		errorSource = status.SourceDownstream
	} else if IsPluginError(err) {
		errorSource = status.SourcePlugin
	}

	// Unless the error is explicitly marked as a plugin or downstream error, we don't enrich it.
	if errorSource == "" {
		return err
	}

	status := grpcstatus.New(codes.Unknown, err.Error())
	status, innerErr := status.WithDetails(&errdetails.ErrorInfo{
		Metadata: map[string]string{
			errorSourceMetadataKey: errorSource.String(),
		},
	})
	if innerErr != nil {
		return err
	}

	return status.Err()
}

// HandleGrpcStatusError handles gRPC status errors by extracting the error source from the error details and injecting
// the error source into context.
func ErrorSourceFromGrpcStatusError(ctx context.Context, err error) (status.Source, bool) {
	st := grpcstatus.Convert(err)
	if st == nil {
		return status.DefaultSource, false
	}
	for _, detail := range st.Details() {
		if errorInfo, ok := detail.(*errdetails.ErrorInfo); ok {
			errorSource, exists := errorInfo.Metadata[errorSourceMetadataKey]
			if !exists {
				break
			}

			switch errorSource {
			case string(ErrorSourceDownstream):
				innerErr := WithErrorSource(ctx, ErrorSourceDownstream)
				if innerErr != nil {
					Logger.Error("Could not set downstream error source", "error", innerErr)
				}
				return status.SourceDownstream, true
			case string(ErrorSourcePlugin):
				errorSourceErr := WithErrorSource(ctx, ErrorSourcePlugin)
				if errorSourceErr != nil {
					Logger.Error("Could not set plugin error source", "error", errorSourceErr)
				}
				return status.SourcePlugin, true
			}
		}
	}
	return status.DefaultSource, false
}
