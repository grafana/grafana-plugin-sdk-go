package patterns

import (
	"go/ast"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/load"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
)

func RunTyped(ctx *Context) (*Result, error) {
	if ctx == nil || ctx.Load == nil {
		return &Result{}, nil
	}

	if ctx.Report == nil {
		ctx.Report = &model.Report{}
	}

	if ctx.Inspector == nil {
		var files []*ast.File
		for _, pkg := range ctx.Load.Packages {
			files = append(files, pkg.Syntax...)
		}
		ctx.Inspector = inspector.New(files)
	}

	var pending []Pending

	for _, pkg := range ctx.Load.Packages {
		if pkg.TypesInfo == nil {
			continue
		}
		if !packageInDir(ctx.Load.Dir, pkg) {
			continue
		}

		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
				switch n := node.(type) {
				case *ast.CallExpr:
					findings, unresolved := FindJSONUnmarshalTargets(pkg, n)
					ctx.Report.Findings = append(ctx.Report.Findings, findings...)
					pending = append(pending, unresolved...)

					findings, unresolved = FindMapstructureDecodeTargets(pkg, n)
					ctx.Report.Findings = append(ctx.Report.Findings, findings...)
					pending = append(pending, unresolved...)
				case *ast.IndexExpr:
					findings, unresolved := FindSecureLiteralKeys(pkg, n)
					ctx.Report.Findings = append(ctx.Report.Findings, findings...)
					pending = append(pending, unresolved...)
				case *ast.RangeStmt:
					findings, unresolved := FindSecureBulkRanges(pkg, n)
					ctx.Report.Findings = append(ctx.Report.Findings, findings...)
					pending = append(pending, unresolved...)
				}

				return true
			})
		}
	}

	return &Result{
		Report:  *ctx.Report,
		Pending: pending,
	}, nil
}

func NewContext(loadRes *load.Result) *Context {
	return &Context{
		Load:   loadRes,
		Report: &model.Report{},
	}
}

func packageInDir(root string, pkg *packages.Package) bool {
	if pkg == nil {
		return false
	}
	for _, file := range pkg.CompiledGoFiles {
		if fileInDir(root, file) {
			return true
		}
	}
	for _, file := range pkg.GoFiles {
		if fileInDir(root, file) {
			return true
		}
	}
	return false
}

func fileInDir(root string, file string) bool {
	if root == "" || file == "" {
		return false
	}
	absFile, err := filepath.Abs(file)
	if err != nil {
		return false
	}
	if absFile == root {
		return true
	}
	prefix := root
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(absFile, prefix)
}
