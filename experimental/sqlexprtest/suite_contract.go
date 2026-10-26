package sqlexprtest

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

// Contract states the SQL input a scenario expects. An empty Mode, or a nil
// Rows or Columns, is not checked. A non-nil empty Columns slice means no
// columns. Otherwise the names, order, field types and SQL types must match
// exactly. MaxCells limits the cells of one input; zero or negative means no
// limit. It doesn't check Grafana's limit, which counts all inputs together,
// or the size of the SQL output.
type Contract struct {
	Mode     Mode
	Columns  []Column
	Rows     *int
	MaxCells int64
}

// Cells returns the predicted input cells (rows multiplied by columns), or zero
// when conversion fails. It saturates at MaxInt64 if the count overflows.
func (r *Report) Cells() int64 {
	if r.Mode == ModeNone || r.Rows <= 0 {
		return 0
	}
	rows, columns := int64(r.Rows), int64(len(r.Columns))
	if columns != 0 && rows > math.MaxInt64/columns {
		return math.MaxInt64
	}
	return rows * columns
}

func validateContract(c Contract) error {
	if c.Mode != "" && c.Mode != ModeNone && c.Mode != ModeTable && c.Mode != ModeFullLong {
		return fmt.Errorf("unknown conversion mode %q", c.Mode)
	}
	if c.Rows != nil && *c.Rows < 0 {
		return fmt.Errorf("expected row count must not be negative")
	}
	if c.Mode == "" && c.Columns == nil && c.Rows == nil && c.MaxCells <= 0 {
		return fmt.Errorf("provide a mode, schema, row-count, or positive cell-limit expectation")
	}
	return nil
}

func compareContract(r *Report, c Contract) []string {
	var problems []string
	if c.Mode != "" && r.Mode != c.Mode {
		problems = append(problems, fmt.Sprintf("conversion mode: want %s, got %s", c.Mode, r.Mode))
	}
	if c.Rows != nil && r.Rows != *c.Rows {
		problems = append(problems, fmt.Sprintf("rows: want %d, got %d", *c.Rows, r.Rows))
	}
	if c.Columns != nil && !slices.Equal(c.Columns, r.Columns) {
		problems = append(problems, fmt.Sprintf("ordered columns: want %+v, got %+v", c.Columns, r.Columns))
	}
	if r.Rows < 0 {
		problems = append(problems, "report has a negative row count")
	}
	if r.Mode != ModeNone && len(r.Columns) > 0 && int64(r.Rows) > math.MaxInt64/int64(len(r.Columns)) {
		problems = append(problems, "predicted cell count overflows int64")
	}
	if c.MaxCells > 0 && r.Cells() > c.MaxCells {
		problems = append(problems, fmt.Sprintf("input cells: %d exceeds scenario's per-input limit %d", r.Cells(), c.MaxCells))
	}
	return problems
}

// Allowance accepts a known finding within one scenario. Code and Reason are
// required. Empty RefID and nil Frame/Field match any value; otherwise matching
// is exact. Field matches Finding.Field, including generated column names.
// Same-named fields within one frame cannot be distinguished by this selector.
// The first matching allowance in declaration order accepts a finding; redundant
// overlapping allowances fail when they remain unused. Findings below the
// failure threshold do not consume allowances, so severity downgrades expose
// obsolete suppressions.
// Unmatched allowances fail the scenario, and matched findings remain visible.
type Allowance struct {
	Code   Code
	RefID  string
	Frame  *int
	Field  *string
	Reason string
}

func validateAllowances(allowances []Allowance, want map[string]Contract) error {
	for i, allowance := range allowances {
		if !slices.Contains(Codes(), allowance.Code) {
			return fmt.Errorf("allowance %d has unknown code %q", i, allowance.Code)
		}
		if strings.TrimSpace(allowance.Reason) == "" {
			return fmt.Errorf("allowance %d needs a reason", i)
		}
		if allowance.RefID != "" {
			if _, exists := want[allowance.RefID]; !exists {
				return fmt.Errorf("allowance %d refers to undeclared RefID %q", i, allowance.RefID)
			}
		}
		if allowance.Frame != nil && *allowance.Frame < -1 {
			return fmt.Errorf("allowance %d has invalid frame index", i)
		}
	}
	return nil
}

func matchAllowance(refID string, finding Finding, allowances []Allowance, used []bool) (int, bool) {
	for i, a := range allowances {
		if a.Code != finding.Code || a.RefID != "" && a.RefID != refID || a.Frame != nil && *a.Frame != finding.Frame || a.Field != nil && *a.Field != finding.Field {
			continue
		}
		used[i] = true
		return i, true
	}
	return 0, false
}
