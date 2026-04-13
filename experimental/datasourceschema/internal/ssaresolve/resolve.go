package ssaresolve

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/load"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/patterns"
)

type Resolver struct {
	Load           *load.Result
	Prog           *ssa.Program
	allFunctions   map[*ssa.Function]bool
	methodsByNamed map[string][]*ssa.Function
	localModuleDir map[string]struct{}
}

type frameworkQueryCandidate struct {
	PkgPath     string
	Function    *ssa.Function
	Target      *model.TargetRef
	Position    model.Position
	Score       int
	FromDecode  bool
	Description string
}

type frameworkSeed struct {
	Function *ssa.Function
	Position model.Position
}

type DataSourceSettingsUsage struct {
	UsesURL         bool
	UsesHTTPOptions bool
}

type localQueryCandidate struct {
	Target      *model.TargetRef
	Position    model.Position
	Score       int
	Description string
}

func Build(loadRes *load.Result) (*Resolver, error) {
	if loadRes == nil {
		return nil, fmt.Errorf("load result is nil")
	}

	prog, pkgs := ssautil.Packages(loadRes.Packages, ssa.InstantiateGenerics)
	for _, pkg := range pkgs {
		if pkg != nil {
			pkg.SetDebugMode(true)
		}
	}
	prog.Build()

	return &Resolver{
		Load:           loadRes,
		Prog:           prog,
		allFunctions:   ssautil.AllFunctions(prog),
		methodsByNamed: map[string][]*ssa.Function{},
		localModuleDir: localModuleDirs(loadRes),
	}, nil
}

func localModuleDirs(loadRes *load.Result) map[string]struct{} {
	dirs := map[string]struct{}{}
	if loadRes == nil {
		return dirs
	}

	for _, pkg := range loadRes.RootPackages {
		if pkg == nil || pkg.Module == nil || pkg.Module.Dir == "" {
			continue
		}
		if absDir, ok := absPath(pkg.Module.Dir); ok {
			dirs[absDir] = struct{}{}
		}
	}

	return dirs
}

func (r *Resolver) Resolve(pending []patterns.Pending) ([]model.Finding, []model.Warning, error) {
	var findings []model.Finding
	var warnings []model.Warning

	for _, item := range pending {
		switch item.Kind {
		case patterns.PendingDecodeTarget:
			f, w := r.resolveDecodeTarget(item)
			findings = append(findings, f...)
			warnings = append(warnings, w...)
		case patterns.PendingSecureKey:
			f, w := r.resolveSecureKey(item)
			findings = append(findings, f...)
			warnings = append(warnings, w...)
		}
	}

	return findings, warnings, nil
}

func (r *Resolver) InferLocalQueryTargets() ([]model.Finding, []model.Warning) {
	if r == nil || r.Load == nil {
		return nil, nil
	}

	candidates := make([]localQueryCandidate, 0)
	for _, pkg := range r.Load.Packages {
		if !r.isLocalPackage(pkg) || pkg == nil || pkg.TypesInfo == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				calleePkg, calleeDecl, calleeName, ok := r.staticCalleeDecl(pkg, call)
				if !ok {
					return true
				}

				for _, paramIndex := range queryJSONArgIndices(pkg, call) {
					target, depth, ok := r.localQueryTargetFromParam(calleePkg, calleeDecl, paramIndex, map[string]struct{}{}, 8)
					if !ok || target == nil {
						continue
					}
					candidates = append(candidates, localQueryCandidate{
						Target:      target,
						Position:    patterns.PositionOf(pkg, call),
						Score:       1000 - depth,
						Description: calleePkg.PkgPath + "." + calleeName,
					})
				}

				return true
			})
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	sort.SliceStable(candidates, func(i int, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].Description < candidates[j].Description
	})

	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if sameTarget(*best.Target, *candidate.Target) {
			continue
		}
		return nil, []model.Warning{{
			Position: best.Position,
			Code:     "ssa_local_query_ambiguous",
			Message:  fmt.Sprintf("multiple local query targets inferred via wrapper analysis: %s and %s", describeLocalQueryCandidate(best), describeLocalQueryCandidate(candidate)),
		}}
	}

	return []model.Finding{{
		Kind:       model.DecodeKindJSONUnmarshal,
		Source:     model.SourceKindQueryJSON,
		Position:   best.Position,
		Target:     best.Target,
		Confidence: model.ConfidenceLow,
		Notes:      []string{fmt.Sprintf("query target inferred via local wrapper analysis from %s", describeLocalQueryCandidate(best))},
	}}, nil
}

func (r *Resolver) InferFrameworkQueryTargets() ([]model.Finding, []model.Warning) {
	if r == nil || r.Load == nil || r.Prog == nil {
		return nil, nil
	}

	seeds := r.frameworkSeedsInUse()
	if len(seeds) == 0 {
		return nil, nil
	}

	candidates := make([]frameworkQueryCandidate, 0)
	for _, seed := range seeds {
		candidate, ok := r.bestFrameworkQueryCandidate(seed)
		if !ok {
			continue
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	sort.SliceStable(candidates, func(i int, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		if candidates[i].PkgPath != candidates[j].PkgPath {
			return candidates[i].PkgPath < candidates[j].PkgPath
		}
		return candidates[i].Description < candidates[j].Description
	})

	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if sameTarget(*best.Target, *candidate.Target) {
			continue
		}
		if best.Score-candidate.Score >= 300 {
			continue
		}
		return nil, []model.Warning{{
			Position: best.Position,
			Code:     "ssa_framework_query_ambiguous",
			Message:  fmt.Sprintf("multiple framework query targets inferred via SSA; keeping analysis-based results only: %s and %s", describeFrameworkCandidate(best), describeFrameworkCandidate(candidate)),
		}}
	}

	return []model.Finding{{
		Kind:       model.DecodeKindJSONUnmarshal,
		Source:     model.SourceKindQueryJSON,
		Position:   best.Position,
		Target:     best.Target,
		Confidence: model.ConfidenceLow,
		Notes:      []string{fmt.Sprintf("query target inferred via SSA framework analysis from %s", describeFrameworkCandidate(best))},
	}}, nil
}

func (r *Resolver) InferFrameworkDataSourceSettingsUsage() DataSourceSettingsUsage {
	if r == nil || r.Load == nil || r.Prog == nil {
		return DataSourceSettingsUsage{}
	}

	seeds := r.frameworkSeedsInUse()
	if len(seeds) == 0 {
		return DataSourceSettingsUsage{}
	}

	usage := DataSourceSettingsUsage{}
	seen := map[*ssa.Function]struct{}{}
	for _, seed := range seeds {
		functions := r.reachableFrameworkFunctions([]*ssa.Function{seed.Function}, 16)
		if len(functions) == 0 {
			functions = r.functionsInPackage(seed.Function.Pkg.Pkg.Path())
		} else {
			functions = append(functions, r.functionsInPackage(seed.Function.Pkg.Pkg.Path())...)
		}

		for _, fn := range functions {
			if fn == nil {
				continue
			}
			if _, ok := seen[fn]; ok {
				continue
			}
			seen[fn] = struct{}{}
			usage.merge(r.dataSourceSettingsUsageInFunction(fn))
			if usage.UsesURL && usage.UsesHTTPOptions {
				return usage
			}
		}
	}

	return usage
}

func queryJSONArgIndices(pkg *packages.Package, call *ast.CallExpr) []int {
	indices := make([]int, 0)
	for i, arg := range call.Args {
		if sourceKind, ok := patterns.SourceKindForJSONExpr(pkg, arg); ok && sourceKind == model.SourceKindQueryJSON {
			indices = append(indices, i)
		}
	}
	return indices
}

func (r *Resolver) staticCalleeDecl(pkg *packages.Package, call *ast.CallExpr) (*packages.Package, ast.Node, string, bool) {
	if pkg == nil || pkg.TypesInfo == nil || call == nil {
		return nil, nil, "", false
	}

	var obj types.Object
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		obj = pkg.TypesInfo.Uses[fun]
	case *ast.SelectorExpr:
		obj = pkg.TypesInfo.Uses[fun.Sel]
	}
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil {
		return nil, nil, "", false
	}

	calleePkg := r.findPackage(fn.Pkg().Path())
	if calleePkg == nil {
		return nil, nil, "", false
	}

	decl, ok := findFuncDeclByObject(calleePkg, fn)
	if !ok {
		return nil, nil, "", false
	}

	return calleePkg, decl, fn.Name(), true
}

func findFuncDeclByObject(pkg *packages.Package, target *types.Func) (ast.Node, bool) {
	if pkg == nil || pkg.TypesInfo == nil || target == nil {
		return nil, false
	}

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if obj, ok := pkg.TypesInfo.Defs[fn.Name].(*types.Func); ok && obj == target {
				return fn, true
			}
		}
	}

	return nil, false
}

func (r *Resolver) localQueryTargetFromParam(pkg *packages.Package, fnNode ast.Node, paramIndex int, visited map[string]struct{}, depth int) (*model.TargetRef, int, bool) {
	if pkg == nil || fnNode == nil || depth <= 0 {
		return nil, 0, false
	}

	paramObj, body, ok := functionParamObject(pkg, fnNode, paramIndex)
	if !ok || paramObj == nil || body == nil {
		return nil, 0, false
	}

	pos := patterns.PositionOf(pkg, fnNode)
	key := fmt.Sprintf("%s:%s:%d:%d:%d", pkg.PkgPath, pos.File, pos.Line, pos.Column, paramIndex)
	if _, ok := visited[key]; ok {
		return nil, 0, false
	}
	visited[key] = struct{}{}

	bestDepth := 0
	var bestTarget *model.TargetRef
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		if patterns.IsJSONUnmarshalCall(pkg.TypesInfo, call) && len(call.Args) >= 2 && exprUsesObject(pkg, call.Args[0], paramObj) {
			target, ok := patterns.ResolveTargetType(pkg, call.Args[1])
			if ok && target != nil {
				bestTarget = target
				bestDepth = 1
				return false
			}
		}

		calleePkg, calleeDecl, _, ok := r.staticCalleeDecl(pkg, call)
		if !ok {
			return true
		}
		for i, arg := range call.Args {
			if !exprUsesObject(pkg, arg, paramObj) {
				continue
			}
			target, childDepth, ok := r.localQueryTargetFromParam(calleePkg, calleeDecl, i, visited, depth-1)
			if ok && target != nil {
				bestTarget = target
				bestDepth = childDepth + 1
				return false
			}
		}
		return true
	})

	if bestTarget == nil {
		return nil, 0, false
	}
	return bestTarget, bestDepth, true
}

func functionParamObject(pkg *packages.Package, fnNode ast.Node, paramIndex int) (types.Object, *ast.BlockStmt, bool) {
	switch fn := fnNode.(type) {
	case *ast.FuncDecl:
		index := 0
		if fn.Type == nil || fn.Type.Params == nil || fn.Body == nil {
			return nil, nil, false
		}
		for _, field := range fn.Type.Params.List {
			names := field.Names
			if len(names) == 0 {
				index++
				continue
			}
			for _, name := range names {
				if index == paramIndex {
					return pkg.TypesInfo.Defs[name], fn.Body, true
				}
				index++
			}
		}
	case *ast.FuncLit:
		index := 0
		if fn.Type == nil || fn.Type.Params == nil || fn.Body == nil {
			return nil, nil, false
		}
		for _, field := range fn.Type.Params.List {
			names := field.Names
			if len(names) == 0 {
				index++
				continue
			}
			for _, name := range names {
				if index == paramIndex {
					return pkg.TypesInfo.Defs[name], fn.Body, true
				}
				index++
			}
		}
	}

	return nil, nil, false
}

func exprUsesObject(pkg *packages.Package, expr ast.Expr, target types.Object) bool {
	if pkg == nil || pkg.TypesInfo == nil || expr == nil || target == nil {
		return false
	}

	used := false
	ast.Inspect(expr, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		if pkg.TypesInfo.Uses[ident] == target {
			used = true
			return false
		}
		return true
	})

	return used
}

func describeLocalQueryCandidate(candidate localQueryCandidate) string {
	if candidate.Target == nil {
		return candidate.Description
	}
	return candidate.Description + " -> " + candidate.Target.PackagePath + "." + candidate.Target.TypeName
}

func (r *Resolver) resolveDecodeTarget(p patterns.Pending) ([]model.Finding, []model.Warning) {
	call, ok := p.Node.(*ast.CallExpr)
	if !ok || len(call.Args) < 2 {
		return nil, []model.Warning{r.warningForPending(p, "ssa_decode_unresolved", "pending decode target does not point to a supported call expression")}
	}

	pkg := r.findPackage(p.PackagePath)
	if pkg == nil {
		return nil, []model.Warning{r.warningForPending(p, "ssa_decode_unresolved", "unable to locate package for pending decode target")}
	}

	sourceKind := model.SourceKind("")
	switch {
	case patterns.IsDatasourceJSONExpr(pkg.TypesInfo, call.Args[0]):
		sourceKind = model.SourceKindDatasourceJSON
	case patterns.IsQueryJSONExpr(pkg.TypesInfo, call.Args[0]):
		sourceKind = model.SourceKindQueryJSON
	default:
		return nil, []model.Warning{r.warningForPending(p, "ssa_decode_unresolved", "unable to classify decode source kind")}
	}

	fn, _ := r.ssaFunctionForNode(pkg, call)
	if fn == nil {
		return nil, []model.Warning{r.warningForPending(p, "ssa_decode_unresolved", "unable to locate enclosing SSA function")}
	}

	value, _ := fn.ValueForExpr(call.Args[1])
	if value == nil {
		return nil, []model.Warning{r.warningForPending(p, "ssa_decode_unresolved", "unable to map decode target expression into SSA")}
	}

	target, ok := r.tracePointeeType(value, pkg.PkgPath, 8)
	if !ok {
		return nil, []model.Warning{r.warningForPending(p, "ssa_decode_unresolved", "unable to resolve decode target type via SSA")}
	}
	if target.Expr == nil {
		position := patterns.PositionOf(pkg, call.Args[1])
		target.Expr = &position
	}

	return []model.Finding{{
		Kind:         model.DecodeKindJSONUnmarshal,
		Source:       sourceKind,
		Position:     patterns.PositionOf(pkg, call),
		FunctionName: p.FunctionName,
		Target:       target,
		Confidence:   model.ConfidenceMedium,
		Notes:        []string{"resolved from SSA after typed AST target resolution failed"},
	}}, nil
}

func (r *Resolver) resolveSecureKey(p patterns.Pending) ([]model.Finding, []model.Warning) {
	index, ok := p.Node.(*ast.IndexExpr)
	if !ok {
		return nil, []model.Warning{r.warningForPending(p, "ssa_secure_key_unresolved", "pending secure key does not point to an index expression")}
	}

	pkg := r.findPackage(p.PackagePath)
	if pkg == nil {
		return nil, []model.Warning{r.warningForPending(p, "ssa_secure_key_unresolved", "unable to locate package for pending secure key")}
	}

	fn, _ := r.ssaFunctionForNode(pkg, index)
	if fn == nil {
		return nil, []model.Warning{r.warningForPending(p, "ssa_secure_key_unresolved", "unable to locate enclosing SSA function")}
	}

	value, _ := fn.ValueForExpr(index.Index)
	if value == nil {
		return nil, []model.Warning{r.warningForPending(p, "ssa_secure_key_unresolved", "unable to map secure key expression into SSA")}
	}

	if !patterns.IsDatasourceSecureExpr(pkg, index.X) {
		base, _ := fn.ValueForExpr(index.X)
		if base == nil {
			return nil, []model.Warning{r.warningForPending(p, "ssa_secure_key_unresolved", "unable to map secure source expression into SSA")}
		}
		if !r.traceSecureMap(base, 8) {
			return nil, nil
		}
	}

	literal, pattern, ok := r.traceString(value, 8)
	if !ok {
		return nil, []model.Warning{r.warningForPending(p, "ssa_secure_key_unresolved", "unable to resolve secure key expression via SSA")}
	}

	finding := model.Finding{
		Source:       model.SourceKindDatasourceSecure,
		Position:     patterns.PositionOf(pkg, index),
		FunctionName: p.FunctionName,
		Destination:  patterns.DestinationForSecureAssignment(index),
		Confidence:   model.ConfidenceMedium,
		Notes:        []string{"resolved from SSA after typed AST key resolution failed"},
	}
	if literal != "" {
		finding.Kind = model.DecodeKindSecureLiteral
		finding.Key = literal
	} else {
		finding.Kind = model.DecodeKindSecureTemplate
		finding.Pattern = pattern
	}

	return []model.Finding{finding}, nil
}

func (r *Resolver) functionByName(pkgPath string, fnName string) *ssa.Function {
	if r == nil || r.Prog == nil {
		return nil
	}

	pkg := r.Prog.ImportedPackage(pkgPath)
	if pkg == nil || pkg.Members == nil {
		return nil
	}

	if member, ok := pkg.Members[fnName]; ok {
		if fn, ok := member.(*ssa.Function); ok {
			return fn
		}
	}

	return nil
}

func (r *Resolver) traceString(value ssa.Value, depth int) (literal string, pattern string, ok bool) {
	if value == nil || depth <= 0 {
		return "", "", false
	}

	switch typed := value.(type) {
	case *ssa.Const:
		if typed.Value == nil || typed.Value.Kind() != constant.String {
			return "", "", false
		}
		return constant.StringVal(typed.Value), "", true
	case *ssa.MakeInterface:
		return r.traceString(typed.X, depth-1)
	case *ssa.ChangeInterface:
		return r.traceString(typed.X, depth-1)
	case *ssa.ChangeType:
		return r.traceString(typed.X, depth-1)
	case *ssa.Convert:
		return r.traceString(typed.X, depth-1)
	case *ssa.UnOp:
		if typed.Op == token.MUL {
			return r.traceLoadedString(typed.X, depth-1)
		}
	case *ssa.Phi:
		return r.tracePhiString(typed, depth-1)
	case *ssa.Call:
		return r.traceCallString(typed.Common(), depth-1)
	}

	return "", "", false
}

func (r *Resolver) tracePointeeType(value ssa.Value, pkgPath string, depth int) (*model.TargetRef, bool) {
	if value == nil || depth <= 0 {
		return nil, false
	}

	switch typed := value.(type) {
	case *ssa.Alloc:
		return targetRefFromType(typed.Type(), pkgPath, true)
	case *ssa.MakeInterface:
		return r.tracePointeeType(typed.X, pkgPath, depth-1)
	case *ssa.ChangeInterface:
		return r.tracePointeeType(typed.X, pkgPath, depth-1)
	case *ssa.ChangeType:
		return r.tracePointeeType(typed.X, pkgPath, depth-1)
	case *ssa.Convert:
		return r.tracePointeeType(typed.X, pkgPath, depth-1)
	case *ssa.UnOp:
		if typed.Op == token.MUL {
			return r.traceLoadedPointeeType(typed.X, pkgPath, depth-1)
		}
	case *ssa.Phi:
		return r.tracePhiPointeeType(typed, pkgPath, depth-1)
	case *ssa.Call:
		return r.traceCallPointeeType(typed.Common(), pkgPath, depth-1)
	}

	return targetRefFromType(value.Type(), pkgPath, false)
}

func (r *Resolver) traceLoadedString(addr ssa.Value, depth int) (string, string, bool) {
	alloc, ok := addr.(*ssa.Alloc)
	if !ok || alloc.Referrers() == nil || depth <= 0 {
		return "", "", false
	}

	var literal string
	var pattern string
	for _, ref := range *alloc.Referrers() {
		store, ok := ref.(*ssa.Store)
		if !ok || store.Addr != addr {
			continue
		}

		nextLiteral, nextPattern, ok := r.traceString(store.Val, depth-1)
		if !ok {
			continue
		}

		if literal == "" && pattern == "" {
			literal = nextLiteral
			pattern = nextPattern
			continue
		}

		if literal != nextLiteral || pattern != nextPattern {
			return "", "", false
		}
	}

	if literal == "" && pattern == "" {
		return "", "", false
	}

	return literal, pattern, true
}

func (r *Resolver) traceLoadedPointeeType(addr ssa.Value, pkgPath string, depth int) (*model.TargetRef, bool) {
	alloc, ok := addr.(*ssa.Alloc)
	if !ok || alloc.Referrers() == nil || depth <= 0 {
		return nil, false
	}

	var found *model.TargetRef
	for _, ref := range *alloc.Referrers() {
		store, ok := ref.(*ssa.Store)
		if !ok || store.Addr != addr {
			continue
		}

		target, ok := r.tracePointeeType(store.Val, pkgPath, depth-1)
		if !ok {
			continue
		}

		if found == nil {
			found = target
			continue
		}

		if !sameTarget(*found, *target) {
			return nil, false
		}
	}

	if found != nil {
		return found, true
	}

	return targetRefFromType(addr.Type(), pkgPath, true)
}

func (r *Resolver) tracePhiString(phi *ssa.Phi, depth int) (string, string, bool) {
	var literal string
	var pattern string
	for _, edge := range phi.Edges {
		nextLiteral, nextPattern, ok := r.traceString(edge, depth)
		if !ok {
			return "", "", false
		}
		if literal == "" && pattern == "" {
			literal = nextLiteral
			pattern = nextPattern
			continue
		}
		if literal != nextLiteral || pattern != nextPattern {
			return "", "", false
		}
	}
	return literal, pattern, literal != "" || pattern != ""
}

func (r *Resolver) tracePhiPointeeType(phi *ssa.Phi, pkgPath string, depth int) (*model.TargetRef, bool) {
	var found *model.TargetRef
	for _, edge := range phi.Edges {
		target, ok := r.tracePointeeType(edge, pkgPath, depth)
		if !ok {
			return nil, false
		}
		if found == nil {
			found = target
			continue
		}
		if !sameTarget(*found, *target) {
			return nil, false
		}
	}
	return found, found != nil
}

func (r *Resolver) traceCallString(call *ssa.CallCommon, depth int) (string, string, bool) {
	if call == nil || depth <= 0 {
		return "", "", false
	}

	callee := call.StaticCallee()
	if callee == nil {
		return "", "", false
	}

	if callee.Pkg != nil && callee.Pkg.Pkg != nil && callee.Pkg.Pkg.Path() == "fmt" && callee.Name() == "Sprintf" && len(call.Args) > 0 {
		format, _, ok := r.traceString(call.Args[0], depth-1)
		if !ok || format == "" {
			return "", "", false
		}
		return "", patterns.NormalizeTemplate(format), true
	}

	return r.traceReturnedString(callee, depth-1)
}

func (r *Resolver) traceCallPointeeType(call *ssa.CallCommon, pkgPath string, depth int) (*model.TargetRef, bool) {
	if call == nil || depth <= 0 {
		return nil, false
	}

	callee := call.StaticCallee()
	if callee == nil {
		return nil, false
	}

	return r.traceReturnedTarget(callee, pkgPath, depth-1)
}

func (r *Resolver) traceSecureMap(value ssa.Value, depth int) bool {
	if value == nil || depth <= 0 {
		return false
	}

	switch typed := value.(type) {
	case *ssa.Field:
		return isSecureFieldValue(typed)
	case *ssa.FieldAddr:
		return isSecureFieldAddrValue(typed)
	case *ssa.MakeInterface:
		return r.traceSecureMap(typed.X, depth-1)
	case *ssa.ChangeInterface:
		return r.traceSecureMap(typed.X, depth-1)
	case *ssa.ChangeType:
		return r.traceSecureMap(typed.X, depth-1)
	case *ssa.Convert:
		return r.traceSecureMap(typed.X, depth-1)
	case *ssa.UnOp:
		if typed.Op == token.MUL {
			return r.traceLoadedSecureMap(typed.X, depth-1)
		}
	case *ssa.Phi:
		for _, edge := range typed.Edges {
			if r.traceSecureMap(edge, depth-1) {
				return true
			}
		}
	case *ssa.Call:
		callee := typed.Common().StaticCallee()
		if callee != nil {
			for _, block := range callee.Blocks {
				for _, instr := range block.Instrs {
					ret, ok := instr.(*ssa.Return)
					if !ok || len(ret.Results) == 0 {
						continue
					}
					if r.traceSecureMap(ret.Results[0], depth-1) {
						return true
					}
				}
			}
		}
	}

	return false
}

func (r *Resolver) traceLoadedSecureMap(addr ssa.Value, depth int) bool {
	alloc, ok := addr.(*ssa.Alloc)
	if !ok || alloc.Referrers() == nil || depth <= 0 {
		return false
	}

	for _, ref := range *alloc.Referrers() {
		store, ok := ref.(*ssa.Store)
		if !ok || store.Addr != addr {
			continue
		}
		if r.traceSecureMap(store.Val, depth-1) {
			return true
		}
	}

	return false
}

func (r *Resolver) traceReturnedString(fn *ssa.Function, depth int) (string, string, bool) {
	if fn == nil || depth <= 0 {
		return "", "", false
	}

	var literal string
	var pattern string
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			ret, ok := instr.(*ssa.Return)
			if !ok || len(ret.Results) == 0 {
				continue
			}

			nextLiteral, nextPattern, ok := r.traceString(ret.Results[0], depth-1)
			if !ok {
				return "", "", false
			}

			if literal == "" && pattern == "" {
				literal = nextLiteral
				pattern = nextPattern
				continue
			}

			if literal != nextLiteral || pattern != nextPattern {
				return "", "", false
			}
		}
	}

	return literal, pattern, literal != "" || pattern != ""
}

func (r *Resolver) traceReturnedTarget(fn *ssa.Function, pkgPath string, depth int) (*model.TargetRef, bool) {
	if fn == nil || depth <= 0 {
		return nil, false
	}

	var found *model.TargetRef
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			ret, ok := instr.(*ssa.Return)
			if !ok || len(ret.Results) == 0 {
				continue
			}

			target, ok := r.tracePointeeType(ret.Results[0], pkgPath, depth-1)
			if !ok {
				return nil, false
			}

			if found == nil {
				found = target
				continue
			}

			if !sameTarget(*found, *target) {
				return nil, false
			}
		}
	}

	return found, found != nil
}

func (r *Resolver) ssaFunctionForNode(pkg *packages.Package, node ast.Node) (*ssa.Function, *ast.File) {
	if pkg == nil || node == nil || r == nil || r.Prog == nil {
		return nil, nil
	}

	ssaPkg := r.Prog.Package(pkg.Types)
	if ssaPkg == nil {
		return nil, nil
	}

	for _, file := range pkg.Syntax {
		if node.Pos() < file.Pos() || node.End() > file.End() {
			continue
		}

		path, _ := astutil.PathEnclosingInterval(file, node.Pos(), node.End())
		if len(path) == 0 {
			continue
		}

		return ssa.EnclosingFunction(ssaPkg, path), file
	}

	return nil, nil
}

func (r *Resolver) findPackage(pkgPath string) *packages.Package {
	if r == nil || r.Load == nil {
		return nil
	}

	for _, pkg := range r.Load.Packages {
		if pkg.PkgPath == pkgPath {
			return pkg
		}
	}

	return nil
}

func (r *Resolver) warningForPending(p patterns.Pending, code string, message string) model.Warning {
	position := model.Position{}
	if pkg := r.findPackage(p.PackagePath); pkg != nil && p.Node != nil {
		position = patterns.PositionOf(pkg, p.Node)
	}

	return model.Warning{
		Position: position,
		Code:     code,
		Message:  message,
	}
}

func targetRefFromType(typ types.Type, pkgPath string, assumePointer bool) (*model.TargetRef, bool) {
	if typ == nil {
		return nil, false
	}

	pointer := assumePointer
	typ = types.Unalias(typ)
	if ptr, ok := typ.(*types.Pointer); ok {
		pointer = true
		typ = types.Unalias(ptr.Elem())
	}
	typ = types.Unalias(typ)

	typeString := types.TypeString(typ, nil)
	named, ok := typ.(*types.Named)
	if !ok {
		if _, ok := typ.Underlying().(*types.Struct); !ok {
			return nil, false
		}
		return &model.TargetRef{
			PackagePath: pkgPath,
			Pointer:     pointer,
			TypeString:  typeString,
		}, true
	}

	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return nil, false
	}

	return &model.TargetRef{
		PackagePath: obj.Pkg().Path(),
		TypeName:    obj.Name(),
		Pointer:     pointer,
		TypeString:  typeString,
	}, true
}

func sameTarget(left model.TargetRef, right model.TargetRef) bool {
	return left.PackagePath == right.PackagePath &&
		left.TypeName == right.TypeName &&
		left.Pointer == right.Pointer &&
		left.TypeString == right.TypeString
}

func isSecureFieldValue(field *ssa.Field) bool {
	if field == nil {
		return false
	}

	named := rootNamedType(field.X.Type())
	if named == nil {
		return false
	}

	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != "github.com/grafana/grafana-plugin-sdk-go/backend" || obj.Name() != "DataSourceInstanceSettings" {
		return false
	}

	st, ok := named.Underlying().(*types.Struct)
	if !ok || field.Field >= st.NumFields() {
		return false
	}

	return st.Field(field.Field).Name() == "DecryptedSecureJSONData"
}

func isSecureFieldAddrValue(field *ssa.FieldAddr) bool {
	if field == nil {
		return false
	}

	named := rootNamedType(field.X.Type())
	if named == nil {
		return false
	}

	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != "github.com/grafana/grafana-plugin-sdk-go/backend" || obj.Name() != "DataSourceInstanceSettings" {
		return false
	}

	st, ok := named.Underlying().(*types.Struct)
	if !ok || field.Field >= st.NumFields() {
		return false
	}

	return st.Field(field.Field).Name() == "DecryptedSecureJSONData"
}

func rootNamedType(typ types.Type) *types.Named {
	typ = types.Unalias(typ)
	for {
		ptr, ok := typ.(*types.Pointer)
		if !ok {
			break
		}
		typ = types.Unalias(ptr.Elem())
	}

	named, _ := typ.(*types.Named)
	return named
}

func (r *Resolver) frameworkSeedsInUse() []frameworkSeed {
	out := make([]frameworkSeed, 0)
	seen := map[string]struct{}{}
	for _, pkg := range r.Load.Packages {
		if !r.isLocalPackage(pkg) || pkg == nil || pkg.Types == nil {
			continue
		}

		ssaPkg := r.Prog.Package(pkg.Types)
		if ssaPkg == nil {
			continue
		}

		for _, member := range ssaPkg.Members {
			fn, ok := member.(*ssa.Function)
			if !ok {
				continue
			}
			out = append(out, r.collectFrameworkCallsFromFunction(pkg, fn, seen)...)
		}
	}
	return out
}

func (r *Resolver) collectFrameworkCallsFromFunction(pkg *packages.Package, fn *ssa.Function, seen map[string]struct{}) []frameworkSeed {
	if pkg == nil || fn == nil {
		return nil
	}

	out := make([]frameworkSeed, 0)
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(ssa.CallInstruction)
			if !ok {
				continue
			}

			common := call.Common()
			if common == nil {
				continue
			}

			callee := common.StaticCallee()
			if callee == nil || callee.Pkg == nil || callee.Pkg.Pkg == nil {
				continue
			}
			calleePkg := callee.Pkg.Pkg.Path()
			if calleePkg == "" || calleePkg == pkg.PkgPath || r.isLocalPackagePath(calleePkg) {
				continue
			}
			if !looksLikeFrameworkSeed(callee) {
				continue
			}
			key := functionDescription(callee)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			position := pkg.Fset.PositionFor(call.Pos(), false)
			out = append(out, frameworkSeed{
				Function: callee,
				Position: model.Position{
					File:   position.Filename,
					Line:   position.Line,
					Column: position.Column,
				},
			})
		}
	}
	return out
}

func looksLikeFrameworkSeed(fn *ssa.Function) bool {
	if fn == nil {
		return false
	}
	name := strings.ToLower(fn.Name())
	switch name {
	case "newdatasource", "newservice", "querydata", "callresource", "checkhealth":
		return true
	}
	if fn.Signature != nil && fn.Signature.Recv() != nil {
		if named := rootNamedType(fn.Signature.Recv().Type()); named != nil {
			if obj := named.Obj(); obj != nil {
				recvName := strings.ToLower(obj.Name())
				if strings.Contains(recvName, "datasource") || strings.Contains(recvName, "service") {
					return true
				}
			}
		}
	}
	if fn.Signature != nil {
		results := fn.Signature.Results()
		for i := 0; i < results.Len(); i++ {
			if named := rootNamedType(results.At(i).Type()); named != nil {
				if obj := named.Obj(); obj != nil {
					resultName := strings.ToLower(obj.Name())
					if strings.Contains(resultName, "datasource") || strings.Contains(resultName, "service") {
						return true
					}
				}
			}
		}
	}
	return false
}

func (r *Resolver) bestFrameworkQueryCandidate(seed frameworkSeed) (frameworkQueryCandidate, bool) {
	functions := r.reachableFrameworkFunctions([]*ssa.Function{seed.Function}, 16)
	if len(functions) == 0 {
		functions = r.functionsInPackage(seed.Function.Pkg.Pkg.Path())
	} else {
		functions = append(functions, r.functionsInPackage(seed.Function.Pkg.Pkg.Path())...)
	}
	if len(functions) == 0 {
		return frameworkQueryCandidate{}, false
	}

	best := frameworkQueryCandidate{}
	for _, fn := range functions {
		target, fromDecode, ok := r.frameworkQueryTarget(fn)
		if !ok || target == nil || !looksLikeQueryTarget(*target) {
			continue
		}

		candidate := frameworkQueryCandidate{
			PkgPath:     seed.Function.Pkg.Pkg.Path(),
			Function:    fn,
			Target:      target,
			Position:    seed.Position,
			Score:       frameworkQueryCandidateScore(fn, target, fromDecode),
			FromDecode:  fromDecode,
			Description: functionDescription(fn),
		}
		if best.Target == nil || candidate.Score > best.Score {
			best = candidate
		}
	}

	return best, best.Target != nil
}

func (r *Resolver) functionsInPackage(pkgPath string) []*ssa.Function {
	all := ssautil.AllFunctions(r.Prog)
	out := make([]*ssa.Function, 0)
	for fn := range all {
		if fn == nil || fn.Pkg == nil || fn.Pkg.Pkg == nil || fn.Pkg.Pkg.Path() != pkgPath {
			continue
		}
		out = append(out, fn)
	}
	return out
}

func (r *Resolver) reachableFrameworkFunctions(roots []*ssa.Function, depth int) []*ssa.Function {
	seen := map[*ssa.Function]struct{}{}
	out := make([]*ssa.Function, 0)

	var visit func(fn *ssa.Function, remaining int)
	visit = func(fn *ssa.Function, remaining int) {
		if fn == nil || remaining <= 0 {
			return
		}
		if _, ok := seen[fn]; ok {
			return
		}
		seen[fn] = struct{}{}
		out = append(out, fn)

		for _, method := range r.methodsForReturnTypes(fn) {
			visit(method, remaining-1)
		}
		for _, anon := range fn.AnonFuncs {
			visit(anon, remaining-1)
		}

		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if closure, ok := instr.(*ssa.MakeClosure); ok {
					if anon, ok := closure.Fn.(*ssa.Function); ok {
						visit(anon, remaining-1)
					}
				}
				call, ok := instr.(ssa.CallInstruction)
				if !ok {
					continue
				}
				common := call.Common()
				if common == nil {
					continue
				}
				if callee := common.StaticCallee(); callee != nil {
					visit(callee, remaining-1)
				}
			}
		}
	}

	for _, fn := range roots {
		visit(fn, depth)
	}

	return out
}

func (r *Resolver) methodsForReturnTypes(fn *ssa.Function) []*ssa.Function {
	if fn == nil || fn.Signature == nil || fn.Signature.Results() == nil {
		return nil
	}

	methods := make([]*ssa.Function, 0)
	for i := 0; i < fn.Signature.Results().Len(); i++ {
		named := rootNamedType(fn.Signature.Results().At(i).Type())
		if named == nil {
			continue
		}
		methods = append(methods, r.methodsForNamedType(named)...)
	}

	return methods
}

func (r *Resolver) methodsForNamedType(named *types.Named) []*ssa.Function {
	key := namedKey(named)
	if key == "" {
		return nil
	}
	if methods, ok := r.methodsByNamed[key]; ok {
		return methods
	}

	methods := make([]*ssa.Function, 0)
	for candidate := range r.allFunctions {
		if candidate == nil || candidate.Signature == nil || candidate.Signature.Recv() == nil {
			continue
		}
		if recv := rootNamedType(candidate.Signature.Recv().Type()); recv != nil && sameNamedType(recv, named) {
			methods = append(methods, candidate)
		}
	}
	r.methodsByNamed[key] = methods
	return methods
}

func (r *Resolver) frameworkQueryTarget(fn *ssa.Function) (*model.TargetRef, bool, bool) {
	if fn == nil || fn.Signature == nil {
		return nil, false, false
	}
	if !hasBackendDataQueryParam(fn.Signature) {
		return nil, false, false
	}

	if target, ok := r.frameworkQueryDecodeTarget(fn); ok {
		return target, true, true
	}

	target, ok := r.traceReturnedTarget(fn, fn.Pkg.Pkg.Path(), 8)
	return target, false, ok
}

func (r *Resolver) dataSourceSettingsUsageInFunction(fn *ssa.Function) DataSourceSettingsUsage {
	if fn == nil || fn.Pkg == nil || fn.Pkg.Pkg == nil {
		return DataSourceSettingsUsage{}
	}

	pkg := r.findPackage(fn.Pkg.Pkg.Path())
	if pkg == nil || pkg.TypesInfo == nil {
		return DataSourceSettingsUsage{}
	}

	syntax := fn.Syntax()
	if syntax == nil {
		return DataSourceSettingsUsage{}
	}

	usage := DataSourceSettingsUsage{}
	ast.Inspect(syntax, func(node ast.Node) bool {
		sel, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if selection := pkg.TypesInfo.Selections[sel]; selection != nil {
			if isDataSourceInstanceSettingsType(selection.Recv()) {
				switch selection.Obj().Name() {
				case "URL":
					usage.UsesURL = true
				case "HTTPClientOptions", "ProxyOptions", "ProxyOptionsFromContext", "ProxyClient":
					usage.UsesHTTPOptions = true
				}
			}
			return !(usage.UsesURL && usage.UsesHTTPOptions)
		}
		if obj, ok := pkg.TypesInfo.Uses[sel.Sel].(*types.Func); ok && obj.Name() == "HTTPClientOptions" {
			if sig, ok := obj.Type().(*types.Signature); ok && sig.Recv() != nil && isDataSourceInstanceSettingsType(sig.Recv().Type()) {
				usage.UsesHTTPOptions = true
			}
		}
		return !(usage.UsesURL && usage.UsesHTTPOptions)
	})

	return usage
}

func (r *Resolver) frameworkQueryDecodeTarget(fn *ssa.Function) (*model.TargetRef, bool) {
	if fn == nil || fn.Pkg == nil || fn.Pkg.Pkg == nil {
		return nil, false
	}

	pkg := r.findPackage(fn.Pkg.Pkg.Path())
	if pkg == nil {
		return nil, false
	}

	syntax := fn.Syntax()
	if syntax == nil {
		return nil, false
	}

	targets := make([]*model.TargetRef, 0)
	ast.Inspect(syntax, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.FuncDecl:
			return typed == syntax
		case *ast.FuncLit:
			return typed == syntax
		case *ast.CallExpr:
			findings, _ := patterns.FindJSONUnmarshalTargets(pkg, typed)
			for _, finding := range findings {
				if finding.Source != model.SourceKindQueryJSON || finding.Target == nil {
					continue
				}
				if containsTarget(targets, finding.Target) {
					continue
				}
				targets = append(targets, finding.Target)
			}
		}
		return true
	})

	if len(targets) != 1 {
		return nil, false
	}

	return targets[0], true
}

func hasBackendDataQueryParam(sig *types.Signature) bool {
	if sig == nil || sig.Params() == nil {
		return false
	}
	for i := 0; i < sig.Params().Len(); i++ {
		if isBackendDataQueryType(sig.Params().At(i).Type()) {
			return true
		}
	}
	return false
}

func isBackendDataQueryType(typ types.Type) bool {
	named := rootNamedType(typ)
	if named == nil {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == "github.com/grafana/grafana-plugin-sdk-go/backend" && obj.Name() == "DataQuery"
}

func frameworkQueryCandidateScore(fn *ssa.Function, target *model.TargetRef, fromDecode bool) int {
	score := 0
	if fromDecode {
		score += 2000
	}
	name := strings.ToLower(fn.Name())
	switch {
	case name == "getquery":
		score += 1000
	case strings.Contains(name, "query"):
		score += 300
	}

	if target != nil {
		typeName := strings.ToLower(target.TypeName)
		switch {
		case typeName == "query":
			score += 400
		case strings.Contains(typeName, "query"):
			score += 200
		}
		if strings.Contains(strings.ToLower(target.PackagePath), "sqlutil") {
			score += 200
		}
	}
	return score
}

func containsTarget(targets []*model.TargetRef, target *model.TargetRef) bool {
	if target == nil {
		return false
	}
	for _, candidate := range targets {
		if candidate == nil {
			continue
		}
		if sameTarget(*candidate, *target) {
			return true
		}
	}
	return false
}

func looksLikeQueryTarget(target model.TargetRef) bool {
	if strings.Contains(target.PackagePath, "/genproto/") {
		return false
	}
	typeName := strings.ToLower(target.TypeName)
	if target.PackagePath == "github.com/grafana/grafana-plugin-sdk-go/backend" && typeName == "dataquery" {
		return false
	}
	typeString := strings.ToLower(target.TypeString)
	if typeName != "" && strings.Contains(typeName, "query") {
		return true
	}
	return strings.Contains(typeString, "query")
}

func functionDescription(fn *ssa.Function) string {
	if fn == nil {
		return ""
	}
	if fn.Pkg == nil || fn.Pkg.Pkg == nil {
		return fn.String()
	}
	return fn.Pkg.Pkg.Path() + "." + fn.Name()
}

func sameNamedType(left *types.Named, right *types.Named) bool {
	if left == nil || right == nil {
		return false
	}
	leftObj := left.Obj()
	rightObj := right.Obj()
	if leftObj == nil || rightObj == nil {
		return false
	}
	if leftObj.Name() != rightObj.Name() {
		return false
	}
	if leftObj.Pkg() == nil || rightObj.Pkg() == nil {
		return leftObj.Pkg() == rightObj.Pkg()
	}
	return leftObj.Pkg().Path() == rightObj.Pkg().Path()
}

func namedKey(named *types.Named) string {
	if named == nil || named.Obj() == nil {
		return ""
	}
	obj := named.Obj()
	if obj.Pkg() == nil {
		return obj.Name()
	}
	return obj.Pkg().Path() + "." + obj.Name()
}

func describeFrameworkCandidate(candidate frameworkQueryCandidate) string {
	target := ""
	if candidate.Target != nil {
		target = candidate.Target.PackagePath + "." + candidate.Target.TypeName
	}
	if target == "" {
		return candidate.Description
	}
	return candidate.Description + " -> " + target
}

func (r *Resolver) isLocalPackage(pkg *packages.Package) bool {
	if pkg == nil {
		return false
	}
	if r.inLocalModule(pkg.Module) {
		return true
	}
	if pkg.Module == nil && packageHasFileInDir(pkg, r.Load.Dir) {
		return true
	}
	return r.isLocalPackagePath(pkg.PkgPath)
}

func (r *Resolver) isLocalPackagePath(pkgPath string) bool {
	for _, pkg := range r.Load.Packages {
		if pkg == nil || pkg.PkgPath != pkgPath {
			continue
		}
		if r.inLocalModule(pkg.Module) {
			return true
		}
		if pkg.Module == nil && packageHasFileInDir(pkg, r.Load.Dir) {
			return true
		}
	}
	return false
}

func (r *Resolver) inLocalModule(module *packages.Module) bool {
	if module == nil || module.Dir == "" {
		return false
	}
	absDir, ok := absPath(module.Dir)
	if !ok {
		return false
	}
	_, ok = r.localModuleDir[absDir]
	return ok
}

func (u *DataSourceSettingsUsage) merge(other DataSourceSettingsUsage) {
	if u == nil {
		return
	}
	u.UsesURL = u.UsesURL || other.UsesURL
	u.UsesHTTPOptions = u.UsesHTTPOptions || other.UsesHTTPOptions
}

func isDataSourceInstanceSettingsType(typ types.Type) bool {
	named := rootNamedType(typ)
	if named == nil || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path() == "github.com/grafana/grafana-plugin-sdk-go/backend" && named.Obj().Name() == "DataSourceInstanceSettings"
}

func packageHasFileInDir(pkg *packages.Package, dir string) bool {
	if pkg == nil {
		return false
	}
	for _, file := range pkg.GoFiles {
		if fileInDir(file, dir) {
			return true
		}
	}
	for _, file := range pkg.CompiledGoFiles {
		if fileInDir(file, dir) {
			return true
		}
	}
	return false
}

func absPath(name string) (string, bool) {
	if name == "" {
		return "", false
	}
	absName, err := filepath.Abs(name)
	if err != nil {
		return "", false
	}
	return absName, true
}

func fileInDir(name string, dir string) bool {
	if name == "" || dir == "" {
		return false
	}
	absName, ok := absPath(name)
	if !ok {
		return false
	}
	absDir, ok := absPath(dir)
	if !ok {
		return false
	}
	rel, err := filepath.Rel(absDir, absName)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && rel != "..")
}

func sameDir(left string, right string) bool {
	if left == "" || right == "" {
		return false
	}
	absLeft, ok := absPath(left)
	if !ok {
		return false
	}
	absRight, ok := absPath(right)
	if !ok {
		return false
	}
	return absLeft == absRight
}
