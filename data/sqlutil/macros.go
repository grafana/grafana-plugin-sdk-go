package sqlutil

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/exp/maps"
)

var (
	// ErrorBadArgumentCount is returned from macros when the wrong number of arguments were provided
	ErrorBadArgumentCount = errors.New("unexpected number of arguments")
)

// MacroFunc defines a signature for applying a query macro
// Query macro implementations are defined by users/consumers of this package
type MacroFunc func(*Query, []string) (string, error)

// Macros is a map of macro name to MacroFunc. The name must be regex friendly.
type Macros map[string]MacroFunc

// Default time filter for SQL based on the query time range.
// It requires one argument, the time column to filter.
// Example:
//
//	$__timeFilter(time) => "time BETWEEN '2006-01-02T15:04:05Z07:00' AND '2006-01-02T15:04:05Z07:00'"
func macroTimeFilter(query *Query, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("%w: expected 1 argument, received %d", ErrorBadArgumentCount, len(args))
	}

	var (
		column = args[0]
		from   = query.TimeRange.From.UTC().Format(time.RFC3339)
		to     = query.TimeRange.To.UTC().Format(time.RFC3339)
	)

	return fmt.Sprintf("%s >= '%s' AND %s <= '%s'", column, from, column, to), nil
}

// Default time filter for SQL based on the starting query time range.
// It requires one argument, the time column to filter.
// Example:
//
//	$__timeFrom(time) => "time > '2006-01-02T15:04:05Z07:00'"
func macroTimeFrom(query *Query, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("%w: expected 1 argument, received %d", ErrorBadArgumentCount, len(args))
	}

	return fmt.Sprintf("%s >= '%s'", args[0], query.TimeRange.From.UTC().Format(time.RFC3339)), nil
}

// Default time filter for SQL based on the ending query time range.
// It requires one argument, the time column to filter.
// Example:
//
//	$__timeTo(time) => "time < '2006-01-02T15:04:05Z07:00'"
func macroTimeTo(query *Query, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("%w: expected 1 argument, received %d", ErrorBadArgumentCount, len(args))
	}

	return fmt.Sprintf("%s <= '%s'", args[0], query.TimeRange.To.UTC().Format(time.RFC3339)), nil
}

// Default time group for SQL based the given period.
// This basic example is meant to be customized with more complex periods.
// It requires two arguments, the column to filter and the period.
// Example:
//
//	$__timeGroup(time, month) => "datepart(year, time), datepart(month, time)'"
func macroTimeGroup(_ *Query, args []string) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("%w: expected 1 argument, received %d", ErrorBadArgumentCount, len(args))
	}

	res := ""
	switch args[1] {
	case "minute":
		res += fmt.Sprintf("datepart(minute, %s),", args[0])
		fallthrough
	case "hour":
		res += fmt.Sprintf("datepart(hour, %s),", args[0])
		fallthrough
	case "day":
		res += fmt.Sprintf("datepart(day, %s),", args[0])
		fallthrough
	case "month":
		res += fmt.Sprintf("datepart(month, %s),", args[0])
		fallthrough
	case "year":
		res += fmt.Sprintf("datepart(year, %s)", args[0])
	}

	return res, nil
}

// Default macro to return the query table name.
// Example:
//
//	$__table => "my_table"
func macroTable(query *Query, _ []string) (string, error) {
	return query.Table, nil
}

// Default macro to return the query column name.
// Example:
//
//	$__column => "my_col"
func macroColumn(query *Query, _ []string) (string, error) {
	return query.Column, nil
}

var DefaultMacros = Macros{
	"timeFilter": macroTimeFilter,
	"timeFrom":   macroTimeFrom,
	"timeGroup":  macroTimeGroup,
	"timeTo":     macroTimeTo,
	"table":      macroTable,
	"column":     macroColumn,
}

type Macro struct {
	Name string
	Args []string
}

// getMacroMatches extracts macro strings with their respective arguments from the sql input given
// It manually parses the string to find the closing parenthesis of the macro (because regex has no memory)
func getMacroMatches(input string, name string) ([]Macro, error) {
	macroName := fmt.Sprintf("\\$__%s\\b", name)
	matchedMacros := []Macro{}
	rgx, err := regexp.Compile(macroName)

	if err != nil {
		return nil, err
	}

	// get all matching macro instances
	matched := rgx.FindAllStringIndex(input, -1)

	if matched == nil {
		return nil, nil
	}

	for matchedIndex := 0; matchedIndex < len(matched); matchedIndex++ {
		var macroEnd = 0
		var argStart = 0
		// quick exit from the loop, when we encounter a closing bracket before an opening one (ie "($__macro)", where we can skip the closing one from the result)
		var forceBreak = false
		macroStart := matched[matchedIndex][0]
		inputCopy := input[macroStart:]
		cache := make([]rune, 0)

		// find the opening and closing arguments brackets
		for idx, r := range inputCopy {
			if len(cache) == 0 && macroEnd > 0 || forceBreak {
				break
			}
			switch r {
			case '(':
				cache = append(cache, r)
				if argStart == 0 {
					argStart = idx + 1
				}
			case ' ':
				// when we are inside an argument, we do not want to exit on space
				if argStart != 0 {
					continue
				}
				fallthrough
			case ')':
				l := len(cache)
				if l == 0 {
					macroEnd = 0
					forceBreak = true
					break
				}
				cache = cache[:l-1]
				macroEnd = idx + 1
			default:
				continue
			}
		}

		// macroEnd equals to 0 means there are no parentheses, so just set it
		// to the end of the regex match
		if macroEnd == 0 {
			macroEnd = matched[matchedIndex][1] - macroStart
		}
		macroString := inputCopy[0:macroEnd]
		macroMatch := Macro{Name: macroString}

		args := ""
		// if opening parenthesis was found, extract contents as arguments
		if argStart > 0 {
			args = inputCopy[argStart : macroEnd-1]
		}
		macroMatch.Args = parseArgs(args)
		matchedMacros = append(matchedMacros, macroMatch)
	}
	return matchedMacros, nil
}

func parseArgs(args string) []string {
	argsArray := []string{}
	phrase := []rune{}
	bracketCount := 0
	for _, v := range args {
		phrase = append(phrase, v)
		if v == '(' {
			bracketCount++
			continue
		}
		if v == ')' {
			bracketCount--
			continue
		}
		if v == ',' && bracketCount == 0 {
			removeComma := phrase[:len(phrase)-1]
			argsArray = append(argsArray, string(removeComma))
			phrase = []rune{}
		}
	}
	argsArray = append(argsArray, strings.TrimSpace(string(phrase)))
	return argsArray
}

// Interpolate returns an interpolated query string given a backend.DataQuery
func Interpolate(query *Query, macros Macros) (string, error) {
	mergedMacros := Macros{}
	maps.Copy(mergedMacros, DefaultMacros)
	maps.Copy(mergedMacros, macros)

	rawSQL := query.RawSQL

	for key, macro := range mergedMacros {
		matches, err := getMacroMatches(rawSQL, key)
		if err != nil {
			return rawSQL, err
		}
		if len(matches) == 0 {
			continue
		}

		for _, match := range matches {
			res, err := macro(query.WithSQL(rawSQL), match.Args)
			if err != nil {
				return rawSQL, err
			}

			rawSQL = strings.ReplaceAll(rawSQL, match.Name, res)
		}
	}

	return rawSQL, nil
}
