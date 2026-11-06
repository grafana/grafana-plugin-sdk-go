package sqlexprtest

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

// checkJSONNumbers receives syntactically valid JSON. Match the SQL reader's
// float64 decoding first: even an overflowing value under a duplicate object
// key fails the whole document. UseNumber then retains the surviving numbers'
// original text so integer rounding can be checked without float64 loss.
// outOfRange counts failed JSON documents; rounded counts changed integers.
func checkJSONNumbers(raw json.RawMessage) (outOfRange, rounded int) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return 1, 0
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return 0, 0 // The same input was successfully decoded above.
	}
	return 0, roundedJSONIntegers(value)
}

func roundedJSONIntegers(value any) int {
	switch value := value.(type) {
	case json.Number:
		decoded, err := value.Float64()
		if err == nil && roundedJSONInteger(value.String(), decoded) {
			return 1
		}
	case []any:
		count := 0
		for _, item := range value {
			count += roundedJSONIntegers(item)
		}
		return count
	case map[string]any:
		count := 0
		for _, item := range value {
			count += roundedJSONIntegers(item)
		}
		return count
	}
	return 0
}

// roundedJSONInteger reports whether an integer literal, written without a
// fraction or exponent, changes when decoded as float64. A number with a
// fraction or exponent, such as 6.02214076e23, is a float, and float64
// approximation of it is expected.
func roundedJSONInteger(raw string, decoded float64) bool {
	if math.Abs(decoded) < maxExactInteger || strings.ContainsAny(raw, ".eE") {
		return false
	}
	return strings.TrimPrefix(raw, "-") != strconv.FormatFloat(math.Abs(decoded), 'f', 0, 64)
}
