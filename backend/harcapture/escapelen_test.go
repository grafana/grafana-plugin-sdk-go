package harcapture

import (
	"encoding/json"
	"testing"
)

// marshaledLen returns the byte length of s once json.Marshal encodes it as a JSON string value,
// the ground truth jsonEscapedLen must never fall short of.
func marshaledLen(t *testing.T, s string) int64 {
	t.Helper()
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal(%q): %v", s, err)
	}
	return int64(len(raw))
}

func TestJSONEscapedLen_matchesOrExceedsRealMarshal(t *testing.T) {
	cases := map[string]string{
		"empty":                  "",
		"plain":                  "hello world",
		"quote and backslash":    "quote \" and backslash \\",
		"short escapes":          "newline\ntab\treturn\rbackspace\bformfeed\f",
		"other control bytes":    string([]byte{0x00, 0x01, 0x02, 0x1f}), // -> \u00XX
		"html unsafe":            "<script>&amp;</script>",
		"line/paragraph sep":     "before\u2028middle\u2029after",
		"multibyte no escaping":  "café éèê",
		"astral plane (emoji)":   "emoji \U0001F600 rocket \U0001F680",
		"invalid utf8":           string([]byte{0xff, 0xfe, 0x80}),
		"valid bytes around bad": string([]byte{'a', 0xff, 'b'}),
		"literal replacement":    "�",
		"mixed":                  "mixed <tag> and \"quotes\" and \bbell and \u2028sep and 日本語",
	}

	for name, s := range cases {
		t.Run(name, func(t *testing.T) {
			estimate := jsonEscapedLen(s)
			real := marshaledLen(t, s)
			if estimate < real {
				t.Errorf("jsonEscapedLen(%q) = %d, underestimates real marshaled length %d", s, estimate, real)
			}
		})
	}
}

// FuzzJSONEscapedLen proves the safety invariant across arbitrary input, not just the hand-picked
// cases above: jsonEscapedLen must never fall below the real json.Marshal length, for any string.
func FuzzJSONEscapedLen(f *testing.F) {
	for _, seed := range []string{
		"", "a", "\"", "\\", "\n", "\t", "<>&", "\u2028\u2029", "日本語", "\U0001F600",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		estimate := jsonEscapedLen(s)
		raw, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("json.Marshal(%q): %v", s, err)
		}
		if real := int64(len(raw)); estimate < real {
			t.Errorf("jsonEscapedLen(%q) = %d, underestimates real marshaled length %d", s, estimate, real)
		}
	})
}
