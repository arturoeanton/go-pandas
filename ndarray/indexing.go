package ndarray

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
)

// offsetOf validates full indices and returns the physical offset.
func (a *NDArray) offsetOf(indices []int) (int, error) {
	if len(indices) != len(a.shape) {
		return 0, fmt.Errorf("%w: got %d indices for %d dimensions", errs.ErrIndexOutOfBounds, len(indices), len(a.shape))
	}
	off := a.offset
	for d, i := range indices {
		if i < 0 {
			i += a.shape[d]
		}
		if i < 0 || i >= a.shape[d] {
			return 0, fmt.Errorf("%w: index %d out of range for axis %d with size %d", errs.ErrIndexOutOfBounds, indices[d], d, a.shape[d])
		}
		off += i * a.strides[d]
	}
	return off, nil
}

// At returns the element at the given indices. Negative indices count from
// the end, as in NumPy.
func (a *NDArray) At(indices ...int) (float64, error) {
	off, err := a.offsetOf(indices)
	if err != nil {
		return 0, err
	}
	return a.data[off], nil
}

// MustAt is At that panics on error.
func (a *NDArray) MustAt(indices ...int) float64 {
	v, err := a.At(indices...)
	if err != nil {
		panic(err)
	}
	return v
}

// Set writes an element at the given indices.
func (a *NDArray) Set(value float64, indices ...int) error {
	off, err := a.offsetOf(indices)
	if err != nil {
		return err
	}
	a.data[off] = value
	return nil
}

// Take selects elements along an axis by position, returning a copy.
func (a *NDArray) Take(indices []int, axis int) (*NDArray, error) {
	if err := a.checkAxis(axis); err != nil {
		return nil, err
	}
	outShape := a.Shape()
	outShape[axis] = len(indices)
	out := Zeros(outShape...)
	dim := a.shape[axis]
	for j, src := range indices {
		if src < 0 {
			src += dim
		}
		if src < 0 || src >= dim {
			return nil, fmt.Errorf("%w: take index %d out of range for axis %d with size %d", errs.ErrIndexOutOfBounds, indices[j], axis, dim)
		}
		srcView, err := a.axisSlice(axis, src)
		if err != nil {
			return nil, err
		}
		dstView, err := out.axisSlice(axis, j)
		if err != nil {
			return nil, err
		}
		srcData := srcView.Data()
		i := 0
		dstView.iter(func(off int) {
			dstView.data[off] = srcData[i]
			i++
		})
	}
	return out, nil
}

// axisSlice returns the view selecting a single position along an axis
// (keeping the axis with size 1 removed).
func (a *NDArray) axisSlice(axis, pos int) (*NDArray, error) {
	if err := a.checkAxis(axis); err != nil {
		return nil, err
	}
	shape := make([]int, 0, len(a.shape)-1)
	strides := make([]int, 0, len(a.shape)-1)
	for d := range a.shape {
		if d == axis {
			continue
		}
		shape = append(shape, a.shape[d])
		strides = append(strides, a.strides[d])
	}
	return &NDArray{
		data:    a.data,
		shape:   shape,
		strides: strides,
		offset:  a.offset + pos*a.strides[axis],
		dtype:   a.dtype,
		view:    true,
	}, nil
}
