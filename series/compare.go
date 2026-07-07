package series

import (
	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// cmp builds a boolean series from a per-position predicate. Missing
// values compare as false, like pandas.
func (s *Series) cmp(name string, f func(v any) bool) *Series {
	return s.boolSeries(s.name, func(i int) bool {
		if s.col.IsNA(i) {
			return false
		}
		return f(s.col.Value(i))
	})
}

// allFalse implements the uniform NA comparison rule: any comparison
// against a missing comparand is false everywhere (including Ne — a
// documented difference from pandas' NaN != x).
func (s *Series) allFalse() *Series {
	return s.boolSeries(s.name, func(int) bool { return false })
}

// Eq returns s == v elementwise. v may be a scalar or another *Series.
func (s *Series) Eq(v any) *Series {
	if o, ok := v.(*Series); ok {
		return s.cmpSeries(o, func(a, b any) bool { return expr.EqualValues(a, b) })
	}
	if dtype.IsNA(v) {
		return s.allFalse()
	}
	return s.cmp("eq", func(x any) bool { return expr.EqualValues(x, v) })
}

// Ne returns s != v elementwise.
func (s *Series) Ne(v any) *Series {
	if o, ok := v.(*Series); ok {
		return s.cmpSeries(o, func(a, b any) bool { return !expr.EqualValues(a, b) })
	}
	if dtype.IsNA(v) {
		return s.allFalse()
	}
	return s.cmp("ne", func(x any) bool { return !expr.EqualValues(x, v) })
}

func ordCmp(a, b any, ok func(c int) bool) bool {
	c, comparable := expr.CompareValues(a, b)
	return comparable && ok(c)
}

// catCompare intercepts scalar ordered comparisons on categorical
// storage: ordered categoricals compare by category rank (not label
// value); unordered categoricals are incomparable, so the uniform
// incomparable-is-false rule applies. Cat() accessor comparisons and the
// expr engine surface the explicit ErrInvalidOperation instead.
func (s *Series) catCompare(v any, satisfied func(c int) bool) (*Series, bool) {
	cc, ok := column.AsCategorical(s.col)
	if !ok {
		return nil, false
	}
	if !cc.Ordered() {
		return s.allFalse(), true
	}
	return s.catOrderedCompare(cc, v, satisfied), true
}

// Gt returns s > v elementwise.
func (s *Series) Gt(v any) *Series {
	if o, ok := v.(*Series); ok {
		return s.cmpSeries(o, func(a, b any) bool { return ordCmp(a, b, func(c int) bool { return c > 0 }) })
	}
	if out, ok := s.catCompare(v, func(c int) bool { return c > 0 }); ok {
		return out
	}
	return s.cmp("gt", func(x any) bool { return ordCmp(x, v, func(c int) bool { return c > 0 }) })
}

// Ge returns s >= v elementwise.
func (s *Series) Ge(v any) *Series {
	if o, ok := v.(*Series); ok {
		return s.cmpSeries(o, func(a, b any) bool { return ordCmp(a, b, func(c int) bool { return c >= 0 }) })
	}
	if out, ok := s.catCompare(v, func(c int) bool { return c >= 0 }); ok {
		return out
	}
	return s.cmp("ge", func(x any) bool { return ordCmp(x, v, func(c int) bool { return c >= 0 }) })
}

// Lt returns s < v elementwise.
func (s *Series) Lt(v any) *Series {
	if o, ok := v.(*Series); ok {
		return s.cmpSeries(o, func(a, b any) bool { return ordCmp(a, b, func(c int) bool { return c < 0 }) })
	}
	if out, ok := s.catCompare(v, func(c int) bool { return c < 0 }); ok {
		return out
	}
	return s.cmp("lt", func(x any) bool { return ordCmp(x, v, func(c int) bool { return c < 0 }) })
}

// Le returns s <= v elementwise.
func (s *Series) Le(v any) *Series {
	if o, ok := v.(*Series); ok {
		return s.cmpSeries(o, func(a, b any) bool { return ordCmp(a, b, func(c int) bool { return c <= 0 }) })
	}
	if out, ok := s.catCompare(v, func(c int) bool { return c <= 0 }); ok {
		return out
	}
	return s.cmp("le", func(x any) bool { return ordCmp(x, v, func(c int) bool { return c <= 0 }) })
}

func (s *Series) cmpSeries(other *Series, f func(a, b any) bool) *Series {
	return s.boolSeries(s.name, func(i int) bool {
		if s.col.IsNA(i) || i >= other.Len() || other.col.IsNA(i) {
			return false
		}
		return f(s.col.Value(i), other.col.Value(i))
	})
}

// Between checks left <= s <= right; inclusive is one of "both" (default),
// "neither", "left" or "right".
func (s *Series) Between(left, right any, inclusive string) *Series {
	lo := func(x any) bool { return ordCmp(x, left, func(c int) bool { return c >= 0 }) }
	hi := func(x any) bool { return ordCmp(x, right, func(c int) bool { return c <= 0 }) }
	switch inclusive {
	case "neither":
		lo = func(x any) bool { return ordCmp(x, left, func(c int) bool { return c > 0 }) }
		hi = func(x any) bool { return ordCmp(x, right, func(c int) bool { return c < 0 }) }
	case "left":
		hi = func(x any) bool { return ordCmp(x, right, func(c int) bool { return c < 0 }) }
	case "right":
		lo = func(x any) bool { return ordCmp(x, left, func(c int) bool { return c > 0 }) }
	}
	return s.cmp("between", func(x any) bool { return lo(x) && hi(x) })
}

// IsIn checks membership against a list of values.
func (s *Series) IsIn(values ...any) *Series {
	return s.cmp("isin", func(x any) bool {
		for _, v := range values {
			if expr.EqualValues(x, v) {
				return true
			}
		}
		return false
	})
}

// AsMask converts a Bool series to []bool (missing -> false). Bool
// columns take a buffer fast path without boxing (v0.4.1).
func (s *Series) AsMask() []bool {
	if data, mask, ok := column.Bools(s.col); ok {
		out := make([]bool, len(data))
		for i, v := range data {
			out[i] = v && !mask[i]
		}
		return out
	}
	out := make([]bool, s.Len())
	for i := range out {
		if s.col.IsNA(i) {
			continue
		}
		if b, ok := s.col.Value(i).(bool); ok {
			out[i] = b
		}
	}
	return out
}
