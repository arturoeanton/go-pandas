package ndarray

import (
	"fmt"
	"math"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

func newOwned(data []float64, shape []int) *NDArray {
	return &NDArray{
		data:    data,
		shape:   append([]int(nil), shape...),
		strides: computeStrides(shape),
		dtype:   dtype.Float64,
	}
}

// Array builds a 1-D array from a float64 slice (the slice is copied).
func Array(data []float64) *NDArray {
	return newOwned(append([]float64(nil), data...), []int{len(data)})
}

// ArrayOf builds a 1-D array from any numeric slice.
func ArrayOf[T Number](data []T) *NDArray {
	out := make([]float64, len(data))
	for i, v := range data {
		out[i] = float64(v)
	}
	return newOwned(out, []int{len(data)})
}

// Array2D builds a 2-D array from a slice of rows.
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

// FromSlice builds an array with an explicit shape from flat data.
func FromSlice(data []float64, shape ...int) (*NDArray, error) {
	if len(shape) == 0 {
		shape = []int{len(data)}
	}
	if shapeSize(shape) != len(data) {
		return nil, fmt.Errorf("%w: cannot shape %d elements into %v", errs.ErrShapeMismatch, len(data), shape)
	}
	return newOwned(append([]float64(nil), data...), shape), nil
}

// MustFromSlice is FromSlice that panics on shape mismatch; convenient in
// examples and tests.
func MustFromSlice(data []float64, shape ...int) *NDArray {
	a, err := FromSlice(data, shape...)
	if err != nil {
		panic(err)
	}
	return a
}

// Zeros returns an array of zeros with the given shape.
func Zeros(shape ...int) *NDArray {
	if len(shape) == 0 {
		shape = []int{}
	}
	return newOwned(make([]float64, shapeSize(shape)), shape)
}

// Ones returns an array of ones with the given shape.
func Ones(shape ...int) *NDArray { return Full(1, shape...) }

// Full returns an array filled with a constant value.
func Full(value float64, shape ...int) *NDArray {
	data := make([]float64, shapeSize(shape))
	for i := range data {
		data[i] = value
	}
	return newOwned(data, shape)
}

// Empty returns an uninitialized (zeroed in Go) array.
func Empty(shape ...int) *NDArray { return Zeros(shape...) }

// Arange mirrors np.arange: Arange(stop), Arange(start, stop) or
// Arange(start, stop, step).
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
	for i, v := range lin.data {
		lin.data[i] = math.Pow(10, v)
	}
	return lin
}

// Eye returns the n x n identity matrix.
func Eye(n int) *NDArray {
	a := Zeros(n, n)
	for i := 0; i < n; i++ {
		a.data[i*n+i] = 1
	}
	return a
}

// Identity is an alias of Eye.
func Identity(n int) *NDArray { return Eye(n) }

// Diag builds a square matrix with v on the diagonal (v must be 1-D).
func Diag(v *NDArray) (*NDArray, error) {
	if v.NDim() != 1 {
		return nil, fmt.Errorf("%w: Diag expects a 1-D array", errs.ErrShapeMismatch)
	}
	n := v.shape[0]
	out := Zeros(n, n)
	src := v.Data()
	for i := 0; i < n; i++ {
		out.data[i*n+i] = src[i]
	}
	return out, nil
}
