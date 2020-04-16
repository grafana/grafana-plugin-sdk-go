package backend

type TransformHandlers interface {
	TransformDataHandler
}

type TransformDataHandler interface {
	TransformData(pCtx PluginContext, req *QueryDataRequest, callBack TransformDataCallBackHandler) (*QueryDataResponse, error)
}

// Callback

type TransformDataCallBackHandler interface {
	// TODO: Forget if I actually need PluginConfig on the callback or not.
	QueryData(pCtx PluginContext, req *QueryDataRequest) (*QueryDataResponse, error)
}
