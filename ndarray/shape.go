package ndarray

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
)

// Reshape returns an array with the same data and a new shape. One
// dimension may be -1 and is inferred. The result is a view when the array
// is contiguous, otherwise a copy.
func (a *NDArray) Reshape(shape ...int) (*NDArray, error) {
	shape = append([]int(nil), shape...)
	infer := -1
	known := 1
	for d, s := range shape {
		if s == -1 {
			if infer >= 0 {
				return nil, fmt.Errorf("%w: only one dimension can be -1", errs.ErrShapeMismatch)
			}
			infer = d
			continue
		}
		known *= s
	}
	size := a.Size()
	if infer >= 0 {
		if known == 0 || size%known != 0 {
			return nil, fmt.Errorf("%w: cannot infer dimension for size %d into %v", errs.ErrShapeMismatch, size, shape)
		}
		shape[infer] = size / known
		known *= shape[infer]
	}
	if known != size {
		return nil, fmt.Errorf("%w: cannot reshape array of size %d into %v", errs.ErrShapeMismatch, size, shape)
	}
	if a.isContiguous() {
		return &NDArray{
			data:    a.data,
			shape:   shape,
			strides: computeStrides(shape),
			offset:  a.offset,
			dtype:   a.dtype,
			view:    true,
		}, nil
	}
	c := a.Copy()
	c.shape = shape
	c.strides = computeStrides(shape)
	return c, nil
}

// Flatten returns a 1-D copy of the array.
func (a *NDArray) Flatten() *NDArray {
	// Data() may alias the backing buffer for contiguous arrays; copy so
	// the result is independent, as documented.
	return newOwned(append([]float64(nil), a.Data()...), []int{a.Size()})
}

// Ravel returns a 1-D view when the array is contiguous, otherwise a copy.
func (a *NDArray) Ravel() *NDArray {
	if a.isContiguous() {
		return &NDArray{
			data:    a.data,
			shape:   []int{a.Size()},
			strides: []int{1},
			offset:  a.offset,
			dtype:   a.dtype,
			view:    true,
		}
	}
	return a.Flatten()
}

// Transpose permutes the axes (all axes reversed when none are given) and
// returns a view.
func (a *NDArray) Transpose(axes ...int) (*NDArray, error) {
	n := len(a.shape)
	if len(axes) == 0 {
		axes = make([]int, n)
		for i := range axes {
			axes[i] = n - 1 - i
		}
	}
	if len(axes) != n {
		return nil, fmt.Errorf("%w: transpose axes %v for %d dimensions", errs.ErrInvalidAxis, axes, n)
	}
	seen := make([]bool, n)
	shape := make([]int, n)
	strides := make([]int, n)
	for i, ax := range axes {
		if ax < 0 || ax >= n || seen[ax] {
			return nil, fmt.Errorf("%w: invalid transpose axes %v", errs.ErrInvalidAxis, axes)
		}
		seen[ax] = true
		shape[i] = a.shape[ax]
		strides[i] = a.strides[ax]
	}
	return &NDArray{
		data:    a.data,
		shape:   shape,
		strides: strides,
		offset:  a.offset,
		dtype:   a.dtype,
		view:    true,
	}, nil
}

// T reverses the axes (matrix transpose for 2-D arrays).
func (a *NDArray) T() (*NDArray, error) { return a.Transpose() }

// Squeeze removes size-1 dimensions (all of them, or only the given axes).
func (a *NDArray) Squeeze(axis ...int) (*NDArray, error) {
	drop := make(map[int]bool)
	if len(axis) == 0 {
		for d, s := range a.shape {
			if s == 1 {
				drop[d] = true
			}
		}
	} else {
		for _, ax := range axis {
			if err := a.checkAxis(ax); err != nil {
				return nil, err
			}
			if a.shape[ax] != 1 {
				return nil, fmt.Errorf("%w: cannot squeeze axis %d with size %d", errs.ErrShapeMismatch, ax, a.shape[ax])
			}
			drop[ax] = true
		}
	}
	var shape, strides []int
	for d := range a.shape {
		if drop[d] {
			continue
		}
		shape = append(shape, a.shape[d])
		strides = append(strides, a.strides[d])
	}
	return &NDArray{
		data:    a.data,
		shape:   shape,
		strides: strides,
		offset:  a.offset,
		dtype:   a.dtype,
		view:    true,
	}, nil
}

// ExpandDims inserts a size-1 dimension at the given axis.
func (a *NDArray) ExpandDims(axis int) (*NDArray, error) {
	n := len(a.shape)
	if axis < 0 {
		axis += n + 1
	}
	if axis < 0 || axis > n {
		return nil, fmt.Errorf("%w: axis %d for expand_dims on %d dimensions", errs.ErrInvalidAxis, axis, n)
	}
	shape := make([]int, 0, n+1)
	strides := make([]int, 0, n+1)
	for d := 0; d <= n; d++ {
		if d == axis {
			shape = append(shape, 1)
			strides = append(strides, 0)
		}
		if d < n {
			shape = append(shape, a.shape[d])
			strides = append(strides, a.strides[d])
		}
	}
	return &NDArray{
		data:    a.data,
		shape:   shape,
		strides: strides,
		offset:  a.offset,
		dtype:   a.dtype,
		view:    true,
	}, nil
}
