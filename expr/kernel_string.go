package expr

import (
	"strings"

	"github.com/arturoeanton/go-pandas/internal/column"
)

// stringOperand extracts the string buffer of a column operand.
func stringOperand(c column.Column) ([]string, []bool, error) {
	vals, mask, ok := column.Strings(c)
	if !ok {
		return nil, nil, ErrNotColumnar
	}
	return vals, mask, nil
}

// stringCompare runs ==, !=, >, >=, <, <= lexicographically over string
// operands, matching the row evaluator's CompareValues.
func stringCompare(op string, left, right colValue, n int) (*Mask, error) {
	out := newMask(n)
	cmp := func(x, y string) bool {
		switch {
		case x < y:
			return cmpSatisfied(op, -1)
		case x > y:
			return cmpSatisfied(op, 1)
		default:
			return cmpSatisfied(op, 0)
		}
	}
	switch {
	case right.isScalar():
		vals, mask, err := stringOperand(left.col)
		if err != nil {
			return nil, err
		}
		v, ok := right.scalar.(string)
		if !ok {
			return nil, ErrNotColumnar
		}
		for i := 0; i < n; i++ {
			if mask[i] {
				out.NA[i] = true
				continue
			}
			out.Data[i] = cmp(vals[i], v)
		}
	case left.isScalar():
		vals, mask, err := stringOperand(right.col)
		if err != nil {
			return nil, err
		}
		v, ok := left.scalar.(string)
		if !ok {
			return nil, ErrNotColumnar
		}
		for i := 0; i < n; i++ {
			if mask[i] {
				out.NA[i] = true
				continue
			}
			out.Data[i] = cmp(v, vals[i])
		}
	default:
		lv, lm, err := stringOperand(left.col)
		if err != nil {
			return nil, err
		}
		rv, rm, err := stringOperand(right.col)
		if err != nil {
			return nil, err
		}
		for i := 0; i < n; i++ {
			if lm[i] || rm[i] {
				out.NA[i] = true
				continue
			}
			out.Data[i] = cmp(lv[i], rv[i])
		}
	}
	return out, nil
}

// stringMatchKernel evaluates contains/startswith/endswith over a string
// column. NA strings produce NA predicate results (dropped by filters).
func stringMatchKernel(name string, c column.Column, arg string, n int) (*Mask, error) {
	vals, mask, err := stringOperand(c)
	if err != nil {
		return nil, err
	}
	var match func(s string) bool
	switch name {
	case "contains":
		match = func(s string) bool { return strings.Contains(s, arg) }
	case "startswith":
		match = func(s string) bool { return strings.HasPrefix(s, arg) }
	case "endswith":
		match = func(s string) bool { return strings.HasSuffix(s, arg) }
	default:
		return nil, ErrNotColumnar
	}
	out := newMask(n)
	for i := 0; i < n; i++ {
		if mask[i] {
			out.NA[i] = true
			continue
		}
		out.Data[i] = match(vals[i])
	}
	return out, nil
}

// stringConcat implements columnar Add over two string operands.
func stringConcat(left, right colValue, n int) (colValue, error) {
	out := make([]string, n)
	mask := make([]bool, n)
	get := func(v colValue) (func(i int) (string, bool), error) {
		if v.isScalar() {
			s, ok := v.scalar.(string)
			if !ok {
				return nil, ErrNotColumnar
			}
			return func(int) (string, bool) { return s, true }, nil
		}
		vals, m, err := stringOperand(v.col)
		if err != nil {
			return nil, err
		}
		return func(i int) (string, bool) { return vals[i], !m[i] }, nil
	}
	lg, err := get(left)
	if err != nil {
		return colValue{}, err
	}
	rg, err := get(right)
	if err != nil {
		return colValue{}, err
	}
	for i := 0; i < n; i++ {
		x, okX := lg(i)
		y, okY := rg(i)
		if !okX || !okY {
			mask[i] = true
			continue
		}
		out[i] = x + y
	}
	return colValue{col: column.NewString(out, mask)}, nil
}

// stringFuncKernel implements Lower/Upper/Len columnar.
func stringFuncKernel(name string, inner colValue, n int) (colValue, error) {
	vals, mask, err := stringOperand(inner.col)
	if err != nil {
		return colValue{}, err
	}
	outMask := append([]bool(nil), mask...)
	switch name {
	case "lower", "upper":
		out := make([]string, n)
		f := strings.ToLower
		if name == "upper" {
			f = strings.ToUpper
		}
		for i := 0; i < n; i++ {
			if outMask[i] {
				continue
			}
			out[i] = f(vals[i])
		}
		return colValue{col: column.NewString(out, outMask)}, nil
	case "len":
		out := make([]int, n)
		for i := 0; i < n; i++ {
			if outMask[i] {
				continue
			}
			out[i] = len(vals[i])
		}
		return colValue{col: column.NewInt(out, outMask)}, nil
	}
	return colValue{}, ErrNotColumnar
}
