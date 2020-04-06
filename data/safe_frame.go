package data

import "fmt"

// SafeFrame is an alternative type of Frame that has methods that will
// check conditions and returns error where the equivilently named methods
// on Frame would panic.
type SafeFrame Frame

// AppendRow is the same as Frame.AppendRow() but will return an error
// under conditions where Frame.AppendRow() would panic.
func (f *SafeFrame) AppendRow(vals ...interface{}) error {
	if len(vals) != len(f.Fields) {
		return fmt.Errorf("failed to append vals to Frame. Frame has %v fields but was given %v to append", len(f.Fields), len(vals))
	}
	// check validity before any modification
	for i, v := range vals {
		if f.Fields[i] == nil || f.Fields[i].vector == nil {
			return fmt.Errorf("can not append to uninitalized Field at field index %v", i)
		}
		dfPType := f.Fields[i].Type()
		if v == nil {
			if !dfPType.Nullable() {
				return fmt.Errorf("can not append nil to non-nullable vector with underlying type %s at field index %v", dfPType, i)
			}
		}
		if v != nil && fieldTypeFromVal(v) != dfPType {
			return fmt.Errorf("invalid type appending row at index %v, got %T want %v", i, v, dfPType.ItemTypeString())
		}
		f.Fields[i].vector.Append(v)
	}
	return nil
}
