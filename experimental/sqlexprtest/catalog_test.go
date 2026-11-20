package sqlexprtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeCatalogMatchesDeclarations(t *testing.T) {
	// Inspect declarations so adding a constant without registering it fails here.
	f, err := parser.ParseFile(token.NewFileSet(), "report.go", nil, 0)
	require.NoError(t, err)
	var declared []Code
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, spec := range g.Specs {
			v := spec.(*ast.ValueSpec)
			if !strings.HasPrefix(v.Names[0].Name, "Code") {
				continue
			}
			require.Len(t, v.Names, 1)
			require.Len(t, v.Values, 1)
			literal, ok := v.Values[0].(*ast.BasicLit)
			require.True(t, ok)
			value, err := strconv.Unquote(literal.Value)
			require.NoError(t, err)
			declared = append(declared, Code(value))
		}
	}
	require.NotEmpty(t, declared)
	assert.Equal(t, declared, Codes())
	seen := map[Code]bool{}
	for _, c := range Codes() {
		assert.NotEmpty(t, c)
		assert.False(t, seen[c], "duplicate code %s", c)
		seen[c] = true
	}
	copy := Codes()
	copy[0] = "changed"
	assert.Equal(t, declared, Codes(), "callers cannot mutate the catalog")
}
