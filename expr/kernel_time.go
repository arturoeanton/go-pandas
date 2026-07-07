package expr

import (
	"time"

	"github.com/arturoeanton/go-pandas/internal/column"
)

// timeOperand extracts the time buffer of a column operand.
func timeOperand(c column.Column) ([]time.Time, []bool, error) {
	vals, mask, ok := column.Times(c)
	if !ok {
		return nil, nil, ErrNotColumnar
	}
	return vals, mask, nil
}

func timeCmp(op string, x, y time.Time) bool {
	switch {
	case x.Before(y):
		return cmpSatisfied(op, -1)
	case x.After(y):
		return cmpSatisfied(op, 1)
	default:
		return cmpSatisfied(op, 0)
	}
}

// timeCompare runs the six comparisons over datetime operands.
func timeCompare(op string, left, right colValue, n int) (*Mask, error) {
	out := newMask(n)
	switch {
	case right.isScalar():
		vals, mask, err := timeOperand(left.col)
		if err != nil {
			return nil, err
		}
		v, ok := right.scalar.(time.Time)
		if !ok {
			return nil, ErrNotColumnar
		}
		for i := 0; i < n; i++ {
			if mask[i] {
				out.NA[i] = true
				continue
			}
			out.Data[i] = timeCmp(op, vals[i], v)
		}
	case left.isScalar():
		vals, mask, err := timeOperand(right.col)
		if err != nil {
			return nil, err
		}
		v, ok := left.scalar.(time.Time)
		if !ok {
			return nil, ErrNotColumnar
		}
		for i := 0; i < n; i++ {
			if mask[i] {
				out.NA[i] = true
				continue
			}
			out.Data[i] = timeCmp(op, v, vals[i])
		}
	default:
		lv, lm, err := timeOperand(left.col)
		if err != nil {
			return nil, err
		}
		rv, rm, err := timeOperand(right.col)
		if err != nil {
			return nil, err
		}
		for i := 0; i < n; i++ {
			if lm[i] || rm[i] {
				out.NA[i] = true
				continue
			}
			out.Data[i] = timeCmp(op, lv[i], rv[i])
		}
	}
	return out, nil
}
