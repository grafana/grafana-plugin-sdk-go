package backend

import "context"

// TODO implement RoleRegistrer
// TODO add callback to PluginContext

type Permission struct {
	Action string
	Scope  string
}

type Role struct {
	Name        string
	UID         string
	Version     int64
	DisplayName string
	Description string
	Group       string
	Hidden      bool
	Permissions []Permission
}

type RoleRegistration struct {
	Role   Role
	Grants []string
}

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

type QueryRolesRequest struct {
	PluginContext PluginContext
}

type QueryRolesResponse struct {
	Registrations []RoleRegistration
}

type HasAccessRequest struct {
	User      User
	Evaluator Evaluator
}

type HasAccessResponse struct {
	HasAccess bool
}

type IsDisabledResponse struct {
	IsDisabled bool
}

type Void struct{}

type RegistrationHandler interface {
	QueryRoles(ctx context.Context, req *QueryRolesRequest) (*QueryRolesResponse, error)
}

type QueryRolesHandlerFunc func(ctx context.Context, req *QueryRolesRequest) (*QueryRolesResponse, error)

// QueryRoles calls fn(ctx, req).
func (fn QueryRolesHandlerFunc) QueryRoles(ctx context.Context, req *QueryRolesRequest) (*QueryRolesResponse, error) {
	return fn(ctx, req)
}

//   /* Services */
//   // Service implemented by Grafana for callbacks from the plugin
//   service AccessControl {
// 	rpc IsDisabled(Void) returns (IsDisabledResponse);
// 	rpc HasAccess(HasAccessRequest) returns (HasAccessResponse);
//   }
