package ndarray

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
)

// Where selects elements from x where cond is true and from y elsewhere,
// like np.where. All three must share the same shape in v0.1.
func Where(cond *BoolArray, x, y *NDArray) (*NDArray, error) {
	if !sameShape(cond.shape, x.shape) || !sameShape(cond.shape, y.shape) {
		return nil, fmt.Errorf("%w: where cond %v, x %v, y %v", errs.ErrShapeMismatch, cond.shape, x.Shape(), y.Shape())
	}
	out := Zeros(cond.shape...)
	dx, dy := x.Data(), y.Data()
	for i := range out.data {
		if cond.data[i] {
			out.data[i] = dx[i]
		} else {
			out.data[i] = dy[i]
		}
	}
	return out, nil
}

// Compress returns the elements of a (flattened) where mask is true.
func Compress(mask *BoolArray, a *NDArray) (*NDArray, error) {
	if mask.Size() != a.Size() {
		return nil, fmt.Errorf("%w: mask size %d for array size %d", errs.ErrLengthMismatch, mask.Size(), a.Size())
	}
	data := a.Data()
	var out []float64
	for i, keep := range mask.data {
		if keep {
			out = append(out, data[i])
		}
	}
	return Array(out), nil
}

func sameShape(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
