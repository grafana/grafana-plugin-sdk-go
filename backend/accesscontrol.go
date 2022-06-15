package backend

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

//   /* Services */
//   // Service implemented by Grafana for callbacks from the plugin
//   service AccessControl {
// 	rpc IsDisabled(Void) returns (IsDisabledResponse);
// 	rpc HasAccess(HasAccessRequest) returns (HasAccessResponse);
//   }
