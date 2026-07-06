package expr

import (
	"fmt"
	"math"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// binaryExpr implements arithmetic between two expressions. String
// concatenation is supported for "+" on two strings. Missing operands
// propagate NA (nil result).
type binaryExpr struct {
	left  Expr
	right Expr
	op    string
}

func (b binaryExpr) Eval(row map[string]any) (any, error) {
	lv, err := b.left.Eval(row)
	if err != nil {
		return nil, err
	}
	rv, err := b.right.Eval(row)
	if err != nil {
		return nil, err
	}
	if dtype.IsNA(lv) || dtype.IsNA(rv) {
		return nil, nil
	}
	if b.op == "+" {
		if ls, ok := lv.(string); ok {
			if rs, ok := rv.(string); ok {
				return ls + rs, nil
			}
		}
	}
	lf, err := asFloat(lv)
	if err != nil {
		return nil, err
	}
	rf, err := asFloat(rv)
	if err != nil {
		return nil, err
	}
	var out float64
	switch b.op {
	case "+":
		out = lf + rf
	case "-":
		out = lf - rf
	case "*":
		out = lf * rf
	case "/":
		out = lf / rf
	case "%":
		out = math.Mod(lf, rf)
	case "**":
		out = math.Pow(lf, rf)
	default:
		return nil, fmt.Errorf("%w: unknown operator %q", errs.ErrInvalidOperation, b.op)
	}
	// Keep integer results as int64 when both inputs were integers and the
	// operation is closed over integers.
	if b.op != "/" && b.op != "**" && isIntegral(lv) && isIntegral(rv) && out == math.Trunc(out) {
		return int64(out), nil
	}
	return out, nil
}

func isIntegral(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	}
	return false
}

func (b binaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", b.left, b.op, b.right)
}
