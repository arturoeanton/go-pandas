package expr

import (
	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// evalLogicalColumnar combines child masks with three-valued (Kleene)
// logic: false && anything == false and true || anything == true even
// when the other side is NA; otherwise NA operands produce NA results.
// Filters then drop NA rows, matching the row evaluator (whose NA
// comparisons are false).
func evalLogicalColumnar(node logicalPred, ctx *EvalContext) (*Mask, error) {
	switch node.op {
	case "not":
		m, err := evalMask(node.preds[0], ctx)
		if err != nil {
			return nil, err
		}
		out := newMask(ctx.Len)
		copy(out.NA, m.NA)
		for i := range m.Data {
			if !m.NA[i] {
				out.Data[i] = !m.Data[i]
			}
		}
		return out, nil
	case "and":
		return combineMasks(node.preds, ctx, true)
	case "or":
		return combineMasks(node.preds, ctx, false)
	}
	return nil, ErrNotColumnar
}

func combineMasks(preds []Predicate, ctx *EvalContext, isAnd bool) (*Mask, error) {
	acc, err := evalMask(preds[0], ctx)
	if err != nil {
		return nil, err
	}
	// Work on copies so child masks are never mutated.
	out := newMask(ctx.Len)
	copy(out.Data, acc.Data)
	copy(out.NA, acc.NA)
	for _, p := range preds[1:] {
		m, err := evalMask(p, ctx)
		if err != nil {
			return nil, err
		}
		for i := 0; i < ctx.Len; i++ {
			aVal, aNA := out.Data[i], out.NA[i]
			bVal, bNA := m.Data[i], m.NA[i]
			if isAnd {
				switch {
				case (!aNA && !aVal) || (!bNA && !bVal): // definitive false
					out.Data[i], out.NA[i] = false, false
				case aNA || bNA:
					out.Data[i], out.NA[i] = false, true
				default:
					out.Data[i], out.NA[i] = aVal && bVal, false
				}
			} else {
				switch {
				case (!aNA && aVal) || (!bNA && bVal): // definitive true
					out.Data[i], out.NA[i] = true, false
				case aNA || bNA:
					out.Data[i], out.NA[i] = false, true
				default:
					out.Data[i], out.NA[i] = aVal || bVal, false
				}
			}
		}
	}
	return out, nil
}

// evalFuncPredColumnar handles the predicate functions that carry
// columnar metadata: isna/notna, isin and the string matchers.
func evalFuncPredColumnar(node funcPred, ctx *EvalContext) (*Mask, error) {
	inner, err := evalValue(node.inner, ctx)
	if err != nil {
		return nil, err
	}
	if inner.isScalar() {
		return nil, ErrNotColumnar
	}
	c := inner.col
	n := ctx.Len
	switch node.name {
	case "isna", "notna":
		out := newMask(n)
		want := node.name == "isna"
		for i := 0; i < n; i++ {
			out.Data[i] = c.IsNA(i) == want
		}
		return out, nil
	case "isin":
		return isInKernel(c, node.values, n)
	case "contains", "startswith", "endswith":
		return stringMatchKernel(node.name, c, node.strArg, n)
	}
	return nil, ErrNotColumnar
}

// isInKernel matches column values against the candidate list. Numeric
// columns build a float set; string columns a string set; other typed
// columns fall back.
func isInKernel(c column.Column, values []any, n int) (*Mask, error) {
	if values == nil {
		return nil, ErrNotColumnar
	}
	out := newMask(n)
	if vals, mask, ok := column.Strings(c); ok {
		set := make(map[string]bool, len(values))
		for _, v := range values {
			if s, isStr := v.(string); isStr {
				set[s] = true
			}
		}
		for i := 0; i < n; i++ {
			if mask[i] {
				out.NA[i] = true
				continue
			}
			out.Data[i] = set[vals[i]]
		}
		return out, nil
	}
	if column.IsObjectBacked(c) {
		return nil, ErrNotColumnar
	}
	if vals, mask, ok := c.Float64s(); ok {
		set := make(map[float64]bool, len(values))
		for _, v := range values {
			if f, isNum := dtype.AsFloat(v); isNum && !dtype.IsNA(v) {
				set[f] = true
			}
		}
		for i := 0; i < n; i++ {
			if mask[i] {
				out.NA[i] = true
				continue
			}
			out.Data[i] = set[vals[i]]
		}
		return out, nil
	}
	return nil, ErrNotColumnar
}

// evalWhereColumnar computes Where(cond, x, y): the condition runs
// columnar; branch values are picked per row and the result column is
// re-inferred, matching the row evaluator's dtype behavior. NA
// conditions pick y (row rule: NA predicate evaluates false).
func evalWhereColumnar(node whereExpr, ctx *EvalContext) (colValue, error) {
	cond, err := evalMask(node.cond, ctx)
	if err != nil {
		return colValue{}, err
	}
	x, err := evalValue(node.x, ctx)
	if err != nil {
		return colValue{}, err
	}
	y, err := evalValue(node.y, ctx)
	if err != nil {
		return colValue{}, err
	}
	pick := func(v colValue, i int) any {
		if v.isScalar() {
			return v.scalar
		}
		if v.col.IsNA(i) {
			return nil
		}
		return v.col.Value(i)
	}
	values := make([]any, ctx.Len)
	for i := 0; i < ctx.Len; i++ {
		if cond.Data[i] && !cond.NA[i] {
			values[i] = pick(x, i)
		} else {
			values[i] = pick(y, i)
		}
	}
	return colValue{col: column.Infer(values)}, nil
}
