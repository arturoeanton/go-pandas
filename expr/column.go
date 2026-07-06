package expr

import (
	"fmt"
	"strings"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// ColumnExpr references a DataFrame column by name.
type ColumnExpr struct {
	name string
}

// Col builds a column reference, the entry point of most expressions.
func Col(name string) ColumnExpr { return ColumnExpr{name: name} }

// Name returns the referenced column name.
func (c ColumnExpr) Name() string { return c.name }

func (c ColumnExpr) Eval(row map[string]any) (any, error) {
	v, ok := row[c.name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errs.ErrColumnNotFound, c.name)
	}
	return v, nil
}

func (c ColumnExpr) String() string { return "col(" + c.name + ")" }

// Comparisons -----------------------------------------------------------

func (c ColumnExpr) Eq(v any) Predicate { return comparePred{left: c, right: asExpr(v), op: "=="} }
func (c ColumnExpr) Ne(v any) Predicate { return comparePred{left: c, right: asExpr(v), op: "!="} }
func (c ColumnExpr) Gt(v any) Predicate { return comparePred{left: c, right: asExpr(v), op: ">"} }
func (c ColumnExpr) Ge(v any) Predicate { return comparePred{left: c, right: asExpr(v), op: ">="} }
func (c ColumnExpr) Lt(v any) Predicate { return comparePred{left: c, right: asExpr(v), op: "<"} }
func (c ColumnExpr) Le(v any) Predicate { return comparePred{left: c, right: asExpr(v), op: "<="} }

// Between is inclusive on both ends, like Series.between default.
func (c ColumnExpr) Between(left, right any) Predicate {
	return And(c.Ge(left), c.Le(right))
}

// IsNA is true when the column value is missing.
func (c ColumnExpr) IsNA() Predicate {
	return funcPred{name: "isna", inner: c, f: func(v any) (bool, error) {
		return dtype.IsNA(v), nil
	}}
}

// NotNA is true when the column value is present.
func (c ColumnExpr) NotNA() Predicate {
	return funcPred{name: "notna", inner: c, f: func(v any) (bool, error) {
		return !dtype.IsNA(v), nil
	}}
}

// IsIn is true when the column value equals one of the given values.
func (c ColumnExpr) IsIn(values ...any) Predicate {
	vals := append([]any(nil), values...)
	return funcPred{name: "isin", inner: c, f: func(v any) (bool, error) {
		if dtype.IsNA(v) {
			return false, nil
		}
		for _, cand := range vals {
			if EqualValues(v, cand) {
				return true, nil
			}
		}
		return false, nil
	}}
}

func (c ColumnExpr) stringPred(name string, f func(s string) bool) Predicate {
	return funcPred{name: name, inner: c, f: func(v any) (bool, error) {
		if dtype.IsNA(v) {
			return false, nil
		}
		s, ok := v.(string)
		if !ok {
			return false, fmt.Errorf("%w: %s requires string column, got %T", errs.ErrTypeMismatch, name, v)
		}
		return f(s), nil
	}}
}

// Contains is true when the string column contains substr.
func (c ColumnExpr) Contains(substr string) Predicate {
	return c.stringPred("contains", func(s string) bool { return strings.Contains(s, substr) })
}

// StartsWith is true when the string column starts with prefix.
func (c ColumnExpr) StartsWith(prefix string) Predicate {
	return c.stringPred("startswith", func(s string) bool { return strings.HasPrefix(s, prefix) })
}

// EndsWith is true when the string column ends with suffix.
func (c ColumnExpr) EndsWith(suffix string) Predicate {
	return c.stringPred("endswith", func(s string) bool { return strings.HasSuffix(s, suffix) })
}

// Arithmetic ------------------------------------------------------------

func (c ColumnExpr) Add(v any) Expr { return binaryExpr{left: c, right: asExpr(v), op: "+"} }
func (c ColumnExpr) Sub(v any) Expr { return binaryExpr{left: c, right: asExpr(v), op: "-"} }
func (c ColumnExpr) Mul(v any) Expr { return binaryExpr{left: c, right: asExpr(v), op: "*"} }
func (c ColumnExpr) Div(v any) Expr { return binaryExpr{left: c, right: asExpr(v), op: "/"} }
func (c ColumnExpr) Mod(v any) Expr { return binaryExpr{left: c, right: asExpr(v), op: "%"} }
func (c ColumnExpr) Pow(v any) Expr { return binaryExpr{left: c, right: asExpr(v), op: "**"} }

// funcPred adapts a single-value boolean function into a Predicate.
type funcPred struct {
	name  string
	inner Expr
	f     func(v any) (bool, error)
}

func (p funcPred) Eval(row map[string]any) (any, error) { return p.EvalBool(row) }

func (p funcPred) EvalBool(row map[string]any) (bool, error) {
	v, err := p.inner.Eval(row)
	if err != nil {
		return false, err
	}
	return p.f(v)
}

func (p funcPred) String() string { return fmt.Sprintf("%s(%s)", p.name, p.inner) }
