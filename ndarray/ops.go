package ndarray

import "math"

// binop applies f elementwise over the broadcast of a and b.
func binop(a, b *NDArray, f func(x, y float64) float64) (*NDArray, error) {
	shape, err := BroadcastShapes(a.shape, b.shape)
	if err != nil {
		return nil, err
	}
	out := Zeros(shape...)
	iter2(a, b, shape, func(pos, offA, offB int) {
		out.data[pos] = f(a.data[offA], b.data[offB])
	})
	return out, nil
}

// Add returns a + b with broadcasting.
func (a *NDArray) Add(b *NDArray) (*NDArray, error) {
	return binop(a, b, func(x, y float64) float64 { return x + y })
}

// Sub returns a - b with broadcasting.
func (a *NDArray) Sub(b *NDArray) (*NDArray, error) {
	return binop(a, b, func(x, y float64) float64 { return x - y })
}

// Mul returns a * b (elementwise) with broadcasting.
func (a *NDArray) Mul(b *NDArray) (*NDArray, error) {
	return binop(a, b, func(x, y float64) float64 { return x * y })
}

// Div returns a / b with broadcasting. Division by zero yields ±Inf/NaN as
// in NumPy.
func (a *NDArray) Div(b *NDArray) (*NDArray, error) {
	return binop(a, b, func(x, y float64) float64 { return x / y })
}

// Pow returns a ** b with broadcasting.
func (a *NDArray) Pow(b *NDArray) (*NDArray, error) {
	return binop(a, b, math.Pow)
}

// Mod returns the elementwise remainder (math.Mod semantics).
func (a *NDArray) Mod(b *NDArray) (*NDArray, error) {
	return binop(a, b, math.Mod)
}

// unaryScalar applies f to every element into a fresh array.
func (a *NDArray) unaryScalar(f func(x float64) float64) *NDArray {
	out := Zeros(a.shape...)
	i := 0
	a.iter(func(off int) {
		out.data[i] = f(a.data[off])
		i++
	})
	return out
}

// AddScalar returns a + v.
func (a *NDArray) AddScalar(v float64) *NDArray {
	return a.unaryScalar(func(x float64) float64 { return x + v })
}

// SubScalar returns a - v.
func (a *NDArray) SubScalar(v float64) *NDArray {
	return a.unaryScalar(func(x float64) float64 { return x - v })
}

// MulScalar returns a * v.
func (a *NDArray) MulScalar(v float64) *NDArray {
	return a.unaryScalar(func(x float64) float64 { return x * v })
}

// DivScalar returns a / v.
func (a *NDArray) DivScalar(v float64) *NDArray {
	return a.unaryScalar(func(x float64) float64 { return x / v })
}

// PowScalar returns a ** v.
func (a *NDArray) PowScalar(v float64) *NDArray {
	return a.unaryScalar(func(x float64) float64 { return math.Pow(x, v) })
}

// Neg returns -a.
func (a *NDArray) Neg() *NDArray {
	return a.unaryScalar(func(x float64) float64 { return -x })
}
