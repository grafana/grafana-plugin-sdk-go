package backend

type CallResourceRequest struct {
	Path    string
	Method  string
	URL     string
	Headers map[string][]string
	Body    []byte
}

type CallResourceResponse struct {
	Status  int
	Headers map[string][]string
	Body    []byte
}

// CallResourceResponseSender used for sending resource call responses.
type CallResourceResponseSender interface {
	Send(*CallResourceResponse) error
}

// CallResourceHandler handles resource calls.
type CallResourceHandler interface {
	CallResource(pCtx PluginContext, req *CallResourceRequest, sender CallResourceResponseSender) error
}
