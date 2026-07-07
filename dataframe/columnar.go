package dataframe

import (
	"errors"
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/internal/column"
	"github.com/arturoeanton/go-pandas/series"
)

// evalContext exposes the frame's typed columns to the expression
// engine.
func (df *DataFrame) evalContext() *expr.EvalContext {
	return &expr.EvalContext{
		Len: df.Len(),
		Column: func(name string) (column.Column, error) {
			i, ok := df.byName[name]
			if !ok {
				return nil, fmt.Errorf("%w: %s", errs.ErrColumnNotFound, name)
			}
			return df.columns[i].Storage(), nil
		},
	}
}

// Plan reports which execution path an expression or predicate takes on
// this frame: "columnar" (typed kernels) or "row-fallback"
// (map[string]any per row). Intended for debugging and tests.
func (df *DataFrame) Plan(e expr.Expr) *expr.Plan {
	return expr.PlanExpr(e, df.evalContext())
}

// whereColumnar attempts the typed mask path; ok is false when the row
// fallback must run.
func (df *DataFrame) whereColumnar(pred expr.Predicate) (*DataFrame, bool, error) {
	mask, err := expr.TryEvalMask(pred, df.evalContext())
	if err != nil {
		if isNotColumnar(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	out, err := df.Take(mask.Selected())
	return out, true, err
}

// assignColumnar attempts the typed expression path for AssignExpr.
func (df *DataFrame) assignColumnar(name string, e expr.Expr) (*DataFrame, bool, error) {
	ctx := df.evalContext()
	// Predicates assign a bool column (NA results become false, like the
	// row evaluator).
	if pred, ok := e.(expr.Predicate); ok {
		mask, err := expr.TryEvalMask(pred, ctx)
		if err != nil {
			if isNotColumnar(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		out, err := df.Assign(name, series.FromColumn(name, mask.BoolColumn(), df.index))
		return out, true, err
	}
	col, err := expr.TryEvalColumnar(e, ctx)
	if err != nil {
		if isNotColumnar(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	out, err := df.Assign(name, series.FromColumn(name, col, df.index))
	return out, true, err
}

func isNotColumnar(err error) bool {
	return err != nil && errors.Is(err, expr.ErrNotColumnar)
}
