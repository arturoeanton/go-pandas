package expr

import (
	"fmt"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// CompareValues orders two non-missing values of compatible kinds. It
// returns -1, 0 or 1, and false when the values are not comparable.
func CompareValues(a, b any) (int, bool) {
	if fa, ok := dtype.AsFloat(a); ok {
		if fb, ok := dtype.AsFloat(b); ok {
			switch {
			case fa < fb:
				return -1, true
			case fa > fb:
				return 1, true
			}
			return 0, true
		}
		return 0, false
	}
	if sa, ok := a.(string); ok {
		if sb, ok := b.(string); ok {
			switch {
			case sa < sb:
				return -1, true
			case sa > sb:
				return 1, true
			}
			return 0, true
		}
		return 0, false
	}
	if ta, ok := a.(time.Time); ok {
		if tb, ok := timeComparand(b); ok {
			switch {
			case ta.Before(tb):
				return -1, true
			case ta.After(tb):
				return 1, true
			}
			return 0, true
		}
		return 0, false
	}
	if tb, ok := b.(time.Time); ok {
		if ta, ok := timeComparand(a); ok {
			switch {
			case ta.Before(tb):
				return -1, true
			case ta.After(tb):
				return 1, true
			}
			return 0, true
		}
		return 0, false
	}
	if ba, ok := a.(bool); ok {
		if bb, ok := b.(bool); ok {
			switch {
			case !ba && bb:
				return -1, true
			case ba && !bb:
				return 1, true
			}
			return 0, true
		}
	}
	return 0, false
}

// timeComparand widens datetime comparisons: a time.Time passes
// through, and a string parseable by the deterministic ToDatetime
// inference list compares as its timestamp (v0.10) — so
// `date >= "2026-01-01"` works in Query/Where like pandas.
func timeComparand(v any) (time.Time, bool) {
	switch x := v.(type) {
	case time.Time:
		return x, true
	case string:
		for _, layout := range dtype.InferTimeLayouts {
			if t, err := time.Parse(layout, x); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

// EqualValues reports loose equality across numeric widths, strings, bools
// and times.
func EqualValues(a, b any) bool {
	if c, ok := CompareValues(a, b); ok {
		return c == 0
	}
	return a == b
}

// comparePred implements >, >=, <, <=, ==, != between two expressions.
// Comparisons involving missing values evaluate to false, like pandas.
type comparePred struct {
	left  Expr
	right Expr
	op    string
}

func (p comparePred) Eval(row map[string]any) (any, error) { return p.EvalBool(row) }

func (p comparePred) EvalBool(row map[string]any) (bool, error) {
	lv, err := p.left.Eval(row)
	if err != nil {
		return false, err
	}
	rv, err := p.right.Eval(row)
	if err != nil {
		return false, err
	}
	if dtype.IsNA(lv) || dtype.IsNA(rv) {
		return false, nil
	}
	switch p.op {
	case "==":
		return EqualValues(lv, rv), nil
	case "!=":
		return !EqualValues(lv, rv), nil
	}
	c, ok := CompareValues(lv, rv)
	if !ok {
		return false, fmt.Errorf("%w: cannot compare %T with %T", errs.ErrTypeMismatch, lv, rv)
	}
	switch p.op {
	case ">":
		return c > 0, nil
	case ">=":
		return c >= 0, nil
	case "<":
		return c < 0, nil
	case "<=":
		return c <= 0, nil
	}
	return false, fmt.Errorf("%w: unknown comparison %q", errs.ErrInvalidOperation, p.op)
}

func (p comparePred) String() string {
	return fmt.Sprintf("(%s %s %s)", p.left, p.op, p.right)
}
