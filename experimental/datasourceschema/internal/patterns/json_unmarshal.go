package patterns

import (
	"go/ast"

	"golang.org/x/tools/go/packages"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
)

func FindJSONUnmarshalTargets(pkg *packages.Package, call *ast.CallExpr) ([]model.Finding, []Pending) {
	if !IsJSONUnmarshalCall(pkg.TypesInfo, call) || len(call.Args) < 2 {
		return nil, nil
	}

	var findings []model.Finding
	var pending []Pending

	sourceKind, ok := SourceKindForJSONExpr(pkg, call.Args[0])
	if !ok {
		sourceKind, ok = EnclosingDecodeSourceKind(pkg, call)
		if !ok || !isStringOrBytesType(pkg.TypesInfo.TypeOf(call.Args[0])) || !looksLikeJSONBufferExpr(call.Args[0]) {
			return nil, nil
		}
	}
	target := call.Args[1]

	targetRef, ok := ResolveTargetType(pkg, target)
	if !ok {
		if !ShouldAttemptTargetResolution(pkg, target) {
			return nil, nil
		}

		pending = append(pending, Pending{
			Kind:         PendingDecodeTarget,
			PackagePath:  pkg.PkgPath,
			FunctionName: EnclosingFunction(pkg, call),
			Node:         call,
			Reason:       "unable to resolve unmarshal target type from typed AST",
		})
		return nil, pending
	}

	findings = append(findings, model.Finding{
		Kind:         model.DecodeKindJSONUnmarshal,
		Source:       sourceKind,
		Position:     PositionOf(pkg, call),
		FunctionName: EnclosingFunction(pkg, call),
		Target:       targetRef,
		Confidence:   model.ConfidenceHigh,
	})

	return findings, pending
}
