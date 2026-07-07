package expr

import (
	"errors"
	"fmt"
)

// PlanKind reports which execution path an expression will take.
type PlanKind int

const (
	// PlanColumnar: typed columnar kernels, no per-row maps.
	PlanColumnar PlanKind = iota
	// PlanFallback: the row-map evaluator runs.
	PlanFallback
	// PlanError: evaluation fails regardless of path (e.g. unknown
	// column).
	PlanError
)

func (k PlanKind) String() string {
	switch k {
	case PlanColumnar:
		return "columnar"
	case PlanFallback:
		return "row-fallback"
	default:
		return "error"
	}
}

// Plan is the diagnostic result of planning an expression against a
// frame's columns.
type Plan struct {
	Kind   PlanKind
	Expr   string
	Reason string
}

func (p *Plan) String() string {
	if p.Reason == "" {
		return fmt.Sprintf("%s: %s", p.Kind, p.Expr)
	}
	return fmt.Sprintf("%s: %s — %s", p.Kind, p.Expr, p.Reason)
}

// PlanPredicate reports whether a predicate takes the columnar path on
// the given context. Planning executes the columnar evaluator once (the
// engine has no separate static analysis pass), so it is intended for
// tests and debugging.
func PlanPredicate(p Predicate, ctx *EvalContext) *Plan {
	_, err := evalMask(p, ctx)
	return planFrom(p.String(), err)
}

// PlanExpr reports the execution path of a value expression.
func PlanExpr(e Expr, ctx *EvalContext) *Plan {
	if p, ok := e.(Predicate); ok {
		return PlanPredicate(p, ctx)
	}
	_, err := evalValue(e, ctx)
	return planFrom(e.String(), err)
}

func planFrom(desc string, err error) *Plan {
	switch {
	case err == nil:
		return &Plan{Kind: PlanColumnar, Expr: desc}
	case errors.Is(err, ErrNotColumnar):
		return &Plan{Kind: PlanFallback, Expr: desc, Reason: err.Error()}
	default:
		return &Plan{Kind: PlanError, Expr: desc, Reason: err.Error()}
	}
}
