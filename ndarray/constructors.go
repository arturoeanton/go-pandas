package ndarray

import (
	"fmt"
	"math"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

func newOwned(data []float64, shape []int) *NDArray {
	return newDense(data, shape, dtype.Float64)
}

// Array builds a 1-D Float64 array from a float64 slice (copied).
func Array(data []float64) *NDArray {
	return newOwned(append([]float64(nil), data...), []int{len(data)})
}

// typedSlice copies a supported Go slice into a backing plus dtype.
func typedSlice(values any) (any, dtype.DType, int, bool) {
	switch d := values.(type) {
	case []bool:
		return append([]bool(nil), d...), dtype.Bool, len(d), true
	case []int:
		return append([]int(nil), d...), dtype.Int, len(d), true
	case []int64:
		return append([]int64(nil), d...), dtype.Int64, len(d), true
	case []float32:
		return append([]float32(nil), d...), dtype.Float32, len(d), true
	case []float64:
		return append([]float64(nil), d...), dtype.Float64, len(d), true
	case []string:
		return append([]string(nil), d...), dtype.String, len(d), true
	}
	return nil, dtype.Invalid, 0, false
}

// ArrayOf builds a 1-D array from any supported element slice; the
// backing storage keeps the element type (v0.3). Unsupported numeric
// widths (int8, uint16, ...) convert to float64.
func ArrayOf[T Element](data []T) *NDArray {
	if backing, dt, n, ok := typedSlice(any(data)); ok {
		return newDense(backing, []int{n}, dt)
	}
	out := make([]float64, len(data))
	for i, v := range data {
		f, _ := dtype.AsFloat(v)
		out[i] = f
	}
	return newOwned(out, []int{len(data)})
}

// Array2D builds a 2-D Float64 array from a slice of rows.
func Array2D(data [][]float64) (*NDArray, error) {
	rows := len(data)
	if rows == 0 {
		return newOwned(nil, []int{0, 0}), nil
	}
	cols := len(data[0])
	flat := make([]float64, 0, rows*cols)
	for _, row := range data {
		if len(row) != cols {
			return nil, fmt.Errorf("%w: ragged rows in Array2D", errs.ErrShapeMismatch)
		}
		flat = append(flat, row...)
	}
	return newOwned(flat, []int{rows, cols}), nil
}

// FromSlice builds an array with an explicit shape from flat typed data.
func FromSlice[T Element](data []T, shape ...int) (*NDArray, error) {
	if len(shape) == 0 {
		shape = []int{len(data)}
	}
	if shapeSize(shape) != len(data) {
		return nil, fmt.Errorf("%w: cannot shape %d elements into %v", errs.ErrShapeMismatch, len(data), shape)
	}
	flat := ArrayOf(data)
	flat.shape = append([]int(nil), shape...)
	flat.strides = computeStrides(shape)
	return flat, nil
}

// MustFromSlice is FromSlice that panics on shape mismatch; convenient in
// examples and tests.
func MustFromSlice[T Element](data []T, shape ...int) *NDArray {
	a, err := FromSlice(data, shape...)
	if err != nil {
		panic(err)
	}
	return a
}

// Zeros returns a Float64 array of zeros with the given shape.
func Zeros(shape ...int) *NDArray {
	if len(shape) == 0 {
		shape = []int{}
	}
	return newOwned(make([]float64, shapeSize(shape)), shape)
}

// Ones returns a Float64 array of ones with the given shape.
func Ones(shape ...int) *NDArray { return Full(1, shape...) }

// Full returns a Float64 array filled with a constant value.
func Full(value float64, shape ...int) *NDArray {
	data := make([]float64, shapeSize(shape))
	for i := range data {
		data[i] = value
	}
	return newOwned(data, shape)
}

// Empty returns an uninitialized (zeroed in Go) Float64 array.
func Empty(shape ...int) *NDArray { return Zeros(shape...) }

// Arange mirrors np.arange: Arange(stop), Arange(start, stop) or
// Arange(start, stop, step). The result dtype is Float64 (a documented
// difference: NumPy returns int64 for integer arguments).
func Arange(args ...float64) *NDArray {
	var start, stop, step float64
	switch len(args) {
	case 1:
		start, stop, step = 0, args[0], 1
	case 2:
		start, stop, step = args[0], args[1], 1
	case 3:
		start, stop, step = args[0], args[1], args[2]
	default:
		return newOwned(nil, []int{0})
	}
	if step == 0 {
		return newOwned(nil, []int{0})
	}
	n := int(math.Ceil((stop - start) / step))
	if n < 0 {
		n = 0
	}
	data := make([]float64, n)
	for i := range data {
		data[i] = start + float64(i)*step
	}
	return newOwned(data, []int{n})
}

// Linspace returns num evenly spaced samples over [start, stop].
func Linspace(start, stop float64, num int) *NDArray {
	if num <= 0 {
		return newOwned(nil, []int{0})
	}
	data := make([]float64, num)
	if num == 1 {
		data[0] = start
	} else {
		step := (stop - start) / float64(num-1)
		for i := range data {
			data[i] = start + float64(i)*step
		}
		data[num-1] = stop
	}
	return newOwned(data, []int{num})
}

// Logspace returns num samples spaced evenly on a log10 scale.
func Logspace(start, stop float64, num int) *NDArray {
	lin := Linspace(start, stop, num)
	d := lin.data.([]float64)
	for i, v := range d {
		d[i] = math.Pow(10, v)
	}
	return lin
}

// Eye returns the n x n identity matrix (Float64).
func Eye(n int) *NDArray {
	a := Zeros(n, n)
	d := a.data.([]float64)
	for i := 0; i < n; i++ {
		d[i*n+i] = 1
	}
	return a
}

// Identity is an alias of Eye.
func Identity(n int) *NDArray { return Eye(n) }

// Diag builds a square matrix with v on the diagonal (v must be 1-D
// numeric).
func Diag(v *NDArray) (*NDArray, error) {
	if v.NDim() != 1 {
		return nil, fmt.Errorf("%w: Diag expects a 1-D array", errs.ErrShapeMismatch)
	}
	load := v.floatLoader()
	if load == nil {
		return nil, fmt.Errorf("%w: Diag on %s array", errs.ErrTypeMismatch, v.dtype)
	}
	n := v.shape[0]
	out := Zeros(n, n)
	d := out.data.([]float64)
	i := 0
	v.iter(func(off int) {
		d[i*n+i] = load(off)
		i++
	})
	return out, nil
}
