package patterns

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
)

func FindSecureLiteralKeys(pkg *packages.Package, index *ast.IndexExpr) ([]model.Finding, []Pending) {
	if !IsDatasourceSecureExpr(pkg, index.X) {
		return nil, nil
	}

	if lit, ok := index.Index.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		key, err := unquote(lit.Value)
		if err != nil {
			key = lit.Value
		}

		return []model.Finding{{
			Kind:         model.DecodeKindSecureLiteral,
			Source:       model.SourceKindDatasourceSecure,
			Position:     PositionOf(pkg, index),
			FunctionName: EnclosingFunction(pkg, index),
			Key:          key,
			Destination:  DestinationForSecureAssignment(index),
			Confidence:   model.ConfidenceHigh,
		}}, nil
	}

	if pattern, ok := extractSecurePattern(pkg, index.Index, index, 3); ok {
		return []model.Finding{{
			Kind:         model.DecodeKindSecureTemplate,
			Source:       model.SourceKindDatasourceSecure,
			Position:     PositionOf(pkg, index),
			FunctionName: EnclosingFunction(pkg, index),
			Pattern:      pattern,
			Destination:  DestinationForSecureAssignment(index),
			Confidence:   model.ConfidenceMedium,
			Notes:        []string{"pattern inferred from guarded dynamic secure key access"},
		}}, nil
	}

	return nil, []Pending{{
		Kind:         PendingSecureKey,
		PackagePath:  pkg.PkgPath,
		FunctionName: EnclosingFunction(pkg, index),
		Node:         index,
		Reason:       "unable to reduce secure key expression to a literal or simple template",
	}}
}

func extractSecurePattern(pkg *packages.Package, expr ast.Expr, index *ast.IndexExpr, depth int) (string, bool) {
	if pkg == nil || expr == nil || index == nil || depth <= 0 {
		return "", false
	}

	switch typed := expr.(type) {
	case *ast.CallExpr:
		return extractTemplatePattern(pkg, typed)
	case *ast.SelectorExpr:
		return extractGuardedDynamicPattern(pkg, index, typed)
	case *ast.Ident:
		source, ok := assignedExprForIdent(pkg, typed)
		if !ok {
			return "", false
		}
		return extractSecurePattern(pkg, source, index, depth-1)
	default:
		return "", false
	}
}

func FindSecureBulkRanges(pkg *packages.Package, rng *ast.RangeStmt) ([]model.Finding, []Pending) {
	if !IsDatasourceSecureExpr(pkg, rng.X) {
		return nil, nil
	}

	return []model.Finding{{
		Kind:         model.DecodeKindSecureBulkRange,
		Source:       model.SourceKindDatasourceSecure,
		Position:     PositionOf(pkg, rng),
		FunctionName: EnclosingFunction(pkg, rng),
		Confidence:   model.ConfidenceMedium,
		Notes:        []string{"range over decrypted secure JSON data may synthesize multiple keys"},
	}}, nil
}

func extractTemplatePattern(pkg *packages.Package, call *ast.CallExpr) (string, bool) {
	if len(call.Args) == 0 {
		return "", false
	}

	if isFmtSprintfCall(pkg, call) {
		format, ok := call.Args[0].(*ast.BasicLit)
		if !ok || format.Kind != token.STRING {
			return "", false
		}

		value, err := unquote(format.Value)
		if err != nil {
			return "", false
		}

		return normalizeTemplate(value), true
	}

	return extractGuardedReplaceTemplate(pkg, call)
}

func extractGuardedDynamicPattern(pkg *packages.Package, index *ast.IndexExpr, sel *ast.SelectorExpr) (string, bool) {
	if sel == nil || sel.Sel == nil {
		return "", false
	}

	switch sel.Sel.Name {
	case "Name", "Key":
	default:
		return "", false
	}

	if !isStringExpr(pkg, index.Index) {
		return "", false
	}

	base, ok := sel.X.(*ast.Ident)
	if !ok || !isGuardedRangeValue(pkg, index, base, "Secure") {
		return "", false
	}

	return "{dynamic}", true
}

func extractGuardedReplaceTemplate(pkg *packages.Package, call *ast.CallExpr) (string, bool) {
	if pkg == nil || pkg.TypesInfo == nil || call == nil {
		return "", false
	}
	if !isPackageFuncCall(pkg.TypesInfo, call, "strings", "Replace") && !isPackageFuncCall(pkg.TypesInfo, call, "strings", "ReplaceAll") {
		return "", false
	}
	if len(call.Args) < 3 {
		return "", false
	}

	key, ok := call.Args[0].(*ast.Ident)
	if !ok {
		return "", false
	}

	oldValue, ok := stringConstantValue(pkg, call.Args[1])
	if !ok || oldValue == "" {
		return "", false
	}
	newValue, ok := stringConstantValue(pkg, call.Args[2])
	if !ok || newValue == "" {
		return "", false
	}

	marker, mode, ok := findGuardedStringConstraint(pkg, call, key)
	if !ok {
		return "", false
	}

	replaced := strings.ReplaceAll(marker, oldValue, newValue)
	switch mode {
	case "suffix":
		return "{dynamic}" + replaced, true
	default:
		return replaced + "{dynamic}", true
	}
}

func isFmtSprintfCall(pkg *packages.Package, call *ast.CallExpr) bool {
	return isPackageFuncCall(pkg.TypesInfo, call, "fmt", "Sprintf")
}

func isStringExpr(pkg *packages.Package, expr ast.Expr) bool {
	if pkg == nil || pkg.TypesInfo == nil || expr == nil {
		return false
	}
	typ := pkg.TypesInfo.TypeOf(expr)
	if typ == nil {
		return false
	}
	basicType, ok := typ.Underlying().(*types.Basic)
	return ok && basicType.Info()&types.IsString != 0
}

func isGuardedRangeValue(pkg *packages.Package, node ast.Node, ident *ast.Ident, guardField string) bool {
	if pkg == nil || pkg.TypesInfo == nil || node == nil || ident == nil {
		return false
	}

	hasGuard := false
	inRange := false
	path := enclosingPath(pkg, node)
	for _, ancestor := range path {
		switch current := ancestor.(type) {
		case *ast.IfStmt:
			if hasBoolFieldGuard(pkg, current.Cond, ident, guardField) {
				hasGuard = true
			}
		case *ast.RangeStmt:
			value, ok := current.Value.(*ast.Ident)
			if ok && sameIdentObject(pkg, value, ident) {
				inRange = true
			}
		}
	}

	return hasGuard && inRange
}

func hasBoolFieldGuard(pkg *packages.Package, expr ast.Expr, ident *ast.Ident, fieldName string) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		sel, ok := node.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != fieldName {
			return true
		}
		base, ok := sel.X.(*ast.Ident)
		if !ok || !sameIdentObject(pkg, base, ident) {
			return true
		}
		found = true
		return false
	})
	return found
}

func findGuardedStringConstraint(pkg *packages.Package, node ast.Node, ident *ast.Ident) (string, string, bool) {
	for _, ancestor := range enclosingPath(pkg, node) {
		if ifStmt, ok := ancestor.(*ast.IfStmt); ok {
			if marker, mode, ok := stringConstraintInExpr(pkg, ifStmt.Cond, ident); ok {
				return marker, mode, true
			}
		}
	}

	return "", "", false
}

func stringConstraintInExpr(pkg *packages.Package, expr ast.Expr, ident *ast.Ident) (string, string, bool) {
	var marker string
	var mode string
	ast.Inspect(expr, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}

		switch {
		case isPackageFuncCall(pkg.TypesInfo, call, "strings", "Contains"):
			mode = "contains"
		case isPackageFuncCall(pkg.TypesInfo, call, "strings", "HasPrefix"):
			mode = "prefix"
		case isPackageFuncCall(pkg.TypesInfo, call, "strings", "HasSuffix"):
			mode = "suffix"
		default:
			return true
		}

		argIdent, ok := call.Args[0].(*ast.Ident)
		if !ok || !sameIdentObject(pkg, argIdent, ident) {
			mode = ""
			return true
		}

		value, ok := stringConstantValue(pkg, call.Args[1])
		if !ok || value == "" {
			mode = ""
			return true
		}

		marker = value
		return false
	})

	return marker, mode, marker != "" && mode != ""
}

func stringConstantValue(pkg *packages.Package, expr ast.Expr) (string, bool) {
	if pkg == nil || pkg.TypesInfo == nil || expr == nil {
		return "", false
	}

	tv, ok := pkg.TypesInfo.Types[expr]
	if ok && tv.Value != nil && tv.Value.Kind() == constant.String {
		return constant.StringVal(tv.Value), true
	}

	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		value, err := unquote(lit.Value)
		if err != nil {
			return "", false
		}
		return value, true
	}

	return "", false
}

func sameIdentObject(pkg *packages.Package, left *ast.Ident, right *ast.Ident) bool {
	if pkg == nil || pkg.TypesInfo == nil || left == nil || right == nil {
		return false
	}
	return pkg.TypesInfo.ObjectOf(left) == pkg.TypesInfo.ObjectOf(right)
}

func enclosingPath(pkg *packages.Package, node ast.Node) []ast.Node {
	if pkg == nil || node == nil {
		return nil
	}

	for _, file := range pkg.Syntax {
		if node.Pos() < file.Pos() || node.End() > file.End() {
			continue
		}

		path, _ := astutil.PathEnclosingInterval(file, node.Pos(), node.End())
		if len(path) > 0 {
			return path
		}
	}

	return nil
}

func unquote(value string) (string, error) {
	if len(value) >= 2 && (value[0] == '"' || value[0] == '`') {
		return value[1 : len(value)-1], nil
	}
	return value, nil
}
