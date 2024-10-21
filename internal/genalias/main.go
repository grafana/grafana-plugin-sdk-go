package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	flag.Parse()

	// We accept either one directory or a list of files. Which do we have?
	args := flag.Args()
	if len(args) == 0 {
		// Default: process whole package in current directory.
		args = []string{"."}
	}

	// Parse the package once.
	var dir string
	if len(args) == 1 && isDirectory(args[0]) {
		dir = args[0]
	} else {
		log.Fatal("missing directory arg")
	}

	g := parse(dir)

	var w io.WriteCloser = os.Stdout

	goFile := os.Getenv("GOFILE")
	if goFile != "" {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		fileWithoutExt := strings.TrimSuffix(goFile, filepath.Ext(goFile))

		w, err = os.Create(filepath.Join(wd, fmt.Sprintf("%s_gen.go", fileWithoutExt)))
		if err != nil {
			log.Fatal(err)
		}
	}

	g.generate(w)
}

//gocyclo:ignore
func parse(dir string) *Generator {
	pkgs, err := packages.Load(&packages.Config{Tests: false, Dir: dir, Mode: packages.LoadAllSyntax})
	if err != nil {
		log.Fatal(err)
	}

	g := newGenerator("backend")

	for _, pkg := range pkgs {
		g.TargetPackageName = pkg.Name
		g.addImport(pkg.ID)

		for _, f := range pkg.Syntax {
			ast.Inspect(f, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.GenDecl:
					for _, nodeSpec := range node.Specs {
						switch spec := nodeSpec.(type) {
						case *ast.TypeSpec:
							if spec.Name.IsExported() {
								s := newParsedStruct(spec.Name.Name)
								if node.Doc != nil {
									for _, com := range node.Doc.List {
										s.addComment(com.Text)
									}
								}

								g.addStruct(s)
							}
						case *ast.ValueSpec:
							if node.Tok == token.CONST {
								for _, name := range spec.Names {
									c := pkg.TypesInfo.ObjectOf(name).(*types.Const)
									if c.Exported() {
										c := newParsedConst(c.Name())
										if spec.Doc != nil {
											for _, com := range spec.Doc.List {
												c.addComment(com.Text)
											}
										}

										g.addConst(c)
									}
								}
							}
						}
					}
				case *ast.FuncDecl:
					if node.Recv == nil && node.Name.IsExported() {
						pf := newParsedFunc(node.Name.Name)

						if node.Doc != nil {
							for _, com := range node.Doc.List {
								pf.addComment(com.Text)
							}
						}

						if node.Type.Params != nil && node.Type.Params.List != nil {
							for _, p := range node.Type.Params.List {
								pType := exprToString(p.Type)

								if se, ok := p.Type.(*ast.StarExpr); ok {
									if ti, exists := pkg.TypesInfo.Types[se.X]; exists {
										if named, ok := ti.Type.(*types.Named); ok {
											if named.Obj() != nil && named.Obj().Pkg() != nil {
												g.addImport(named.Obj().Pkg().Path())
											}
										}
									}
								} else {
									if ti, exists := pkg.TypesInfo.Types[p.Type]; exists {
										if named, ok := ti.Type.(*types.Named); ok {
											if named.Obj() != nil && named.Obj().Pkg() != nil {
												g.addImport(named.Obj().Pkg().Path())
											}
										}
									}
								}

								for _, name := range p.Names {
									pf.addParam(name.Name, pType)
								}
							}
						}

						if node.Type.Results != nil && node.Type.Results.List != nil {
							for _, field := range node.Type.Results.List {
								if se, ok := field.Type.(*ast.StarExpr); ok {
									if ti, exists := pkg.TypesInfo.Types[se.X]; exists {
										if named, ok := ti.Type.(*types.Named); ok {
											if named.Obj() != nil && named.Obj().Pkg() != nil {
												g.addImport(named.Obj().Pkg().Path())
											}
										}
									}
								} else {
									if ti, exists := pkg.TypesInfo.Types[field.Type]; exists {
										if named, ok := ti.Type.(*types.Named); ok {
											if named.Obj() != nil && named.Obj().Pkg() != nil {
												g.addImport(named.Obj().Pkg().Path())
											}
										}
									}
								}

								pf.addReturnType("", exprToString(field.Type))
							}
						}

						g.addFunction(pf)
					}
				}

				return true
			})
		}
	}

	return g
}

// isDirectory reports whether the named file is a directory.
func isDirectory(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		log.Fatal(err)
	}
	return info.IsDir()
}

type Generator struct {
	PackageName       string
	TargetPackageName string
	Imports           map[string]string
	Structs           map[string]*ParsedStruct
	Consts            map[string]*ParsedConst
	Functions         map[string]*ParsedFunc
}

func newGenerator(name string) *Generator {
	return &Generator{
		PackageName: name,
		Imports:     map[string]string{},
		Structs:     map[string]*ParsedStruct{},
		Consts:      map[string]*ParsedConst{},
		Functions:   map[string]*ParsedFunc{},
	}
}

func (g *Generator) addImport(s string) {
	g.Imports[s] = s
}

func (g *Generator) addStruct(s *ParsedStruct) {
	g.Structs[s.Name] = s
}

func (g *Generator) addConst(c *ParsedConst) {
	g.Consts[c.Name] = c
}

func (g *Generator) addFunction(pf *ParsedFunc) {
	g.Functions[pf.Name] = pf
}

func (g *Generator) generate(w io.Writer) {
	fmt.Fprintln(w, "// Code generated by genalias. DO NOT EDIT.")
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "package %s\n", g.PackageName)
	fmt.Fprintln(w, "")

	fmt.Fprintln(w, "import (")

	sortedImports := make([]string, 0, len(g.Imports))
	for k := range g.Imports {
		sortedImports = append(sortedImports, k)
	}

	sort.Strings(sortedImports)

	for _, k := range sortedImports {
		i := g.Imports[k]
		fmt.Fprintf(w, "\t"+`"%s"`+"\n", i)
	}

	fmt.Fprintln(w, ")")
	fmt.Fprintln(w, "")

	if len(g.Consts) > 0 {
		fmt.Fprintln(w, "const (")

		sortedConsts := make([]string, 0, len(g.Consts))
		for k := range g.Consts {
			sortedConsts = append(sortedConsts, k)
		}

		sort.Strings(sortedConsts)

		for i, k := range sortedConsts {
			c := g.Consts[k]
			c.generate(w, g.TargetPackageName)
			if i < (len(sortedConsts) - 1) {
				fmt.Fprintln(w, "")
			}
		}

		fmt.Fprint(w, ")")
		fmt.Fprintln(w, "")
	}

	if len(g.Structs) > 0 {
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "type (")

		sortedStructs := make([]string, 0, len(g.Structs))
		for k := range g.Structs {
			sortedStructs = append(sortedStructs, k)
		}

		sort.Strings(sortedStructs)

		for i, k := range sortedStructs {
			c := g.Structs[k]
			c.generate(w, g.TargetPackageName)
			if i < (len(sortedStructs) - 1) {
				fmt.Fprintln(w, "")
			}
		}

		fmt.Fprint(w, ")")
		fmt.Fprintln(w, "")
	}

	if len(g.Functions) > 0 {
		fmt.Fprintln(w, "")

		sortedFuncts := make([]string, 0, len(g.Functions))
		for k := range g.Functions {
			sortedFuncts = append(sortedFuncts, k)
		}

		sort.Strings(sortedFuncts)

		for i, k := range sortedFuncts {
			c := g.Functions[k]
			c.generate(w, g.TargetPackageName)
			if i < (len(sortedFuncts) - 1) {
				fmt.Fprintln(w, "")
			}
		}

		fmt.Fprintln(w, "")
	}
}

type ParsedStruct struct {
	Name     string
	Comments []string
}

func newParsedStruct(name string) *ParsedStruct {
	return &ParsedStruct{
		Name:     name,
		Comments: []string{},
	}
}

func (s *ParsedStruct) addComment(comment string) {
	s.Comments = append(s.Comments, comment)
}

func (s ParsedStruct) generate(w io.Writer, aliasPkg string) {
	for _, comment := range s.Comments {
		fmt.Fprintf(w, "\t%s\n", comment)
	}

	fmt.Fprintf(w, "\t%s = %s.%s\n", s.Name, aliasPkg, s.Name)
}

type ParsedConst struct {
	Name     string
	Comments []string
}

func newParsedConst(name string) *ParsedConst {
	return &ParsedConst{
		Name:     name,
		Comments: []string{},
	}
}

func (c *ParsedConst) addComment(s string) {
	c.Comments = append(c.Comments, s)
}

func (c *ParsedConst) generate(w io.Writer, aliasPkg string) {
	for _, comment := range c.Comments {
		fmt.Fprintf(w, "\t%s\n", comment)
	}

	fmt.Fprintf(w, "\t%s = %s.%s\n", c.Name, aliasPkg, c.Name)
}

type ParsedFuncParam struct {
	Name string
	Type string
}

func (p ParsedFuncParam) nameAndType() string {
	if p.Name == "" {
		return p.Type
	}

	return fmt.Sprintf("%s %s", p.Name, p.Type)
}

type ParsedFunc struct {
	Name        string
	Params      []*ParsedFuncParam
	ReturnTypes []*ParsedFuncParam
	Comments    []string
}

func newParsedFunc(name string) *ParsedFunc {
	return &ParsedFunc{
		Name:        name,
		Params:      []*ParsedFuncParam{},
		ReturnTypes: []*ParsedFuncParam{},
		Comments:    []string{},
	}
}

func (fn *ParsedFunc) addParam(name string, t string) {
	fn.Params = append(fn.Params, &ParsedFuncParam{
		Name: name,
		Type: t,
	})
}

func (fn *ParsedFunc) addReturnType(name string, t string) {
	fn.ReturnTypes = append(fn.ReturnTypes, &ParsedFuncParam{
		Name: name,
		Type: t,
	})
}

func (fn *ParsedFunc) addComment(s string) {
	fn.Comments = append(fn.Comments, s)
}

func (fn *ParsedFunc) paramsAsInput() string {
	res := ""

	for _, p := range fn.Params {
		if res != "" {
			res += ", "
		}

		res += p.nameAndType()
	}

	return res
}

func (fn *ParsedFunc) paramNames() string {
	res := ""

	for _, p := range fn.Params {
		if res != "" {
			res += ", "
		}

		res += p.Name
	}

	return res
}

func (fn *ParsedFunc) returnTypesString() string {
	res := ""

	for _, p := range fn.ReturnTypes {
		if res != "" {
			res += ", "
		}

		res += p.nameAndType()
	}

	return res
}

func (fn ParsedFunc) generate(w io.Writer, aliasPkg string) {
	for _, comment := range fn.Comments {
		fmt.Fprintln(w, comment)
	}

	fmt.Fprintf(w, "func %s(%s) ", fn.Name, fn.paramsAsInput())
	if len(fn.ReturnTypes) > 1 {
		fmt.Fprint(w, "(")
	}

	fmt.Fprint(w, fn.returnTypesString())
	if len(fn.ReturnTypes) > 1 {
		fmt.Fprint(w, ") ")
	}

	fmt.Fprintln(w, " {")

	fmt.Fprint(w, "\t")

	if len(fn.ReturnTypes) > 0 {
		fmt.Fprint(w, "return ")
	}

	fmt.Fprintf(w, "%s.%s(%s)\n", aliasPkg, fn.Name, fn.paramNames())

	fmt.Fprintln(w, "}")
}

// exprToString converts an AST expression to its string representation.
// It handles different types of expressions like identifiers, selector expressions,
// pointer types, array types, and interfaces.
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.InterfaceType:
		if t.Methods != nil && t.Methods.List != nil && len(t.Methods.List) > 0 {
			return "any{...}"
		}
		return "any"
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", exprToString(t.Key), exprToString(t.Value))
	default:
		return fmt.Sprintf("%#v", expr)
	}
}
