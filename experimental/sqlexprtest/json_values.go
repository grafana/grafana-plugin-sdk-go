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

// roundedJSONInteger compares mathematically integral JSON numbers with the
// exact integer represented by their decoded float64. Fractional decimals are
// not diagnosed: ordinary binary floating-point approximation is expected.
func roundedJSONInteger(raw string, decoded float64) bool {
	if math.Abs(decoded) < maxExactInteger {
		return false
	}
	raw = strings.TrimPrefix(raw, "-")
	mantissa, exponent := raw, 0
	if i := strings.IndexAny(raw, "eE"); i >= 0 {
		mantissa = raw[:i]
		var err error
		exponent, err = strconv.Atoi(raw[i+1:])
		if err != nil {
			return false
		}
	}
	fractionalDigits := 0
	if i := strings.IndexByte(mantissa, '.'); i >= 0 {
		fractionalDigits = len(mantissa) - i - 1
		mantissa = mantissa[:i] + mantissa[i+1:]
	}
	digits := strings.TrimLeft(mantissa, "0")
	trimmed := strings.TrimRight(digits, "0")
	if trimmed == "" {
		return false
	}
	// A finite float64 has at most 309 integer digits. Bound the exponent
	// before arithmetic or allocation, including untrusted huge exponents.
	if exponent < -len(raw) || exponent > len(raw)+309 {
		return false
	}
	scale := exponent - fractionalDigits + len(digits) - len(trimmed)
	if scale < 0 || scale > 309 || len(trimmed) > 309-scale {
		return false
	}
	exact := trimmed + strings.Repeat("0", scale)
	return exact != strconv.FormatFloat(math.Abs(decoded), 'f', 0, 64)
}
