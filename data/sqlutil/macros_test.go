package sqlutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func staticMacro(output string) MacroFunc {
	return argMacro(func([]string) string { return output })
}

func argMacro(f func([]string) string) MacroFunc {
	return func(_ *Query, args []string) (string, error) { return f(args), nil }
}

var macros = Macros{
	"foo":    staticMacro("baz"),
	"fooBaz": staticMacro("qux"),
	"params": argMacro(func(args []string) string {
		if args[0] != "" {
			return "bar_" + args[0]
		}
		return "bar"
	}),
	"f": staticMacro("f(1)"),
	"g": staticMacro("g(1)"),
	"multiParams": argMacro(func(args []string) string {
		return strings.Join(append([]string{"bar"}, args...), "_")
	}),
	// overwrite a default macro
	"timeGroup": staticMacro("grouped!"),
}

func TestInterpolate(t *testing.T) {
	tableName := "my_table"
	tableColumn := "my_col"
	type test struct {
		name   string
		input  string
		output string
	}
	tests := []test{
		{
			name:   "macro with incorrect syntax",
			input:  "select * from foo",
			output: "select * from foo",
		},
		{
			name:   "correct macro",
			input:  "select * from $__foo()",
			output: "select * from baz",
		},
		{
			name:   "this macro name's substring is another macro",
			input:  "select * from $__fooBaz()",
			output: "select * from qux",
		},
		{
			name:   "multiple instances of same macro",
			input:  "select '$__foo()' from $__foo()",
			output: "select 'baz' from baz",
		},
		{
			name:   "multiple instances of same macro without space",
			input:  "select * from $__foo()$__foo()",
			output: "select * from bazbaz",
		},
		{
			name:   "macro without parenthesis",
			input:  "select * from $__foo",
			output: "select * from baz",
		},
		{
			name:   "macro without params",
			input:  "select * from $__params()",
			output: "select * from bar",
		},
		{
			name:   "with param",
			input:  "select * from $__params(hello)",
			output: "select * from bar_hello",
		},
		{
			name:   "with short param",
			input:  "select * from $__params(h)",
			output: "select * from bar_h",
		},
		{
			name:   "same macro multiple times with same param",
			input:  "select * from $__params(hello) AND $__params(hello)",
			output: "select * from bar_hello AND bar_hello",
		},
		{
			name:   "same macro multiple times with same param and additional parentheses",
			input:  "(select * from $__params(hello) AND $__params(hello))",
			output: "(select * from bar_hello AND bar_hello)",
		},
		{
			name:   "same macro multiple times with different param",
			input:  "select * from $__params(hello) AND $__params(world)",
			output: "select * from bar_hello AND bar_world",
		},
		{
			name:   "different macros with different params",
			input:  "select * from $__params(world) AND $__foo() AND $__params(hello)",
			output: "select * from bar_world AND baz AND bar_hello",
		},
		{
			name:   "default timeFilter",
			input:  "select * from foo where $__timeFilter(time)",
			output: "select * from foo where time >= '0001-01-01T00:00:00Z' AND time <= '0001-01-01T00:00:00Z'",
		},
		{
			name:   "default timeFilter with function",
			input:  "select * from foo where $__timeFilter(cast(sth as timestamp))",
			output: "select * from foo where cast(sth as timestamp) >= '0001-01-01T00:00:00Z' AND cast(sth as timestamp) <= '0001-01-01T00:00:00Z'",
		},
		{
			name:   "default timeFilter with empty spaces",
			input:  "select * from foo where $__timeFilter(cast(sth as timestamp) )",
			output: "select * from foo where cast(sth as timestamp) >= '0001-01-01T00:00:00Z' AND cast(sth as timestamp) <= '0001-01-01T00:00:00Z'",
		},
		{
			name:   "default timeTo macro",
			input:  "select * from foo where $__timeTo(time)",
			output: "select * from foo where time <= '0001-01-01T00:00:00Z'",
		},
		{
			name:   "default timeFrom macro",
			input:  "select * from foo where $__timeFrom(time)",
			output: "select * from foo where time >= '0001-01-01T00:00:00Z'",
		},
		{
			name:   "default timeFrom macro with function",
			input:  "select * from foo where $__timeFrom(cast(sth as timestamp))",
			output: "select * from foo where cast(sth as timestamp) >= '0001-01-01T00:00:00Z'",
		},
		{
			name:   "overridden timeGroup macro",
			input:  "select * from foo where $__timeGroup(time,minute)",
			output: "select * from foo where grouped!",
		},
		{
			name:   "table and column macros",
			input:  "select $__column from $__table",
			output: "select my_col from my_table",
		},
		{
			name:   "macro functions inside more complex clauses",
			input:  "select * from table where ( datetime >= $__foo() ) AND ( datetime <= $__foo() ) limit 100",
			output: "select * from table where ( datetime >= baz ) AND ( datetime <= baz ) limit 100",
		},
		{
			name:   "macros inside more complex clauses",
			input:  "select * from table where ( datetime >= $__foo ) AND ( datetime <= $__foo ) limit 100",
			output: "select * from table where ( datetime >= baz ) AND ( datetime <= baz ) limit 100",
		},
		{
			input:  "select * from foo where $__multiParams(foo, bar)",
			output: "select * from foo where bar_foo_bar",
			name:   "macro with multiple parameters",
		},
		{
			input:  "select * from foo where $__params(FUNC(foo, bar))",
			output: "select * from foo where bar_FUNC(foo, bar)",
			name:   "function in macro with multiple parameters",
		},
		{
			input:  "select * from foo where $__f > $__g",
			output: "select * from foo where f(1) > g(1)",
			name:   "don't consume args after a space (see https://github.com/grafana/sqlds/issues/82)",
		},
	}
	for i, tc := range tests {
		t.Run(fmt.Sprintf("[%d/%d] %s", i+1, len(tests), tc.name), func(t *testing.T) {
			query := &Query{
				RawSQL: tc.input,
				Table:  tableName,
				Column: tableColumn,
			}
			interpolatedQuery, err := Interpolate(query, macros)
			require.Nil(t, err)
			assert.Equal(t, tc.output, interpolatedQuery)
		})
	}
}

func TestGetMacroMatches(t *testing.T) {
	t.Run("getMacroMatches applies DefaultMacros", func(t *testing.T) {
		for macroName := range DefaultMacros {
			matches, err := getMacroMatches(fmt.Sprintf("$__%s", macroName), macroName)

			assert.NoError(t, err)
			assert.Equal(t, []Macro{{fmt.Sprintf("$__%s", macroName), []string{""}}}, matches)
		}
	})
	t.Run("does not return matches for macro name which is substring", func(t *testing.T) {
		matches, err := getMacroMatches("$__timeFilterEpoch(time_column)", "timeFilter")

		assert.NoError(t, err)
		assert.Nil(t, matches)
	})
}
