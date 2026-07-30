package sqlutil

import (
	"database/sql"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// unwrapWrapperTypes recursively unwraps ClickHouse-style wrapper types, e.g.
// "SimpleAggregateFunction(max, Float64)" -> "Float64",
// "LowCardinality(String)" -> "String", including nested combinations such as
// "SimpleAggregateFunction(anyLast, LowCardinality(Float64))" -> "Float64".
func unwrapWrapperTypes(dbType string) string {
	for {
		switch {
		case strings.HasPrefix(dbType, "LowCardinality(") && strings.HasSuffix(dbType, ")"):
			dbType = dbType[len("LowCardinality(") : len(dbType)-1]
		case strings.HasPrefix(dbType, "SimpleAggregateFunction(") && strings.HasSuffix(dbType, ")"):
			inner := dbType[len("SimpleAggregateFunction(") : len(dbType)-1]
			depth := 0
			cut := -1
			for i, ch := range inner {
				switch {
				case ch == '(':
					depth++
				case ch == ')':
					depth--
				case ch == ',' && depth == 0:
					cut = i
				}
				if cut >= 0 {
					break
				}
			}
			if cut < 0 {
				return dbType
			}
			dbType = strings.TrimSpace(inner[cut+1:])
		default:
			return dbType
		}
	}
}

func TestConverterMatchesWithInputTypeMatcher(t *testing.T) {
	float64Converter := Converter{
		Name: "Float64",
		// An unset InputTypeName exact-matches an empty DatabaseTypeName, which
		// would short-circuit the empty-type-name case before the matcher runs,
		// so pin it to a value that never matches.
		InputTypeName: "sentinel-never-matches",
		InputTypeMatcher: func(dbType string) bool {
			return unwrapWrapperTypes(dbType) == "Float64"
		},
	}

	tests := []struct {
		dbType string
		want   bool
	}{
		{"Float64", true},
		{"SimpleAggregateFunction(max, Float64)", true},
		{"LowCardinality(Float64)", true},
		{"SimpleAggregateFunction(anyLast, LowCardinality(Float64))", true},
		{"Float32", false},
		{"Map(String, Float64)", false},
		{"", false},
	}
	for _, tt := range tests {
		name := tt.dbType
		if name == "" {
			name = "empty type name"
		}
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tt.want, converterMatches(float64Converter, tt.dbType, "col"))
		})
	}
}

func TestConverterMatchesExistingPredicatesWithMatcher(t *testing.T) {
	tests := []struct {
		name    string
		conv    Converter
		dbType  string
		colName string
		want    bool
	}{
		{
			name: "nil matcher and no other predicate does not match",
			conv: Converter{},
			// zero-value predicates must not match arbitrary types
			dbType: "Float64", colName: "col", want: false,
		},
		{
			name:   "type name still matches with nil matcher",
			conv:   Converter{InputTypeName: "Float64"},
			dbType: "Float64", colName: "col", want: true,
		},
		{
			name:   "regex still matches with nil matcher",
			conv:   Converter{InputTypeRegex: regexp.MustCompile(`^Float`)},
			dbType: "Float64", colName: "col", want: true,
		},
		{
			name:   "column name still matches with nil matcher",
			conv:   Converter{InputColumnName: "col"},
			dbType: "Float64", colName: "col", want: true,
		},
		{
			name: "matcher matches when other predicates do not",
			conv: Converter{
				InputTypeName:    "Other",
				InputTypeMatcher: func(string) bool { return true },
			},
			dbType: "Float64", colName: "col", want: true,
		},
		{
			name: "non-matching matcher does not prevent a type name match",
			conv: Converter{
				InputTypeName:    "Float64",
				InputTypeMatcher: func(string) bool { return false },
			},
			dbType: "Float64", colName: "col", want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, converterMatches(tt.conv, tt.dbType, tt.colName))
		})
	}
}

func TestMakeScanRowMatcherPrecedence(t *testing.T) {
	matcherConverter := Converter{
		Name:             "matcher",
		InputScanType:    reflect.TypeOf(""),
		InputTypeMatcher: func(string) bool { return true },
	}
	regexConverter := Converter{
		Name:           "regex",
		InputScanType:  reflect.TypeOf(""),
		InputTypeRegex: regexp.MustCompile(`.*`),
	}

	// The first matching converter in slice order wins, regardless of which
	// mechanism (matcher or regex) it matched with.
	rc, err := MakeScanRow([]*sql.ColumnType{nil}, []string{"a"}, matcherConverter, regexConverter)
	require.NoError(t, err)
	require.Len(t, rc.Converters, 1)
	require.Equal(t, "matcher", rc.Converters[0].Name)

	rc, err = MakeScanRow([]*sql.ColumnType{nil}, []string{"a"}, regexConverter, matcherConverter)
	require.NoError(t, err)
	require.Len(t, rc.Converters, 1)
	require.Equal(t, "regex", rc.Converters[0].Name)
}
