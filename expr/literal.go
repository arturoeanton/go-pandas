package expr

import "fmt"

// LiteralExpr wraps a constant value.
type LiteralExpr struct {
	Value any
}

// Lit builds a literal expression.
func Lit(v any) LiteralExpr { return LiteralExpr{Value: v} }

func (l LiteralExpr) Eval(row map[string]any) (any, error) { return l.Value, nil }

func (l LiteralExpr) String() string {
	if s, ok := l.Value.(string); ok {
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprint(l.Value)
}
