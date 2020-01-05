package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type CallResourceRequest struct {
	Headers map[string]string
	Method  string
	Path    string
	Body    []byte
}

func resourceRequestFromProtobuf(req *pluginv2.CallResource_Request) *CallResourceRequest {
	return &CallResourceRequest{
		Headers: req.Headers,
		Method:  req.Method,
		Path:    req.Path,
		Body:    req.Body,
	}
}

type CallResourceResponse struct {
	Headers map[string]string
	Code    int32
	Body    []byte
}

func (rr *CallResourceResponse) toProtobuf() *pluginv2.CallResource_Response {
	return &pluginv2.CallResource_Response{
		Headers: rr.Headers,
		Code:    rr.Code,
		Body:    rr.Body,
	}
}

// ResourceHandler handles backend plugin checks.
type ResourceHandler interface {
	CallResource(ctx context.Context, pc PluginConfig, req *CallResourceRequest) (*CallResourceResponse, error)
}

func (p *coreWrapper) Resource(ctx context.Context, req *pluginv2.CallResource_Request) (*pluginv2.CallResource_Response, error) {
	pc := pluginConfigFromProto(req.Config)
	resourceReq := resourceRequestFromProtobuf(req)
	res, err := p.handlers.CallResource(ctx, pc, resourceReq)
	if err != nil {
		return nil, err
	}
	return res.toProtobuf(), nil
}
