package series

import (
	"fmt"
	"math"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/internal/column"
)

func applyOp(op string, x, y float64) float64 {
	switch op {
	case "+":
		return x + y
	case "-":
		return x - y
	case "*":
		return x * y
	case "/":
		return x / y
	case "%":
		return math.Mod(x, y)
	case "**":
		return math.Pow(x, y)
	}
	return math.NaN()
}

// binop applies an arithmetic operation elementwise between two series of
// equal length. Missing operands produce missing results. Integer inputs
// keep integer results (typed Int64 storage) for closed operations; Div
// always yields floats.
func (s *Series) binop(other *Series, op string) (*Series, error) {
	if other.Len() != s.Len() {
		return nil, fmt.Errorf("%w: series lengths %d and %d", errs.ErrLengthMismatch, s.Len(), other.Len())
	}
	intResult := dtype.IsInteger(s.DType()) && dtype.IsInteger(other.DType()) && op != "/" && op != "**"

	// Fast path: both sides expose numeric buffers.
	fa, ma, okA := s.col.Float64s()
	fb, mb, okB := other.col.Float64s()
	if okA && okB {
		n := s.Len()
		mask := make([]bool, n)
		if intResult {
			data := make([]int64, n)
			for i := 0; i < n; i++ {
				if ma[i] || mb[i] {
					mask[i] = true
					continue
				}
				data[i] = int64(applyOp(op, fa[i], fb[i]))
			}
			return fromColumn(s.name, column.NewInt64(data, mask), s.index.Clone()), nil
		}
		data := make([]float64, n)
		for i := 0; i < n; i++ {
			if ma[i] || mb[i] {
				mask[i] = true
				continue
			}
			data[i] = applyOp(op, fa[i], fb[i])
		}
		return fromColumn(s.name, column.NewFloat64(data, mask), s.index.Clone()), nil
	}

	// String concatenation via Add.
	if op == "+" && dtype.IsString(s.DType()) && dtype.IsString(other.DType()) {
		n := s.Len()
		data := make([]string, n)
		mask := make([]bool, n)
		for i := 0; i < n; i++ {
			if s.col.IsNA(i) || other.col.IsNA(i) {
				mask[i] = true
				continue
			}
			xs, okX := s.col.Value(i).(string)
			ys, okY := other.col.Value(i).(string)
			if !okX || !okY {
				return nil, fmt.Errorf("%w: + between %T and %T", errs.ErrInvalidOperation, s.col.Value(i), other.col.Value(i))
			}
			data[i] = xs + ys
		}
		return fromColumn(s.name, column.NewString(data, mask), s.index.Clone()), nil
	}

	// Generic fallback for object-backed numeric data.
	n := s.Len()
	data := make([]float64, n)
	mask := make([]bool, n)
	for i := 0; i < n; i++ {
		if s.col.IsNA(i) || other.col.IsNA(i) {
			mask[i] = true
			continue
		}
		x, okX := dtype.AsFloat(s.col.Value(i))
		y, okY := dtype.AsFloat(other.col.Value(i))
		if !okX || !okY {
			return nil, fmt.Errorf("%w: %s between %T and %T", errs.ErrInvalidOperation, op, s.col.Value(i), other.col.Value(i))
		}
		data[i] = applyOp(op, x, y)
	}
	return fromColumn(s.name, column.NewFloat64(data, mask), s.index.Clone()), nil
}

// Add returns s + other elementwise.
func (s *Series) Add(other *Series) (*Series, error) { return s.binop(other, "+") }

// Sub returns s - other elementwise.
func (s *Series) Sub(other *Series) (*Series, error) { return s.binop(other, "-") }

// Mul returns s * other elementwise.
func (s *Series) Mul(other *Series) (*Series, error) { return s.binop(other, "*") }

// Div returns s / other elementwise (always float).
func (s *Series) Div(other *Series) (*Series, error) { return s.binop(other, "/") }

// Mod returns the elementwise remainder.
func (s *Series) Mod(other *Series) (*Series, error) { return s.binop(other, "%") }

// Pow returns s ** other elementwise.
func (s *Series) Pow(other *Series) (*Series, error) { return s.binop(other, "**") }

// scalarSeries builds a constant series matching s for scalar operations.
func (s *Series) scalarSeries(v any) *Series {
	values := make([]any, s.Len())
	for i := range values {
		values[i] = v
	}
	return NewSeries(s.name, values, WithIndex(s.index))
}

// AddScalar returns s + v.
func (s *Series) AddScalar(v any) (*Series, error) { return s.binop(s.scalarSeries(v), "+") }

// SubScalar returns s - v.
func (s *Series) SubScalar(v any) (*Series, error) { return s.binop(s.scalarSeries(v), "-") }

// MulScalar returns s * v.
func (s *Series) MulScalar(v any) (*Series, error) { return s.binop(s.scalarSeries(v), "*") }

// DivScalar returns s / v.
func (s *Series) DivScalar(v any) (*Series, error) { return s.binop(s.scalarSeries(v), "/") }

// ModScalar returns s % v.
func (s *Series) ModScalar(v any) (*Series, error) { return s.binop(s.scalarSeries(v), "%") }

// PowScalar returns s ** v.
func (s *Series) PowScalar(v any) (*Series, error) { return s.binop(s.scalarSeries(v), "**") }

// Apply maps a function over the values (missing entries stay missing).
func (s *Series) Apply(fn func(v any) any) *Series {
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.col.IsNA(i) {
			continue
		}
		v := fn(s.col.Value(i))
		if dtype.IsNA(v) {
			continue
		}
		values[i] = v
	}
	return fromColumn(s.name, column.Infer(values), s.index.Clone())
}
