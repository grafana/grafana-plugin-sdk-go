package load

import (
	"fmt"
	"go/token"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

type Config struct {
	Dir        string
	Patterns   []string
	BuildFlags []string
	Tests      bool
	Overlay    map[string][]byte
	NeedModule bool
}

type Result struct {
	Fset         *token.FileSet
	Dir          string
	RootPackages []*packages.Package
	Packages     []*packages.Package
}

func Packages(cfg Config) (*Result, error) {
	dir := cfg.Dir
	if dir == "" {
		dir = "."
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	pkgCfg := &packages.Config{
		Dir:        dir,
		Mode:       defaultMode(cfg.NeedModule),
		Tests:      cfg.Tests,
		BuildFlags: cfg.BuildFlags,
		Overlay:    cfg.Overlay,
	}

	pkgs, err := packages.Load(pkgCfg, cfg.Patterns...)
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("go/packages returned errors")
	}

	var fset *token.FileSet
	for _, pkg := range pkgs {
		if pkg.Fset != nil {
			fset = pkg.Fset
			break
		}
	}

	return &Result{
		Fset:         fset,
		Dir:          absDir,
		RootPackages: pkgs,
		Packages:     collectPackages(pkgs),
	}, nil
}

func defaultMode(needModule bool) packages.LoadMode {
	mode := packages.NeedName |
		packages.NeedFiles |
		packages.NeedCompiledGoFiles |
		packages.NeedImports |
		packages.NeedSyntax |
		packages.NeedTypes |
		packages.NeedTypesInfo |
		packages.NeedDeps

	if needModule {
		mode |= packages.NeedModule
	}

	return mode
}

func collectPackages(root []*packages.Package) []*packages.Package {
	out := make([]*packages.Package, 0)
	seen := map[string]struct{}{}

	var visit func(pkg *packages.Package)
	visit = func(pkg *packages.Package) {
		if pkg == nil {
			return
		}

		key := pkg.ID
		if key == "" {
			key = pkg.PkgPath
		}
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, pkg)

		for _, imported := range pkg.Imports {
			visit(imported)
		}
	}

	for _, pkg := range root {
		visit(pkg)
	}

	return out
}
