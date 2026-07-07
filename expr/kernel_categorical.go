package expr

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// categoricalCompare compares a categorical column against a scalar
// label purely on codes (v0.7): the label resolves to its code once and
// every row is an int32 comparison. Ordered operators require an ordered
// categorical — unordered ones surface ErrInvalidOperation (a real
// error, not ErrNotColumnar: the row fallback would silently compare
// labels lexically, which is not categorical semantics).
func categoricalCompare(op string, cc *column.CategoricalColumn, scalar any, n int) (*Mask, error) {
	out := newMask(n)
	codes, mask := cc.RawCodes()
	target := cc.CodeOf(scalar) // -1: not a category — matches nothing
	switch op {
	case "==", "!=":
		ne := op == "!="
		for i := 0; i < n; i++ {
			if mask[i] {
				out.NA[i] = true
				continue
			}
			out.Data[i] = (codes[i] == target) != ne
		}
		return out, nil
	}
	if !cc.Ordered() {
		return nil, fmt.Errorf("%w: ordered comparison on unordered categorical", errs.ErrInvalidOperation)
	}
	for i := 0; i < n; i++ {
		if mask[i] {
			out.NA[i] = true
			continue
		}
		if target < 0 {
			continue // unknown label: incomparable, stays false
		}
		switch {
		case codes[i] < target:
			out.Data[i] = cmpSatisfied(op, -1)
		case codes[i] > target:
			out.Data[i] = cmpSatisfied(op, 1)
		default:
			out.Data[i] = cmpSatisfied(op, 0)
		}
	}
	return out, nil
}

// flipCompareOp mirrors an operator for swapped operands (5 < x ≡ x > 5).
func flipCompareOp(op string) string {
	switch op {
	case "<":
		return ">"
	case "<=":
		return ">="
	case ">":
		return "<"
	case ">=":
		return "<="
	}
	return op // == and != are symmetric
}
