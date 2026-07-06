package series

import (
	"fmt"
	"math"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// binop applies an arithmetic operation elementwise between two series of
// equal length. Missing operands produce missing results. Integer inputs
// keep integer results for closed operations; Div always yields floats.
func (s *Series) binop(other *Series, op string) (*Series, error) {
	if other.Len() != s.Len() {
		return nil, fmt.Errorf("%w: series lengths %d and %d", errs.ErrLengthMismatch, s.Len(), other.Len())
	}
	data := make([]any, s.Len())
	mask := make([]bool, s.Len())
	intResult := dtype.IsInteger(s.dtype) && dtype.IsInteger(other.dtype) && op != "/" && op != "**"
	for i := 0; i < s.Len(); i++ {
		if s.mask[i] || other.mask[i] {
			mask[i] = true
			continue
		}
		x, okX := dtype.AsFloat(s.data[i])
		y, okY := dtype.AsFloat(other.data[i])
		if !okX || !okY {
			// String concatenation via Add.
			if op == "+" {
				if xs, ok := s.data[i].(string); ok {
					if ys, ok := other.data[i].(string); ok {
						data[i] = xs + ys
						continue
					}
				}
			}
			return nil, fmt.Errorf("%w: %s between %T and %T", errs.ErrInvalidOperation, op, s.data[i], other.data[i])
		}
		var r float64
		switch op {
		case "+":
			r = x + y
		case "-":
			r = x - y
		case "*":
			r = x * y
		case "/":
			r = x / y
		case "%":
			r = math.Mod(x, y)
		case "**":
			r = math.Pow(x, y)
		}
		if intResult {
			data[i] = int64(r)
		} else {
			data[i] = r
		}
	}
	dt := dtype.Float64
	if intResult {
		dt = dtype.Int64
	}
	if dtype.IsString(s.dtype) && op == "+" {
		dt = dtype.String
	}
	return &Series{name: s.name, dtype: dt, data: data, mask: mask, index: s.index.Clone()}, nil
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
	c := s.Copy()
	for i := range c.data {
		if c.mask[i] {
			continue
		}
		v := fn(c.data[i])
		if dtype.IsNA(v) {
			c.data[i] = nil
			c.mask[i] = true
			continue
		}
		c.data[i] = v
	}
	c.dtype = dtype.InferDType(c.Values())
	return c
}
