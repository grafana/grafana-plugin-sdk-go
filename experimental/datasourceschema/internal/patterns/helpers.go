package patterns

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
)

const (
	backendPackagePath = "github.com/grafana/grafana-plugin-sdk-go/backend"
)

func IsJSONUnmarshalCall(info *types.Info, call *ast.CallExpr) bool {
	return isPackageFuncCall(info, call, "encoding/json", "Unmarshal")
}

func IsMapstructureDecodeCall(info *types.Info, call *ast.CallExpr) bool {
	return isPackageFuncCall(info, call, "github.com/mitchellh/mapstructure", "Decode")
}

func IsDatasourceJSONExpr(info *types.Info, expr ast.Expr) bool {
	return isBackendFieldSelector(info, expr, "DataSourceInstanceSettings", "JSONData")
}

func IsDatasourceSecureExpr(pkg *packages.Package, expr ast.Expr) bool {
	return isDatasourceSecureExprDepth(pkg, expr, 4)
}

func IsQueryJSONExpr(info *types.Info, expr ast.Expr) bool {
	return isBackendFieldSelector(info, expr, "DataQuery", "JSON")
}

func SourceKindForJSONExpr(pkg *packages.Package, expr ast.Expr) (model.SourceKind, bool) {
	return sourceKindForJSONExpr(pkg, expr, 8)
}

func ResolveTargetType(pkg *packages.Package, expr ast.Expr) (*model.TargetRef, bool) {
	targetExpr := expr
	pointer := false
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		targetExpr = unary.X
		pointer = true
	}

	targetType := pkg.TypesInfo.TypeOf(targetExpr)
	if targetType == nil {
		return nil, false
	}
	if ptr, ok := targetType.(*types.Pointer); ok {
		pointer = true
		targetType = ptr.Elem()
	}
	if _, ok := targetType.Underlying().(*types.Struct); !ok {
		return nil, false
	}

	named, ok := targetType.(*types.Named)
	if ok {
		obj := named.Obj()
		if obj == nil || obj.Pkg() == nil {
			return nil, false
		}

		return &model.TargetRef{
			PackagePath: obj.Pkg().Path(),
			TypeName:    obj.Name(),
			Pointer:     pointer,
			TypeString:  types.TypeString(targetType, nil),
			Expr:        positionPtr(PositionOf(pkg, expr)),
		}, true
	}

	if targetType == nil {
		return nil, false
	}

	return &model.TargetRef{
		PackagePath: pkg.PkgPath,
		Pointer:     pointer,
		TypeString:  types.TypeString(targetType, nil),
		Expr:        positionPtr(PositionOf(pkg, expr)),
	}, true
}

func ShouldAttemptTargetResolution(pkg *packages.Package, expr ast.Expr) bool {
	if pkg == nil || pkg.TypesInfo == nil || expr == nil {
		return true
	}

	targetExpr := expr
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		targetExpr = unary.X
	}

	targetType := pkg.TypesInfo.TypeOf(targetExpr)
	if targetType == nil {
		return true
	}

	if ptr, ok := targetType.(*types.Pointer); ok {
		targetType = ptr.Elem()
	}

	switch targetType.Underlying().(type) {
	case *types.Struct, *types.Interface:
		return true
	default:
		return false
	}
}

func PositionOf(pkg *packages.Package, node ast.Node) model.Position {
	if pkg == nil || pkg.Fset == nil || node == nil {
		return model.Position{}
	}

	pos := pkg.Fset.PositionFor(node.Pos(), false)
	return model.Position{
		File:   pos.Filename,
		Line:   pos.Line,
		Column: pos.Column,
	}
}

func EnclosingFunction(pkg *packages.Package, node ast.Node) string {
	if pkg == nil || node == nil {
		return ""
	}

	for _, file := range pkg.Syntax {
		if node.Pos() < file.Pos() || node.End() > file.End() {
			continue
		}

		var closest ast.Node
		ast.Inspect(file, func(current ast.Node) bool {
			if current == nil {
				return true
			}
			switch current.(type) {
			case *ast.FuncDecl, *ast.FuncLit:
				if current.Pos() <= node.Pos() && node.End() <= current.End() {
					if closest == nil || (current.End()-current.Pos()) < (closest.End()-closest.Pos()) {
						closest = current
					}
				}
			}
			return true
		})

		switch fn := closest.(type) {
		case *ast.FuncDecl:
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				recv := pkg.TypesInfo.TypeOf(fn.Recv.List[0].Type)
				return fmt.Sprintf("(%s).%s", types.TypeString(recv, types.RelativeTo(pkg.Types)), fn.Name.Name)
			}
			return fn.Name.Name
		case *ast.FuncLit:
			pos := PositionOf(pkg, fn)
			return fmt.Sprintf("func@%s:%d", pos.File, pos.Line)
		}
	}

	return ""
}

func EnclosingDecodeSourceKind(pkg *packages.Package, node ast.Node) (model.SourceKind, bool) {
	if pkg == nil || node == nil {
		return "", false
	}

	for _, file := range pkg.Syntax {
		if node.Pos() < file.Pos() || node.End() > file.End() {
			continue
		}

		var closest *ast.FuncDecl
		ast.Inspect(file, func(current ast.Node) bool {
			fn, ok := current.(*ast.FuncDecl)
			if !ok {
				return true
			}
			if fn.Pos() <= node.Pos() && node.End() <= fn.End() {
				if closest == nil || (fn.End()-fn.Pos()) < (closest.End()-closest.Pos()) {
					closest = fn
				}
			}
			return true
		})

		if closest == nil || closest.Type == nil || closest.Type.Params == nil {
			return "", false
		}

		for _, field := range closest.Type.Params.List {
			paramType := pkg.TypesInfo.TypeOf(field.Type)
			switch {
			case isBackendNamedType(paramType, "DataQuery"):
				return model.SourceKindQueryJSON, true
			case isBackendNamedType(paramType, "DataSourceInstanceSettings"):
				return model.SourceKindDatasourceJSON, true
			}
		}
	}

	return "", false
}

func DestinationForSecureAssignment(index *ast.IndexExpr) string {
	parent := unwrapParens(index)
	assign, ok := parent.(*ast.AssignStmt)
	if ok {
		for i, rhs := range assign.Rhs {
			if rhs == index && i < len(assign.Lhs) {
				return exprString(assign.Lhs[i])
			}
		}
	}

	if ifStmt, ok := parent.(*ast.IfStmt); ok && ifStmt.Init != nil {
		if assign, ok := ifStmt.Init.(*ast.AssignStmt); ok {
			for i, rhs := range assign.Rhs {
				if rhs == index && i < len(assign.Lhs) {
					return exprString(assign.Lhs[i])
				}
			}
		}
	}

	return ""
}

func exprString(expr ast.Expr) string {
	var b strings.Builder
	_ = formatNode(&b, expr)
	return b.String()
}

func formatNode(b *strings.Builder, node ast.Node) error {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.Ident:
		b.WriteString(n.Name)
	case *ast.SelectorExpr:
		_ = formatNode(b, n.X)
		b.WriteString(".")
		b.WriteString(n.Sel.Name)
	case *ast.StarExpr:
		b.WriteString("*")
		_ = formatNode(b, n.X)
	case *ast.IndexExpr:
		_ = formatNode(b, n.X)
		b.WriteString("[")
		_ = formatNode(b, n.Index)
		b.WriteString("]")
	default:
		return fmt.Errorf("unsupported expression type %T", node)
	}

	return nil
}

func unwrapParens(node ast.Node) ast.Node {
	return node
}

func positionPtr(pos model.Position) *model.Position {
	if pos == (model.Position{}) {
		return nil
	}
	return &pos
}

func isPackageFuncCall(info *types.Info, call *ast.CallExpr, pkgPath string, funcName string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	obj := info.Uses[sel.Sel]
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil {
		return false
	}

	return fn.Name() == funcName && fn.Pkg().Path() == pkgPath
}

func isBackendFieldSelector(info *types.Info, expr ast.Expr, typeName string, fieldName string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != fieldName {
		return false
	}

	typ := info.TypeOf(sel.X)
	if typ == nil {
		return false
	}

	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}

	return obj.Name() == typeName && obj.Pkg().Path() == backendPackagePath
}

func isDatasourceSecureExprDepth(pkg *packages.Package, expr ast.Expr, depth int) bool {
	if pkg == nil || pkg.TypesInfo == nil || expr == nil || depth <= 0 {
		return false
	}

	if isBackendFieldSelector(pkg.TypesInfo, expr, "DataSourceInstanceSettings", "DecryptedSecureJSONData") {
		return true
	}
	if sel, ok := expr.(*ast.SelectorExpr); ok && sel.Sel != nil && sel.Sel.Name == "DecryptedSecureJSONData" && isStringStringMapType(pkg.TypesInfo.TypeOf(expr)) {
		return true
	}

	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}

	source, ok := assignedExprForIdent(pkg, ident)
	if !ok {
		return false
	}

	return isDatasourceSecureExprDepth(pkg, source, depth-1)
}

func assignedExprForIdent(pkg *packages.Package, ident *ast.Ident) (ast.Expr, bool) {
	if pkg == nil || pkg.TypesInfo == nil || ident == nil {
		return nil, false
	}

	obj := pkg.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return nil, false
	}

	for _, file := range pkg.Syntax {
		var found ast.Expr
		ast.Inspect(file, func(node ast.Node) bool {
			if found != nil {
				return false
			}

			switch current := node.(type) {
			case *ast.AssignStmt:
				for i, lhs := range current.Lhs {
					lhsIdent, ok := lhs.(*ast.Ident)
					if !ok || pkg.TypesInfo.ObjectOf(lhsIdent) != obj {
						continue
					}
					if len(current.Rhs) == 1 {
						found = current.Rhs[0]
						return false
					}
					if i < len(current.Rhs) {
						found = current.Rhs[i]
						return false
					}
				}
			case *ast.ValueSpec:
				for i, name := range current.Names {
					if pkg.TypesInfo.ObjectOf(name) != obj {
						continue
					}
					if len(current.Values) == 1 {
						found = current.Values[0]
						return false
					}
					if i < len(current.Values) {
						found = current.Values[i]
						return false
					}
				}
			}

			return true
		})

		if found != nil {
			return found, true
		}
	}

	return nil, false
}

func normalizeTemplate(format string) string {
	replacer := strings.NewReplacer("%s", "{dynamic}", "%d", "{dynamic}", "%v", "{dynamic}")
	return replacer.Replace(format)
}

func sourceKindForJSONExpr(pkg *packages.Package, expr ast.Expr, depth int) (model.SourceKind, bool) {
	if pkg == nil || pkg.TypesInfo == nil || expr == nil || depth <= 0 {
		return "", false
	}

	switch typed := expr.(type) {
	case *ast.ParenExpr:
		return sourceKindForJSONExpr(pkg, typed.X, depth-1)
	case *ast.UnaryExpr:
		return sourceKindForJSONExpr(pkg, typed.X, depth-1)
	case *ast.SelectorExpr:
		switch {
		case IsDatasourceJSONExpr(pkg.TypesInfo, typed):
			return model.SourceKindDatasourceJSON, true
		case IsQueryJSONExpr(pkg.TypesInfo, typed):
			return model.SourceKindQueryJSON, true
		default:
			return "", false
		}
	case *ast.SliceExpr:
		return sourceKindForJSONExpr(pkg, typed.X, depth-1)
	case *ast.CallExpr:
		for _, arg := range typed.Args {
			if sourceKind, ok := sourceKindForJSONExpr(pkg, arg, depth-1); ok {
				return sourceKind, true
			}
		}
	case *ast.Ident:
		if source, ok := assignedExprForIdent(pkg, typed); ok {
			if sourceKind, ok := sourceKindForJSONExpr(pkg, source, depth-1); ok {
				return sourceKind, true
			}
		}
		return sourceKindFromDecodeTarget(pkg, typed, depth-1)
	}

	return "", false
}

func sourceKindFromDecodeTarget(pkg *packages.Package, ident *ast.Ident, depth int) (model.SourceKind, bool) {
	if pkg == nil || pkg.TypesInfo == nil || ident == nil || depth <= 0 {
		return "", false
	}

	for _, file := range pkg.Syntax {
		var out model.SourceKind
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 || !IsJSONUnmarshalCall(pkg.TypesInfo, call) {
				return true
			}
			if !jsonDecodeTargetMatchesIdent(pkg, call.Args[1], ident) {
				return true
			}

			sourceKind, ok := sourceKindForJSONExpr(pkg, call.Args[0], depth-1)
			if !ok {
				return true
			}
			out = sourceKind
			return false
		})
		if out != "" {
			return out, true
		}
	}

	return "", false
}

func jsonDecodeTargetMatchesIdent(pkg *packages.Package, expr ast.Expr, ident *ast.Ident) bool {
	if pkg == nil || ident == nil || expr == nil {
		return false
	}

	switch typed := expr.(type) {
	case *ast.Ident:
		return sameIdentObjectRef(pkg, typed, ident)
	case *ast.UnaryExpr:
		return typed.Op == token.AND && jsonDecodeTargetMatchesIdent(pkg, typed.X, ident)
	default:
		return false
	}
}

func sameIdentObjectRef(pkg *packages.Package, left *ast.Ident, right *ast.Ident) bool {
	if pkg == nil || pkg.TypesInfo == nil || left == nil || right == nil {
		return false
	}
	return pkg.TypesInfo.ObjectOf(left) == pkg.TypesInfo.ObjectOf(right)
}

func isStringStringMapType(typ types.Type) bool {
	if typ == nil {
		return false
	}

	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	m, ok := typ.Underlying().(*types.Map)
	if !ok {
		return false
	}

	key, ok := m.Key().Underlying().(*types.Basic)
	if !ok || key.Kind() != types.String {
		return false
	}

	elem, ok := m.Elem().Underlying().(*types.Basic)
	return ok && elem.Kind() == types.String
}

func isStringOrBytesType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	if basic, ok := typ.Underlying().(*types.Basic); ok {
		return basic.Kind() == types.String
	}
	if slice, ok := typ.Underlying().(*types.Slice); ok {
		elem, ok := slice.Elem().Underlying().(*types.Basic)
		return ok && elem.Kind() == types.Byte
	}
	return false
}

func isBackendNamedType(typ types.Type, typeName string) bool {
	if typ == nil {
		return false
	}
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	named, ok := typ.(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}

	return named.Obj().Name() == typeName && named.Obj().Pkg().Path() == backendPackagePath
}

func looksLikeJSONBufferExpr(expr ast.Expr) bool {
	if expr == nil {
		return false
	}
	return strings.Contains(strings.ToLower(exprString(expr)), "json")
}

func NormalizeTemplate(format string) string {
	return normalizeTemplate(format)
}
