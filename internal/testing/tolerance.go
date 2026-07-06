// Package testutil provides the assertion helpers used by the golden
// compatibility tests: frame/series/array comparison with pandas-style
// NA handling and floating point tolerance.
package testutil

import "math"

// DefaultAbsTolerance and DefaultRelTolerance bound the accepted
// difference between go-pandas and pandas/NumPy floating point results.
const (
	DefaultAbsTolerance = 1e-9
	DefaultRelTolerance = 1e-9
)

// AllClose reports whether two floats are equal within the default
// tolerances; NaN equals NaN (golden semantics).
func AllClose(a, b float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	if math.IsInf(a, 1) && math.IsInf(b, 1) {
		return true
	}
	if math.IsInf(a, -1) && math.IsInf(b, -1) {
		return true
	}
	diff := math.Abs(a - b)
	if diff <= DefaultAbsTolerance {
		return true
	}
	return diff <= DefaultRelTolerance*math.Max(math.Abs(a), math.Abs(b))
}
