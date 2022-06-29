package backend

import "context"

// TODO add callback to PluginContext

type Evaluator interface{}

type PermissionEvaluator struct {
	Action string
	Scopes []string
}

type AnyEvaluator struct {
	AnyOf []Evaluator
}

type AllEvaluator struct {
	AllOf []Evaluator
}
type HasAccessRequest struct {
	User      *User
	Evaluator Evaluator
}

type HasAccessResponse struct {
	HasAccess bool
}

// type IsDisabledResponse struct {
// 	IsDisabled bool
// }

// type Void struct{}

// type IsDisabledHandler interface {
// 	IsDisabled(ctx context.Context, void *Void) (*IsDisabledResponse, error)
// }

type HasAccessHandler interface {
	HasAccess(ctx context.Context, has *HasAccessRequest) (*HasAccessResponse, error)
}

type AccessControl interface {
	// IsDisabledHandler
	HasAccessHandler
}

// HasAccessFunc is an adapter to allow the use of
// ordinary functions as backend.HasAccessHandler. If f is a function
// with the appropriate signature, HasAccessHandlerFunc(f) is a
// Handler that calls f.
type HasAccessFunc func(ctx context.Context, has *HasAccessRequest) (*HasAccessResponse, error)

// HasAccess calls fn(ctx, req).
func (fn HasAccessFunc) HasAccess(ctx context.Context, has *HasAccessRequest) (*HasAccessResponse, error) {
	return fn(ctx, has)
}

// // IsDisabledFunc is an adapter to allow the use of
// // ordinary functions as backend.IsDisabledHandler. If f is a function
// // with the appropriate signature, IsDisabledHandlerFunc(f) is a
// // Handler that calls f.
// type IsDisabledFunc func(ctx context.Context, void *Void) (*IsDisabledResponse, error)

// // IsDisabled calls fn(ctx, req).
// func (fn IsDisabledFunc) IsDisabled(ctx context.Context, void *Void) (*IsDisabledResponse, error) {
// 	return fn(ctx, void)
// }
