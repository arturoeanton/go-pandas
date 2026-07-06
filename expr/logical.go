package expr

import "strings"

// logicalPred combines predicates with and/or/not.
type logicalPred struct {
	op    string // "and", "or", "not"
	preds []Predicate
}

func (p logicalPred) Eval(row map[string]any) (any, error) { return p.EvalBool(row) }

func (p logicalPred) EvalBool(row map[string]any) (bool, error) {
	switch p.op {
	case "and":
		for _, pr := range p.preds {
			ok, err := pr.EvalBool(row)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	case "or":
		for _, pr := range p.preds {
			ok, err := pr.EvalBool(row)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	default: // not
		ok, err := p.preds[0].EvalBool(row)
		if err != nil {
			return false, err
		}
		return !ok, nil
	}
}

func (p logicalPred) String() string {
	if p.op == "not" {
		return "not " + p.preds[0].String()
	}
	parts := make([]string, len(p.preds))
	for i, pr := range p.preds {
		parts[i] = pr.String()
	}
	return "(" + strings.Join(parts, " "+p.op+" ") + ")"
}

// And is true when every predicate is true.
func And(preds ...Predicate) Predicate { return logicalPred{op: "and", preds: preds} }

// Or is true when at least one predicate is true.
func Or(preds ...Predicate) Predicate { return logicalPred{op: "or", preds: preds} }

// Not negates a predicate.
func Not(pred Predicate) Predicate { return logicalPred{op: "not", preds: []Predicate{pred}} }

// whereExpr implements Where(cond, x, y): x when cond else y.
type whereExpr struct {
	cond Predicate
	x, y Expr
}

func (w whereExpr) Eval(row map[string]any) (any, error) {
	ok, err := w.cond.EvalBool(row)
	if err != nil {
		return nil, err
	}
	if ok {
		return w.x.Eval(row)
	}
	return w.y.Eval(row)
}

func (w whereExpr) String() string {
	return "where(" + w.cond.String() + ", " + w.x.String() + ", " + w.y.String() + ")"
}

// Where builds a conditional expression: x when cond is true, otherwise y.
func Where(cond Predicate, x any, y any) Expr {
	return whereExpr{cond: cond, x: asExpr(x), y: asExpr(y)}
}
