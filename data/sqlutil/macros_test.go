package sqlutil

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	"f":   staticMacro("f(1)"),
	"g":   staticMacro("g(1)"),
	"num": staticMacro("10000"),
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
		name    string
		input   string
		output  string
		wantErr bool
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
			input:  "select $__foo, $__fooBaz from foo",
			output: "select baz, qux from foo",
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
			name:   "default interval",
			input:  "select * from foo group by bin(time, $__interval)",
			output: "select * from foo group by bin(time, 10m)",
		},
		{
			name:   "default interval_ms",
			input:  "select count(*) / $__interval_ms * 1000 as qps from foo",
			output: "select count(*) / 600000 * 1000 as qps from foo",
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
		{
			input:  "select * from foo where $__num*(table.a + table.b) > 1000000",
			output: "select * from foo where 10000*(table.a + table.b) > 1000000",
			name:   "don't consume args after other non-paren characters",
		},
		{
			input:   "select * from foo where $__params(whoops",
			name:    "error (not panic) on missing close bracket",
			wantErr: true,
		},
		{
			input:   "select * from foo where $__params(FUNC(foo, bar)",
			name:    "error on missing close bracket, nested",
			wantErr: true,
		},
		{
			input:   "select * from foo where $__params(FUNC(foo, bar)) > $__timeTo(uhoh",
			name:    "error on missing close bracket after good macros",
			wantErr: true,
		},
	}
	for i, tc := range tests {
		t.Run(fmt.Sprintf("[%d/%d] %s", i+1, len(tests), tc.name), func(t *testing.T) {
			query := &Query{
				RawSQL:   tc.input,
				Table:    tableName,
				Column:   tableColumn,
				Interval: time.Duration(10) * time.Minute,
			}
			interpolatedQuery, err := Interpolate(query, macros)
			assert.Equal(t, err != nil, tc.wantErr, "wantErr != gotErr")
			if !tc.wantErr {
				assert.Equal(t, tc.output, interpolatedQuery)
			}
		})
	}
}

func Test_parseArgs(t *testing.T) {
	var tests = []struct {
		name       string
		input      string
		wantArgs   []string
		wantLength int
	}{
		{
			name:       "no parens, no args",
			input:      "foo bar",
			wantArgs:   nil,
			wantLength: 0,
		},
		{
			name:       "parens not at beginning, still no args",
			input:      "foo(bar)",
			wantArgs:   nil,
			wantLength: 0,
		},
		{
			name:       "even just a space is enough",
			input:      " (bar)",
			wantArgs:   nil,
			wantLength: 0,
		},
		{
			name:       "simple one-arg case",
			input:      "(bar)",
			wantArgs:   []string{"bar"},
			wantLength: 5,
		},
		{
			name:       "multiple args, spaces",
			input:      "(bar, baz, quux)",
			wantArgs:   []string{"bar", "baz", "quux"},
			wantLength: 16,
		},
		{
			name:       "nested parens are not parsed further",
			input:      "(bar(some,thing))",
			wantArgs:   []string{"bar(some,thing)"},
			wantLength: 17,
		},
		{
			name:       "stuff after the closing bracket is ignored",
			input:      "(arg1, arg2), not_an_arg, nope",
			wantArgs:   []string{"arg1", "arg2"},
			wantLength: 12,
		},
		{
			name:       "missing close bracket is an error",
			input:      "(arg1, arg2",
			wantArgs:   nil,
			wantLength: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, length := parseArgs(tt.input)
			assert.Equalf(t, tt.wantArgs, args, "parseArgs(%v) wrong args:\nwant: %v\ngot:  %v\n", tt.input, tt.wantArgs, args)
			assert.Equalf(t, tt.wantLength, length, "parseArgs(%v) wrong length:\nwant: %d\ngot:  %d\n", tt.input, tt.wantLength, length)
		})
	}
}

func TestGetMacroMatches(t *testing.T) {
	t.Run("getMacroMatches applies DefaultMacros", func(t *testing.T) {
		for macroName := range DefaultMacros {
			matches, err := getMacroMatches(fmt.Sprintf("$__%s", macroName), macroName)

			assert.NoError(t, err)
			assert.Equal(t, []macroMatch{{fmt.Sprintf("$__%s", macroName), nil}}, matches)
		}
	})
	t.Run("does not return matches for macro name which is substring", func(t *testing.T) {
		matches, err := getMacroMatches("$__timeFilterEpoch(time_column)", "timeFilter")

		assert.NoError(t, err)
		assert.Nil(t, matches)
	})
}
