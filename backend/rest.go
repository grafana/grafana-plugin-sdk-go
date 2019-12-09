package backend

import (
	"context"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/grafana_plugin"
)

type RESTRequest struct {
	Headers map[string]string
	Method  string
	Path    string
	Body    []byte
}

func restRequestFromProtobuf(req *bproto.RESTRequest) *RESTRequest {
	return &RESTRequest{
		Headers: req.Headers,
		Method:  req.Method,
		Path:    req.Path,
		Body:    req.Body,
	}
}

type RESTResponse struct {
	Headers map[string]string
	Code    int32
	Body    []byte
}

func (rr *RESTResponse) toProtobuf() *bproto.RESTResponse {
	return &bproto.RESTResponse{
		Headers: rr.Headers,
		Code:    rr.Code,
		Body:    rr.Body,
	}
}

// RESTHandler handles backend plugin checks.
type RESTHandler interface {
	REST(ctx context.Context, pc PluginConfig, req *RESTRequest) (*RESTResponse, error)
}

func (p *backendPluginWrapper) REST(ctx context.Context, req *bproto.RESTRequest) (*bproto.RESTResponse, error) {
	pc := pluginConfigFromProto(req.Config)
	restReq := restRequestFromProtobuf(req)
	res, err := p.restHandler.REST(ctx, pc, restReq)
	if err != nil {
		return nil, err
	}
	return res.toProtobuf(), nil

}
