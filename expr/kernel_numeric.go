package expr

import (
	"math"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// operandKind buckets an operand for kernel dispatch. Mixed-kind
// comparisons fall back to the row evaluator so behavior stays identical.
type operandKind int

const (
	kindNumeric operandKind = iota
	kindString
	kindTime
	kindOther
)

func kindOfValue(v colValue) operandKind {
	if v.isScalar() {
		switch v.scalar.(type) {
		case string:
			return kindString
		case time.Time:
			return kindTime
		}
		if _, ok := dtype.AsFloat(v.scalar); ok {
			return kindNumeric
		}
		return kindOther
	}
	switch {
	case column.IsObjectBacked(v.col):
		return kindOther
	case v.col.DType() == dtype.String:
		return kindString
	case v.col.DType() == dtype.Time:
		return kindTime
	case dtype.IsNumeric(v.col.DType()) || v.col.DType() == dtype.Bool:
		return kindNumeric
	}
	return kindOther
}

func cmpSatisfied(op string, c int) bool {
	switch op {
	case "==":
		return c == 0
	case "!=":
		return c != 0
	case ">":
		return c > 0
	case ">=":
		return c >= 0
	case "<":
		return c < 0
	case "<=":
		return c <= 0
	}
	return false
}

// evalCompareColumnar dispatches a comparison to the typed kernel for its
// operand kind. Anything mixed or unsupported reports ErrNotColumnar.
func evalCompareColumnar(node comparePred, ctx *EvalContext) (*Mask, error) {
	left, err := evalValue(node.left, ctx)
	if err != nil {
		return nil, err
	}
	right, err := evalValue(node.right, ctx)
	if err != nil {
		return nil, err
	}
	if left.isScalar() && right.isScalar() {
		return nil, ErrNotColumnar
	}
	// An NA comparand makes every comparison false (documented rule).
	if right.isScalar() && dtype.IsNA(right.scalar) || left.isScalar() && dtype.IsNA(left.scalar) {
		return newMask(ctx.Len), nil
	}
	// Categorical column vs scalar label: code kernel (v0.7).
	if right.isScalar() && !left.isScalar() {
		if cc, ok := column.AsCategorical(left.col); ok {
			return categoricalCompare(node.op, cc, right.scalar, ctx.Len)
		}
	}
	if left.isScalar() && !right.isScalar() {
		if cc, ok := column.AsCategorical(right.col); ok {
			return categoricalCompare(flipCompareOp(node.op), cc, left.scalar, ctx.Len)
		}
	}
	// A string comparand against a datetime column parses through the
	// deterministic inference list so the time kernel applies (v0.10).
	if right.isScalar() && !left.isScalar() && left.col.DType() == dtype.Time {
		if s, ok := right.scalar.(string); ok {
			if t, ok := timeComparand(s); ok {
				right.scalar = t
			}
		}
	}
	if left.isScalar() && !right.isScalar() && right.col.DType() == dtype.Time {
		if s, ok := left.scalar.(string); ok {
			if t, ok := timeComparand(s); ok {
				left.scalar = t
			}
		}
	}
	lk, rk := kindOfValue(left), kindOfValue(right)
	if lk != rk || lk == kindOther {
		return nil, ErrNotColumnar
	}
	switch lk {
	case kindNumeric:
		return numericCompare(node.op, left, right, ctx.Len)
	case kindString:
		return stringCompare(node.op, left, right, ctx.Len)
	case kindTime:
		return timeCompare(node.op, left, right, ctx.Len)
	}
	return nil, ErrNotColumnar
}

func numericCompare(op string, left, right colValue, n int) (*Mask, error) {
	out := newMask(n)
	cmp := func(x, y float64) bool {
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
		vals, mask, err := numericOperand(left.col)
		if err != nil {
			return nil, err
		}
		v, ok := scalarFloat(right.scalar)
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
		vals, mask, err := numericOperand(right.col)
		if err != nil {
			return nil, err
		}
		v, ok := scalarFloat(left.scalar)
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
		lv, lm, err := numericOperand(left.col)
		if err != nil {
			return nil, err
		}
		rv, rm, err := numericOperand(right.col)
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

// evalBinaryColumnar runs arithmetic over typed buffers. Integer inputs
// with closed operations produce Int64 columns; division and non-integral
// results produce Float64 — mirroring the row evaluator element rules.
func evalBinaryColumnar(node binaryExpr, ctx *EvalContext) (colValue, error) {
	left, err := evalValue(node.left, ctx)
	if err != nil {
		return colValue{}, err
	}
	right, err := evalValue(node.right, ctx)
	if err != nil {
		return colValue{}, err
	}
	if left.isScalar() && right.isScalar() {
		return colValue{}, ErrNotColumnar
	}
	lk, rk := kindOfValue(left), kindOfValue(right)
	// String concatenation via Add.
	if node.op == "+" && lk == kindString && rk == kindString {
		return stringConcat(left, right, ctx.Len)
	}
	// An NA scalar operand makes the whole result missing (row rule:
	// NA op x -> nil).
	if (left.isScalar() && dtype.IsNA(left.scalar)) || (right.isScalar() && dtype.IsNA(right.scalar)) {
		mask := make([]bool, ctx.Len)
		for i := range mask {
			mask[i] = true
		}
		return colValue{col: column.NewFloat64(make([]float64, ctx.Len), mask)}, nil
	}
	if lk != kindNumeric || rk != kindNumeric {
		return colValue{}, ErrNotColumnar
	}

	extract := func(v colValue) (vals []float64, mask []bool, scalar float64, isScalar, integral bool, err error) {
		if v.isScalar() {
			f, ok := scalarFloat(v.scalar)
			if !ok {
				return nil, nil, 0, true, false, ErrNotColumnar
			}
			return nil, nil, f, true, isIntegral(v.scalar), nil
		}
		vals, mask, e := numericOperand(v.col)
		integral = dtype.IsInteger(v.col.DType())
		return vals, mask, 0, false, integral, e
	}
	lv, lm, ls, lIsS, lInt, err := extract(left)
	if err != nil {
		return colValue{}, err
	}
	rv, rm, rs, rIsS, rInt, err := extract(right)
	if err != nil {
		return colValue{}, err
	}

	f := arithFn(node.op)
	n := ctx.Len
	out := make([]float64, n)
	mask := make([]bool, n)
	allIntegral := lInt && rInt && node.op != "/"
	for i := 0; i < n; i++ {
		x, y := ls, rs
		if !lIsS {
			if lm[i] {
				mask[i] = true
				continue
			}
			x = lv[i]
		}
		if !rIsS {
			if rm[i] {
				mask[i] = true
				continue
			}
			y = rv[i]
		}
		out[i] = f(x, y)
		if allIntegral && out[i] != math.Trunc(out[i]) {
			allIntegral = false
		}
	}
	if allIntegral {
		data := make([]int64, n)
		for i := range out {
			data[i] = int64(out[i])
		}
		return colValue{col: column.NewInt64(data, mask)}, nil
	}
	return colValue{col: column.NewFloat64(out, mask)}, nil
}

func arithFn(op string) func(x, y float64) float64 {
	switch op {
	case "+":
		return func(x, y float64) float64 { return x + y }
	case "-":
		return func(x, y float64) float64 { return x - y }
	case "*":
		return func(x, y float64) float64 { return x * y }
	case "/":
		return func(x, y float64) float64 { return x / y }
	case "%":
		return math.Mod
	case "**":
		return math.Pow
	}
	return func(x, y float64) float64 { return math.NaN() }
}

// evalFuncColumnar handles the expression functions (Abs, Sqrt, Lower...)
// registered with columnar metadata.
func evalFuncColumnar(node funcExpr, ctx *EvalContext) (colValue, error) {
	inner, err := evalValue(node.inner, ctx)
	if err != nil {
		return colValue{}, err
	}
	if inner.isScalar() {
		return colValue{}, ErrNotColumnar
	}
	switch node.name {
	case "abs", "sqrt", "log", "exp":
		vals, mask, err := numericOperand(inner.col)
		if err != nil {
			return colValue{}, err
		}
		var f func(float64) float64
		switch node.name {
		case "abs":
			f = math.Abs
		case "sqrt":
			f = math.Sqrt
		case "log":
			f = math.Log
		case "exp":
			f = math.Exp
		}
		n := ctx.Len
		out := make([]float64, n)
		outMask := append([]bool(nil), mask...)
		for i := 0; i < n; i++ {
			if outMask[i] {
				continue
			}
			out[i] = f(vals[i])
		}
		return colValue{col: column.NewFloat64(out, outMask)}, nil
	case "lower", "upper", "len":
		return stringFuncKernel(node.name, inner, ctx.Len)
	}
	return colValue{}, ErrNotColumnar
}
