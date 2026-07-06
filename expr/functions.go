package expr

import (
	"fmt"
	"math"
	"strings"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// funcExpr applies a named function to an inner expression.
type funcExpr struct {
	name  string
	inner Expr
	f     func(v any) (any, error)
}

func (e funcExpr) Eval(row map[string]any) (any, error) {
	v, err := e.inner.Eval(row)
	if err != nil {
		return nil, err
	}
	if dtype.IsNA(v) {
		return nil, nil
	}
	return e.f(v)
}

func (e funcExpr) String() string { return fmt.Sprintf("%s(%s)", e.name, e.inner) }

func mathFunc(name string, inner Expr, f func(x float64) float64) Expr {
	return funcExpr{name: name, inner: inner, f: func(v any) (any, error) {
		x, err := asFloat(v)
		if err != nil {
			return nil, err
		}
		return f(x), nil
	}}
}

// Abs returns |e|.
func Abs(e Expr) Expr { return mathFunc("abs", e, math.Abs) }

// Sqrt returns the square root of e.
func Sqrt(e Expr) Expr { return mathFunc("sqrt", e, math.Sqrt) }

// Log returns the natural logarithm of e.
func Log(e Expr) Expr { return mathFunc("log", e, math.Log) }

// Exp returns e**x.
func Exp(e Expr) Expr { return mathFunc("exp", e, math.Exp) }

func stringFunc(name string, inner Expr, f func(s string) any) Expr {
	return funcExpr{name: name, inner: inner, f: func(v any) (any, error) {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s requires a string, got %T", errs.ErrTypeMismatch, name, v)
		}
		return f(s), nil
	}}
}

// Lower lowercases a string expression.
func Lower(e Expr) Expr {
	return stringFunc("lower", e, func(s string) any { return strings.ToLower(s) })
}

// Upper uppercases a string expression.
func Upper(e Expr) Expr {
	return stringFunc("upper", e, func(s string) any { return strings.ToUpper(s) })
}

// Len returns the length of a string expression.
func Len(e Expr) Expr {
	return stringFunc("len", e, func(s string) any { return len(s) })
}
