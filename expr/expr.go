// Package expr implements the expression system that replaces the Python
// operators pandas users write inside df[...] and df.assign(...):
//
//	df.Where(pd.Col("age").Gt(30))
//	df.AssignExpr("total", pd.Col("price").Mul(pd.Col("qty")))
package expr

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// Expr is a computation over one row of a DataFrame.
type Expr interface {
	Eval(row map[string]any) (any, error)
	String() string
}

// Predicate is a boolean-valued expression usable as a filter.
type Predicate interface {
	Expr
	EvalBool(row map[string]any) (bool, error)
}

// asExpr lifts values into expressions: an Expr passes through, everything
// else becomes a literal.
func asExpr(v any) Expr {
	if e, ok := v.(Expr); ok {
		return e
	}
	return LiteralExpr{Value: v}
}

// asFloat converts an evaluated value to float64 for arithmetic.
func asFloat(v any) (float64, error) {
	if f, ok := dtype.AsFloat(v); ok {
		return f, nil
	}
	return 0, fmt.Errorf("%w: expected numeric value, got %T", errs.ErrTypeMismatch, v)
}
