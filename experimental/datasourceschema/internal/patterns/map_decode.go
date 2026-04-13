package patterns

import (
	"go/ast"

	"golang.org/x/tools/go/packages"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
)

func FindMapstructureDecodeTargets(pkg *packages.Package, call *ast.CallExpr) ([]model.Finding, []Pending) {
	if !IsMapstructureDecodeCall(pkg.TypesInfo, call) || len(call.Args) < 2 {
		return nil, nil
	}

	source := call.Args[0]
	if !IsDatasourceSecureExpr(pkg, source) {
		return nil, nil
	}

	targetRef, ok := ResolveTargetType(pkg, call.Args[1])
	if !ok {
		if !ShouldAttemptTargetResolution(pkg, call.Args[1]) {
			return nil, nil
		}

		return nil, []Pending{{
			Kind:         PendingDecodeTarget,
			PackagePath:  pkg.PkgPath,
			FunctionName: EnclosingFunction(pkg, call),
			Node:         call,
			Reason:       "unable to resolve mapstructure decode target type from typed AST",
		}}
	}

	return []model.Finding{{
		Kind:         model.DecodeKindMapstructure,
		Source:       model.SourceKindDatasourceSecure,
		Position:     PositionOf(pkg, call),
		FunctionName: EnclosingFunction(pkg, call),
		Target:       targetRef,
		Confidence:   model.ConfidenceHigh,
	}}, nil
}
