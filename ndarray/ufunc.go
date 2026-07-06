package ndarray

import "math"

// Abs returns |a| elementwise.
func (a *NDArray) Abs() *NDArray { return a.unaryScalar(math.Abs) }

// Sqrt returns the elementwise square root.
func (a *NDArray) Sqrt() *NDArray { return a.unaryScalar(math.Sqrt) }

// Exp returns e**a elementwise.
func (a *NDArray) Exp() *NDArray { return a.unaryScalar(math.Exp) }

// Log returns the natural logarithm elementwise.
func (a *NDArray) Log() *NDArray { return a.unaryScalar(math.Log) }

// Log2 returns the base-2 logarithm elementwise.
func (a *NDArray) Log2() *NDArray { return a.unaryScalar(math.Log2) }

// Log10 returns the base-10 logarithm elementwise.
func (a *NDArray) Log10() *NDArray { return a.unaryScalar(math.Log10) }

// Sin returns the elementwise sine.
func (a *NDArray) Sin() *NDArray { return a.unaryScalar(math.Sin) }

// Cos returns the elementwise cosine.
func (a *NDArray) Cos() *NDArray { return a.unaryScalar(math.Cos) }

// Tan returns the elementwise tangent.
func (a *NDArray) Tan() *NDArray { return a.unaryScalar(math.Tan) }

// Floor rounds each element down.
func (a *NDArray) Floor() *NDArray { return a.unaryScalar(math.Floor) }

// Ceil rounds each element up.
func (a *NDArray) Ceil() *NDArray { return a.unaryScalar(math.Ceil) }

// Round rounds each element half to even (banker's rounding), matching
// np.round.
func (a *NDArray) Round() *NDArray { return a.unaryScalar(math.RoundToEven) }

// Clip limits every element to [min, max].
func (a *NDArray) Clip(min, max float64) *NDArray {
	return a.unaryScalar(func(x float64) float64 {
		if x < min {
			return min
		}
		if x > max {
			return max
		}
		return x
	})
}

// Apply runs an arbitrary elementwise function (ufunc escape hatch).
func (a *NDArray) Apply(f func(x float64) float64) *NDArray {
	return a.unaryScalar(f)
}
