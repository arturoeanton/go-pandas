package expr

import (
	"errors"
	"fmt"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// The columnar expression engine (v0.4) evaluates whole expressions over
// typed column buffers instead of one map[string]any per row. Every
// expression node either executes columnar or reports ErrNotColumnar, in
// which case callers fall back to the row-map evaluator (fallback.go).

// ErrNotColumnar is the sentinel returned when an expression (or the
// data underneath it) cannot take the columnar fast path. It signals
// "use the row fallback", never a user-facing failure.
var ErrNotColumnar = errors.New("expression is not columnar")

// EvalContext gives expressions access to the frame's columns without an
// import cycle: the DataFrame installs a resolver closure.
type EvalContext struct {
	// Len is the number of rows.
	Len int
	// Column resolves a column name to its typed storage.
	Column func(name string) (column.Column, error)
}

// Mask is the columnar result of a predicate. Data[i] reports whether
// row i matched; NA[i] marks rows where the predicate itself is missing
// (an NA operand). Filtering treats NA as not selected, matching the
// documented "comparisons with NA are false" rule.
type Mask struct {
	Data []bool
	NA   []bool
}

func newMask(n int) *Mask {
	return &Mask{Data: make([]bool, n), NA: make([]bool, n)}
}

// CountSelected returns how many rows the mask keeps (NA counts as not
// selected).
func (m *Mask) CountSelected() int {
	n := 0
	for i, keep := range m.Data {
		if keep && !m.NA[i] {
			n++
		}
	}
	return n
}

// Selected returns the row positions kept by the mask (NA rows drop).
// The result is allocated exactly once from a pre-count (v0.4.1).
func (m *Mask) Selected() []int {
	pos := make([]int, 0, m.CountSelected())
	for i, keep := range m.Data {
		if keep && !m.NA[i] {
			pos = append(pos, i)
		}
	}
	return pos
}

// PositionsFromMask is the package-level spelling of Mask.Selected.
func PositionsFromMask(m *Mask) []int { return m.Selected() }

// CountTrueMask is the package-level spelling of Mask.CountSelected.
func CountTrueMask(m *Mask) int { return m.CountSelected() }

// BoolColumn converts the mask into a Bool column. NA predicate results
// become false, matching the row evaluator and pandas' classic bool
// arrays.
func (m *Mask) BoolColumn() column.Column {
	data := make([]bool, len(m.Data))
	for i := range data {
		data[i] = m.Data[i] && !m.NA[i]
	}
	return column.NewBool(data, nil)
}

// colValue is an evaluated operand: either a whole column or a scalar.
type colValue struct {
	col    column.Column
	scalar any
}

func (v colValue) isScalar() bool { return v.col == nil }

// evalValue evaluates an expression into a column or scalar, columnar
// only.
func evalValue(e Expr, ctx *EvalContext) (colValue, error) {
	switch node := e.(type) {
	case ColumnExpr:
		c, err := ctx.Column(node.name)
		if err != nil {
			return colValue{}, err
		}
		return colValue{col: c}, nil
	case LiteralExpr:
		return colValue{scalar: node.Value}, nil
	case binaryExpr:
		return evalBinaryColumnar(node, ctx)
	case funcExpr:
		return evalFuncColumnar(node, ctx)
	case whereExpr:
		return evalWhereColumnar(node, ctx)
	case comparePred, funcPred, logicalPred:
		// A predicate used as a value produces a bool column.
		mask, err := evalMask(e.(Predicate), ctx)
		if err != nil {
			return colValue{}, err
		}
		return colValue{col: mask.BoolColumn()}, nil
	}
	return colValue{}, ErrNotColumnar
}

// evalMask evaluates a predicate into a Mask, columnar only.
func evalMask(p Predicate, ctx *EvalContext) (*Mask, error) {
	switch node := p.(type) {
	case comparePred:
		return evalCompareColumnar(node, ctx)
	case funcPred:
		return evalFuncPredColumnar(node, ctx)
	case logicalPred:
		return evalLogicalColumnar(node, ctx)
	}
	return nil, ErrNotColumnar
}

// TryEvalMask is the entry point used by DataFrame.Where/Query: it
// returns ErrNotColumnar when the row fallback must run.
func TryEvalMask(p Predicate, ctx *EvalContext) (*Mask, error) {
	return evalMask(p, ctx)
}

// TryEvalColumnar is the entry point used by DataFrame.AssignExpr. The
// result column is freshly allocated (safe to attach to a frame).
func TryEvalColumnar(e Expr, ctx *EvalContext) (column.Column, error) {
	v, err := evalValue(e, ctx)
	if err != nil {
		return nil, err
	}
	if v.isScalar() {
		// Broadcast the scalar to a column.
		values := make([]any, ctx.Len)
		for i := range values {
			values[i] = v.scalar
		}
		return column.Infer(values), nil
	}
	// Column results may alias frame storage (bare Col("x")); copy so the
	// assigned column is independent.
	return v.col.Copy(), nil
}

// numericOperand extracts float values+mask from a column operand, or
// reports ErrNotColumnar for non-numeric backings.
func numericOperand(c column.Column) ([]float64, []bool, error) {
	if column.IsObjectBacked(c) {
		return nil, nil, ErrNotColumnar
	}
	vals, mask, ok := c.Float64s()
	if !ok {
		return nil, nil, ErrNotColumnar
	}
	return vals, mask, nil
}

// scalarFloat converts a scalar operand for numeric kernels.
func scalarFloat(v any) (float64, bool) {
	if dtype.IsNA(v) {
		return 0, false
	}
	return dtypeAsFloatNoBool(v)
}

func dtypeAsFloatNoBool(v any) (float64, bool) {
	f, ok := dtype.AsFloat(v)
	return f, ok
}

// typeMismatch builds the same error shape the row evaluator produces.
func typeMismatch(op string, a, b any) error {
	return fmt.Errorf("%w: cannot %s %T with %T", errs.ErrTypeMismatch, op, a, b)
}
