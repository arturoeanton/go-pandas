package ndarray

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
)

// BroadcastShapes computes the NumPy broadcast shape of two shapes, or
// returns ErrBroadcastMismatch. Shapes are compared from the trailing
// dimension; sizes are compatible when equal or when either is 1; missing
// leading dimensions are treated as 1.
func BroadcastShapes(s1, s2 []int) ([]int, error) {
	n := len(s1)
	if len(s2) > n {
		n = len(s2)
	}
	out := make([]int, n)
	for i := 1; i <= n; i++ {
		d1, d2 := 1, 1
		if i <= len(s1) {
			d1 = s1[len(s1)-i]
		}
		if i <= len(s2) {
			d2 = s2[len(s2)-i]
		}
		switch {
		case d1 == d2:
			out[n-i] = d1
		case d1 == 1:
			out[n-i] = d2
		case d2 == 1:
			out[n-i] = d1
		default:
			return nil, fmt.Errorf("%w: operands could not be broadcast together with shapes %v %v", errs.ErrBroadcastMismatch, s1, s2)
		}
	}
	return out, nil
}

// broadcastStrides returns the strides to iterate a as if it had the
// target shape: broadcast dimensions get stride 0, so no data is copied.
func (a *NDArray) broadcastStrides(target []int) []int {
	out := make([]int, len(target))
	offset := len(target) - len(a.shape)
	for i := range target {
		if i < offset {
			out[i] = 0
			continue
		}
		d := i - offset
		if a.shape[d] == 1 && target[i] != 1 {
			out[i] = 0
		} else {
			out[i] = a.strides[d]
		}
	}
	return out
}

// BroadcastTo returns a read-only view of a with the target shape.
func (a *NDArray) BroadcastTo(shape ...int) (*NDArray, error) {
	if _, err := BroadcastShapes(a.shape, shape); err != nil {
		return nil, err
	}
	bs, err := BroadcastShapes(a.shape, shape)
	if err != nil {
		return nil, err
	}
	if shapeSize(bs) != shapeSize(shape) || len(bs) != len(shape) {
		return nil, fmt.Errorf("%w: cannot broadcast %v to %v", errs.ErrBroadcastMismatch, a.shape, shape)
	}
	for i := range bs {
		if bs[i] != shape[i] {
			return nil, fmt.Errorf("%w: cannot broadcast %v to %v", errs.ErrBroadcastMismatch, a.shape, shape)
		}
	}
	return &NDArray{
		data:    a.data,
		shape:   append([]int(nil), shape...),
		strides: a.broadcastStrides(shape),
		offset:  a.offset,
		dtype:   a.dtype,
		view:    true,
	}, nil
}

// iter2 walks two arrays in lockstep over their broadcast shape, calling f
// with the physical offsets of each element pair and the linear output
// position.
func iter2(a, b *NDArray, shape []int, f func(pos, offA, offB int)) {
	sa := a.broadcastStrides(shape)
	sb := b.broadcastStrides(shape)
	size := shapeSize(shape)
	if size == 0 {
		return
	}
	if len(shape) == 0 {
		f(0, a.offset, b.offset)
		return
	}
	coords := make([]int, len(shape))
	offA, offB := a.offset, b.offset
	for pos := 0; ; pos++ {
		f(pos, offA, offB)
		d := len(coords) - 1
		for d >= 0 {
			coords[d]++
			offA += sa[d]
			offB += sb[d]
			if coords[d] < shape[d] {
				break
			}
			offA -= coords[d] * sa[d]
			offB -= coords[d] * sb[d]
			coords[d] = 0
			d--
		}
		if d < 0 {
			return
		}
	}
}
