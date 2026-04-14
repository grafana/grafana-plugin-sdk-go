package querygen

import (
	"go/ast"
	"go/constant"
	"go/types"
	"reflect"
	"strings"

	v0alpha1 "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/load"
	"golang.org/x/tools/go/packages"
)

const (
	processQueriesFuncName = "processQueries"
	queryHandlerFuncName   = "queryHandler"
	queryTypeFieldName     = "queryType"
)

func inferDiscriminators(loadRes *load.Result, registration RuntimeRegistration) []v0alpha1.DiscriminatorFieldValue {
	if loadRes == nil {
		return normalizeDiscriminators(registration.Discriminators)
	}

	resolver := newHandlerResolver(loadRes)
	out := append([]v0alpha1.DiscriminatorFieldValue{}, registration.Discriminators...)

	for _, functionName := range registration.FunctionNames {
		for queryType := range resolver.queryTypesForFunction(functionName) {
			out = append(out, v0alpha1.DiscriminatorFieldValue{
				Field: queryTypeFieldName,
				Value: queryType,
			})
		}
	}

	if len(out) == 0 {
		out = append(out, inferEnumDiscriminators(loadRes, registration)...)
	}

	return normalizeDiscriminators(out)
}

type handlerResolver struct {
	load           *load.Result
	queryTypesByFn map[string]map[string]struct{}
	funcDeclByKey  map[string]funcRef
}

type funcRef struct {
	pkg  *packages.Package
	decl *ast.FuncDecl
}

func newHandlerResolver(loadRes *load.Result) *handlerResolver {
	r := &handlerResolver{
		load:           loadRes,
		queryTypesByFn: map[string]map[string]struct{}{},
		funcDeclByKey:  map[string]funcRef{},
	}
	r.indexFunctions()
	r.collectQueryTypeMappings()
	return r
}

func (r *handlerResolver) queryTypesForFunction(functionName string) map[string]struct{} {
	return r.queryTypesByFn[functionName]
}

func (r *handlerResolver) indexFunctions() {
	if r == nil || r.load == nil {
		return
	}

	for _, pkg := range r.load.Packages {
		if pkg == nil || pkg.TypesInfo == nil {
			continue
		}

		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				fnDecl, ok := decl.(*ast.FuncDecl)
				if !ok || fnDecl.Name == nil {
					continue
				}

				obj, _ := pkg.TypesInfo.Defs[fnDecl.Name].(*types.Func)
				if obj == nil {
					continue
				}

				r.funcDeclByKey[funcKey(obj)] = funcRef{
					pkg:  pkg,
					decl: fnDecl,
				}
			}
		}
	}
}

func (r *handlerResolver) collectQueryTypeMappings() {
	for _, ref := range r.funcDeclByKey {
		if ref.pkg == nil || ref.pkg.TypesInfo == nil || ref.decl == nil || ref.decl.Body == nil {
			continue
		}

		ast.Inspect(ref.decl.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}

			queryType, ok := stringConstant(ref.pkg, call.Args[0])
			if !ok || queryType == "" {
				return true
			}

			calleeName, ok := calledFunctionName(ref.pkg, call)
			if !ok || (calleeName != "Handle" && calleeName != "HandleFunc") {
				return true
			}
			if !isQueryTypeMuxCall(ref.pkg, call) {
				return true
			}

			for functionName := range r.resolveHandlerFunctions(ref.pkg, call.Args[1], 0) {
				if _, ok := r.queryTypesByFn[functionName]; !ok {
					r.queryTypesByFn[functionName] = map[string]struct{}{}
				}
				r.queryTypesByFn[functionName][queryType] = struct{}{}
			}

			return true
		})
	}
}

func isQueryTypeMuxCall(pkg *packages.Package, call *ast.CallExpr) bool {
	if pkg == nil || pkg.TypesInfo == nil || call == nil {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	selection := pkg.TypesInfo.Selections[sel]
	if selection == nil {
		return false
	}

	recv := selection.Recv()
	for {
		ptr, ok := recv.(*types.Pointer)
		if !ok {
			break
		}
		recv = ptr.Elem()
	}

	named, ok := recv.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}

	return obj.Name() == "QueryTypeMux" && obj.Pkg().Path() == "github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
}

func (r *handlerResolver) resolveHandlerFunctions(pkg *packages.Package, expr ast.Expr, depth int) map[string]struct{} {
	out := map[string]struct{}{}
	if r == nil || pkg == nil || expr == nil || depth > 6 {
		return out
	}

	if call, ok := expr.(*ast.CallExpr); ok {
		if calleeName, ok := calledFunctionName(pkg, call); ok {
			switch calleeName {
			case processQueriesFuncName:
				if len(call.Args) >= 3 {
					return r.resolveHandlerFunctions(pkg, call.Args[2], depth+1)
				}
			case queryHandlerFuncName:
				if len(call.Args) >= 2 {
					return r.resolveHandlerFunctions(pkg, call.Args[1], depth+1)
				}
			}
		}

		fnObj, fnRef, ok := r.functionForExpr(pkg, call.Fun)
		if !ok {
			return out
		}

		if fnRef.decl != nil && fnRef.decl.Body != nil {
			found := r.resolveHandlerFunctionsFromBody(fnRef, depth+1)
			if len(found) > 0 {
				return found
			}
		}

		out[formatFuncName(fnObj)] = struct{}{}
		return out
	}

	fnObj, fnRef, ok := r.functionForExpr(pkg, expr)
	if !ok {
		return out
	}

	if fnRef.decl != nil && fnRef.decl.Body != nil {
		found := r.resolveHandlerFunctionsFromBody(fnRef, depth+1)
		if len(found) > 0 {
			return found
		}
	}

	out[formatFuncName(fnObj)] = struct{}{}
	return out
}

func (r *handlerResolver) resolveHandlerFunctionsFromBody(ref funcRef, depth int) map[string]struct{} {
	out := map[string]struct{}{}
	if ref.pkg == nil || ref.decl == nil || ref.decl.Body == nil {
		return out
	}

	ast.Inspect(ref.decl.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		calleeName, ok := calledFunctionName(ref.pkg, call)
		if !ok {
			return true
		}

		switch calleeName {
		case processQueriesFuncName:
			if len(call.Args) >= 3 {
				mergeFunctionSets(out, r.resolveHandlerFunctions(ref.pkg, call.Args[2], depth+1))
			}
		case queryHandlerFuncName:
			if len(call.Args) >= 2 {
				mergeFunctionSets(out, r.resolveHandlerFunctions(ref.pkg, call.Args[1], depth+1))
			}
		}

		return true
	})

	if len(out) > 0 {
		return out
	}

	for _, stmt := range ref.decl.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok {
			continue
		}
		for _, result := range ret.Results {
			mergeFunctionSets(out, r.resolveReturnedHandlerFunctions(ref.pkg, result, depth+1))
		}
	}

	return out
}

func (r *handlerResolver) resolveReturnedHandlerFunctions(pkg *packages.Package, expr ast.Expr, depth int) map[string]struct{} {
	out := map[string]struct{}{}
	if r == nil || pkg == nil || expr == nil || depth > 6 {
		return out
	}

	switch typed := expr.(type) {
	case *ast.CallExpr:
		calleeName, ok := calledFunctionName(pkg, typed)
		if !ok {
			return out
		}
		switch calleeName {
		case processQueriesFuncName, queryHandlerFuncName:
			return r.resolveHandlerFunctions(pkg, typed, depth+1)
		default:
			return out
		}
	case *ast.Ident, *ast.SelectorExpr:
		_, fnRef, ok := r.functionForExpr(pkg, typed)
		if !ok || fnRef.decl == nil || fnRef.decl.Body == nil {
			return out
		}
		return r.resolveHandlerFunctions(pkg, typed, depth+1)
	default:
		return out
	}
}

func (r *handlerResolver) functionForExpr(pkg *packages.Package, expr ast.Expr) (*types.Func, funcRef, bool) {
	if pkg == nil || pkg.TypesInfo == nil || expr == nil {
		return nil, funcRef{}, false
	}

	switch typed := expr.(type) {
	case *ast.Ident:
		fn, ok := pkg.TypesInfo.Uses[typed].(*types.Func)
		if !ok {
			return nil, funcRef{}, false
		}
		return fn, r.funcDeclByKey[funcKey(fn)], true
	case *ast.SelectorExpr:
		fn, ok := pkg.TypesInfo.Uses[typed.Sel].(*types.Func)
		if !ok {
			return nil, funcRef{}, false
		}
		return fn, r.funcDeclByKey[funcKey(fn)], true
	default:
		return nil, funcRef{}, false
	}
}

func funcKey(fn *types.Func) string {
	if fn == nil {
		return ""
	}
	if fn.Pkg() == nil {
		return formatFuncName(fn)
	}
	return fn.Pkg().Path() + "\x00" + formatFuncName(fn)
}

func formatFuncName(fn *types.Func) string {
	if fn == nil {
		return ""
	}

	sig, _ := fn.Type().(*types.Signature)
	if sig != nil && sig.Recv() != nil {
		return "(" + types.TypeString(sig.Recv().Type(), types.RelativeTo(fn.Pkg())) + ")." + fn.Name()
	}

	return fn.Name()
}

func stringConstant(pkg *packages.Package, expr ast.Expr) (string, bool) {
	if pkg == nil || pkg.TypesInfo == nil || expr == nil {
		return "", false
	}

	tv, ok := pkg.TypesInfo.Types[expr]
	if ok && tv.Value != nil && tv.Value.Kind() == constant.String {
		return constant.StringVal(tv.Value), true
	}

	switch typed := expr.(type) {
	case *ast.Ident:
		if c, ok := pkg.TypesInfo.Uses[typed].(*types.Const); ok && c.Val().Kind() == constant.String {
			return constant.StringVal(c.Val()), true
		}
	case *ast.SelectorExpr:
		if c, ok := pkg.TypesInfo.Uses[typed.Sel].(*types.Const); ok && c.Val().Kind() == constant.String {
			return constant.StringVal(c.Val()), true
		}
	}

	return "", false
}

func calledFunctionName(pkg *packages.Package, call *ast.CallExpr) (string, bool) {
	if pkg == nil || pkg.TypesInfo == nil || call == nil {
		return "", false
	}

	switch fun := call.Fun.(type) {
	case *ast.Ident:
		if fn, ok := pkg.TypesInfo.Uses[fun].(*types.Func); ok {
			return fn.Name(), true
		}
	case *ast.SelectorExpr:
		if fn, ok := pkg.TypesInfo.Uses[fun.Sel].(*types.Func); ok {
			return fn.Name(), true
		}
	}

	return "", false
}

func mergeFunctionSets(dst map[string]struct{}, src map[string]struct{}) {
	for item := range src {
		dst[item] = struct{}{}
	}
}

func inferEnumDiscriminators(loadRes *load.Result, registration RuntimeRegistration) []v0alpha1.DiscriminatorFieldValue {
	target := registration.Target
	if target == nil || target.PackagePath == "" || target.TypeName == "" || loadRes == nil {
		return nil
	}

	var packageRef *packages.Package
	for _, pkg := range loadRes.Packages {
		if pkg != nil && pkg.Types != nil && pkg.PkgPath == target.PackagePath {
			packageRef = pkg
			break
		}
	}
	if packageRef == nil {
		return nil
	}

	obj := packageRef.Types.Scope().Lookup(target.TypeName)
	typeName, ok := obj.(*types.TypeName)
	if !ok {
		return nil
	}
	named, ok := typeName.Type().(*types.Named)
	if !ok {
		return nil
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil
	}

	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if field == nil {
			continue
		}

		fieldName, hasTag := queryJSONFieldName(st.Tag(i))
		if fieldName != queryTypeFieldName && (fieldName != "" || hasTag || field.Name() != "QueryType") {
			continue
		}

		enumType, ok := field.Type().(*types.Named)
		if !ok {
			return nil
		}

		values := enumStringValues(packageRef, enumType)
		out := make([]v0alpha1.DiscriminatorFieldValue, 0, len(values))
		for _, value := range values {
			out = append(out, v0alpha1.DiscriminatorFieldValue{
				Field: queryTypeFieldName,
				Value: value,
			})
		}
		return out
	}

	return nil
}

func queryJSONFieldName(tag string) (string, bool) {
	jsonTag := reflect.StructTag(tag).Get("json")
	if jsonTag == "" {
		return "", false
	}
	parts := strings.Split(jsonTag, ",")
	if len(parts) == 0 || parts[0] == "-" || parts[0] == "" {
		return "", false
	}
	return parts[0], true
}

func enumStringValues(pkg *packages.Package, named *types.Named) []string {
	if pkg == nil || pkg.Types == nil || named == nil || named.Obj() == nil {
		return nil
	}

	values := make([]string, 0)
	seen := map[string]struct{}{}
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		constObj, ok := obj.(*types.Const)
		if !ok {
			continue
		}
		if !types.Identical(constObj.Type(), named) || constObj.Val().Kind() != constant.String {
			continue
		}

		value := constant.StringVal(constObj.Val())
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}

	return values
}
