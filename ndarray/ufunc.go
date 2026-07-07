package ndarray

import (
	"math"

	"github.com/arturoeanton/go-pandas/dtype"
)

// unaryFloat applies f elementwise producing Float64 (np.sqrt on ints is
// float64). Panics with ErrTypeMismatch on string arrays — numeric
// ufuncs have no error return, matching the NumPy call shape.
func (a *NDArray) unaryFloat(op string, f func(x float64) float64) *NDArray {
	dt := dtype.Float64
	if a.dtype == dtype.Float32 {
		dt = dtype.Float32
	}
	return a.scalarOp(op, dt, f)
}

// unaryPreserve applies f keeping the input dtype (np.abs on ints is
// int).
func (a *NDArray) unaryPreserve(op string, f func(x float64) float64) *NDArray {
	dt := a.dtype
	if dt == dtype.Bool {
		dt = dtype.Int
	}
	return a.scalarOp(op, dt, f)
}

// Abs returns |a| elementwise, preserving the dtype.
func (a *NDArray) Abs() *NDArray { return a.unaryPreserve("abs", math.Abs) }

// Sqrt returns the elementwise square root (floating point result).
func (a *NDArray) Sqrt() *NDArray { return a.unaryFloat("sqrt", math.Sqrt) }

// Exp returns e**a elementwise.
func (a *NDArray) Exp() *NDArray { return a.unaryFloat("exp", math.Exp) }

// Log returns the natural logarithm elementwise.
func (a *NDArray) Log() *NDArray { return a.unaryFloat("log", math.Log) }

// Log2 returns the base-2 logarithm elementwise.
func (a *NDArray) Log2() *NDArray { return a.unaryFloat("log2", math.Log2) }

// Log10 returns the base-10 logarithm elementwise.
func (a *NDArray) Log10() *NDArray { return a.unaryFloat("log10", math.Log10) }

// Sin returns the elementwise sine.
func (a *NDArray) Sin() *NDArray { return a.unaryFloat("sin", math.Sin) }

// Cos returns the elementwise cosine.
func (a *NDArray) Cos() *NDArray { return a.unaryFloat("cos", math.Cos) }

// Tan returns the elementwise tangent.
func (a *NDArray) Tan() *NDArray { return a.unaryFloat("tan", math.Tan) }

// Floor rounds each element down, preserving the dtype.
func (a *NDArray) Floor() *NDArray { return a.unaryPreserve("floor", math.Floor) }

// Ceil rounds each element up, preserving the dtype.
func (a *NDArray) Ceil() *NDArray { return a.unaryPreserve("ceil", math.Ceil) }

// Round rounds each element half to even (banker's rounding, matching
// np.round), preserving the dtype.
func (a *NDArray) Round() *NDArray { return a.unaryPreserve("round", math.RoundToEven) }

// Clip limits every element to [min, max], preserving the dtype.
func (a *NDArray) Clip(min, max float64) *NDArray {
	return a.unaryPreserve("clip", func(x float64) float64 {
		if x < min {
			return min
		}
		if x > max {
			return max
		}
		return x
	})
}

// Apply runs an arbitrary elementwise float function (ufunc escape
// hatch); the result is floating point.
func (a *NDArray) Apply(f func(x float64) float64) *NDArray {
	return a.unaryFloat("apply", f)
}
