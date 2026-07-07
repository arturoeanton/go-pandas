package ndarray

import (
	"math"

	"github.com/arturoeanton/go-pandas/dtype"
)

func arithFunc(op string) func(x, y float64) float64 {
	switch op {
	case "+":
		return func(x, y float64) float64 { return x + y }
	case "-":
		return func(x, y float64) float64 { return x - y }
	case "*":
		return func(x, y float64) float64 { return x * y }
	case "/":
		return func(x, y float64) float64 { return x / y }
	case "**":
		return math.Pow
	case "%":
		return math.Mod
	}
	return nil
}

// binopAs applies f elementwise over the broadcast of a and b, producing
// an array of the given result dtype.
func binopAs(a, b *NDArray, resultDT dtype.DType, f func(x, y float64) float64) (*NDArray, error) {
	shape, err := BroadcastShapes(a.shape, b.shape)
	if err != nil {
		return nil, err
	}
	la := a.mustFloatLoader("arithmetic")
	lb := b.mustFloatLoader("arithmetic")
	data := allocData(resultDT, shapeSize(shape))
	store := floatStore(data)
	// Fast path: contiguous same-shape float64 operands, dense output.
	if da, ok := a.data.([]float64); ok {
		if db, ok := b.data.([]float64); ok {
			if dout, ok := data.([]float64); ok &&
				!a.view && a.offset == 0 && a.isContiguous() && sameShape(a.shape, shape) &&
				!b.view && b.offset == 0 && b.isContiguous() && sameShape(b.shape, shape) {
				for i := range dout {
					dout[i] = f(da[i], db[i])
				}
				return newDense(data, shape, resultDT), nil
			}
		}
	}
	iter2(a, b, shape, func(pos, offA, offB int) {
		store(pos, f(la(offA), lb(offB)))
	})
	return newDense(data, shape, resultDT), nil
}

// arith runs one promoted arithmetic operation.
func arith(a, b *NDArray, op string) (*NDArray, error) {
	p, err := promoteArith(a.dtype, b.dtype)
	if err != nil {
		return nil, err
	}
	if op == "/" {
		p = divDType(p)
	}
	return binopAs(a, b, p, arithFunc(op))
}

// Add returns a + b with broadcasting and dtype promotion.
func (a *NDArray) Add(b *NDArray) (*NDArray, error) { return arith(a, b, "+") }

// Sub returns a - b with broadcasting and dtype promotion.
func (a *NDArray) Sub(b *NDArray) (*NDArray, error) { return arith(a, b, "-") }

// Mul returns a * b (elementwise) with broadcasting and dtype promotion.
func (a *NDArray) Mul(b *NDArray) (*NDArray, error) { return arith(a, b, "*") }

// Div returns a / b (true division): integer inputs produce Float64,
// float inputs keep their width. Division by zero yields ±Inf/NaN as in
// NumPy.
func (a *NDArray) Div(b *NDArray) (*NDArray, error) { return arith(a, b, "/") }

// Pow returns a ** b with broadcasting; integer inputs keep integer
// dtype (computed in floating point and truncated — see known
// differences for negative exponents).
func (a *NDArray) Pow(b *NDArray) (*NDArray, error) { return arith(a, b, "**") }

// Mod returns the elementwise remainder (math.Mod semantics).
func (a *NDArray) Mod(b *NDArray) (*NDArray, error) { return arith(a, b, "%") }

// scalarResultDType keeps integer dtypes for integral scalars, promotes
// to Float64 otherwise.
func (a *NDArray) scalarResultDType(v float64, closed bool) dtype.DType {
	if closed && v == math.Trunc(v) && (dtype.IsInteger(a.dtype) || a.dtype == dtype.Bool) {
		if a.dtype == dtype.Bool {
			return dtype.Int
		}
		return a.dtype
	}
	if a.dtype == dtype.Float32 && closed {
		return dtype.Float32
	}
	return dtype.Float64
}

// scalarOp applies f elementwise, storing into resultDT.
func (a *NDArray) scalarOp(op string, resultDT dtype.DType, f func(x float64) float64) *NDArray {
	load := a.mustFloatLoader(op)
	data := allocData(resultDT, a.Size())
	store := floatStore(data)
	pos := 0
	a.iter(func(off int) {
		store(pos, f(load(off)))
		pos++
	})
	return newDense(data, a.shape, resultDT)
}

// AddScalar returns a + v (integer arrays stay integer for integral v).
func (a *NDArray) AddScalar(v float64) *NDArray {
	return a.scalarOp("+", a.scalarResultDType(v, true), func(x float64) float64 { return x + v })
}

// SubScalar returns a - v.
func (a *NDArray) SubScalar(v float64) *NDArray {
	return a.scalarOp("-", a.scalarResultDType(v, true), func(x float64) float64 { return x - v })
}

// MulScalar returns a * v.
func (a *NDArray) MulScalar(v float64) *NDArray {
	return a.scalarOp("*", a.scalarResultDType(v, true), func(x float64) float64 { return x * v })
}

// DivScalar returns a / v (always floating point).
func (a *NDArray) DivScalar(v float64) *NDArray {
	return a.scalarOp("/", a.scalarResultDType(v, false), func(x float64) float64 { return x / v })
}

// PowScalar returns a ** v.
func (a *NDArray) PowScalar(v float64) *NDArray {
	return a.scalarOp("**", a.scalarResultDType(v, true), func(x float64) float64 { return math.Pow(x, v) })
}

// Neg returns -a.
func (a *NDArray) Neg() *NDArray {
	dt := a.dtype
	if dt == dtype.Bool {
		dt = dtype.Int
	}
	return a.scalarOp("-", dt, func(x float64) float64 { return -x })
}
