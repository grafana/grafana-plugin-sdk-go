package spec

import (
	"io/fs"
	gopath "path"
	"path/filepath"
	"strings"

	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
)

type EnumValue struct {
	Value   string
	Comment string
}

type EnumField struct {
	Package string
	Name    string
	Comment string
	Values  []EnumValue
}

func findEnumFields(base, path string) ([]EnumField, error) {
	fset := token.NewFileSet()
	dict := make(map[string][]*ast.Package)
	err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			d, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			for _, v := range d {
				// paths may have multiple packages, like for tests
				k := gopath.Join(base, path)
				dict[k] = append(dict[k], v)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	fields := make([]EnumField, 0)
	field := &EnumField{}
	dp := &doc.Package{}

	for pkg, p := range dict {
		for _, f := range p {
			gtxt := ""
			typ := ""
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.TypeSpec:
					typ = x.Name.String()
					if !ast.IsExported(typ) {
						typ = ""
					} else {
						txt := x.Doc.Text()
						if txt == "" && gtxt != "" {
							txt = gtxt
							gtxt = ""
						}
						txt = strings.TrimSpace(dp.Synopsis(txt))
						if strings.HasSuffix(txt, "+enum") {
							fields = append(fields, EnumField{
								Package: pkg,
								Name:    typ,
								Comment: strings.TrimSpace(strings.TrimSuffix(txt, "+enum")),
							})
							field = &fields[len(fields)-1]
						}
					}
				case *ast.ValueSpec:
					txt := x.Doc.Text()
					if txt == "" {
						txt = x.Comment.Text()
					}
					if typ == field.Name {
						for _, n := range x.Names {
							if ast.IsExported(n.String()) {
								v, ok := x.Values[0].(*ast.BasicLit)
								if ok {
									val := strings.TrimPrefix(v.Value, `"`)
									val = strings.TrimSuffix(val, `"`)
									txt = strings.TrimSpace(txt)
									field.Values = append(field.Values, EnumValue{
										Value:   val,
										Comment: txt,
									})
								}
							}
						}
					}
				case *ast.GenDecl:
					// remember for the next type
					gtxt = x.Doc.Text()
				}
				return true
			})
		}
	}

	return fields, nil
}
