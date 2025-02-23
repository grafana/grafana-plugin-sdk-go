package gtime

import (
	"testing"
)

// go test -benchmem -run=^$ -bench=BenchmarkParse$ github.com/grafana/grafana-plugin-sdk-go/backend/gtime/ -memprofile p_mem.out -count 6 | tee pmem.0.txt
func BenchmarkParse(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"PureNumber", "30"},
		{"SimpleUnit", "5s"},
		{"ComplexUnit", "1h30m"},
		{"DateUnit", "7d"},
		{"MonthUnit", "3M"},
		{"YearUnit", "1y"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, _ = parse(tc.input)
			}
		})
	}
}
