package dtype

import "math"

// NAMarker is the sentinel for a generic missing value, comparable to
// pandas.NA.
type NAMarker struct{}

func (NAMarker) String() string { return "<NA>" }

// NaTMarker is the sentinel for a missing datetime, comparable to
// pandas.NaT.
type NaTMarker struct{}

func (NaTMarker) String() string { return "NaT" }

// NA returns the generic missing value marker.
func NA() any { return NAMarker{} }

// NaT returns the missing datetime marker.
func NaT() any { return NaTMarker{} }

// IsNA reports whether v represents a missing value. nil, NA(), NaT() and
// floating point NaN are all missing. The empty string is NOT missing.
func IsNA(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case NAMarker, *NAMarker:
		return true
	case NaTMarker, *NaTMarker:
		return true
	case float64:
		return math.IsNaN(x)
	case float32:
		return math.IsNaN(float64(x))
	}
	return false
}

// NotNA is the negation of IsNA.
func NotNA(v any) bool { return !IsNA(v) }

// IsNull is an alias of IsNA (pandas exposes both spellings).
func IsNull(v any) bool { return IsNA(v) }

// NotNull is an alias of NotNA.
func NotNull(v any) bool { return NotNA(v) }
