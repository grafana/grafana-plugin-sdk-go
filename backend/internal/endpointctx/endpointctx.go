// This package has been added to expose the EndpointCtxKey to allow the datasource_metrics_middleware to read it 
package internal

type EndpointCtxKeyType struct{}

var EndpointCtxKey = EndpointCtxKeyType{}
