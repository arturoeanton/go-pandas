package column

import "time"

// Typed buffer accessors used by the columnar expression engine (v0.4).
// The returned slices alias internal storage and must be treated as
// read-only; ok is false when the column has a different backing.

// Strings extracts the string buffer plus mask of a string column.
func Strings(c Column) ([]string, []bool, bool) {
	if tc, ok := c.(*typedColumn[string]); ok {
		return tc.data, tc.mask, true
	}
	return nil, nil, false
}

// Bools extracts the bool buffer plus mask of a bool column.
func Bools(c Column) ([]bool, []bool, bool) {
	if tc, ok := c.(*typedColumn[bool]); ok {
		return tc.data, tc.mask, true
	}
	return nil, nil, false
}

// Times extracts the time buffer plus mask of a datetime column.
func Times(c Column) ([]time.Time, []bool, bool) {
	if tc, ok := c.(*typedColumn[time.Time]); ok {
		return tc.data, tc.mask, true
	}
	return nil, nil, false
}
