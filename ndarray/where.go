package ndarray

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// Where selects elements from x where cond is true and from y elsewhere,
// like np.where. All three must share the same shape in v0.3. The result
// dtype is the arithmetic promotion of x and y (same dtype in, same
// dtype out; string requires both sides string).
func Where(cond *BoolArray, x, y *NDArray) (*NDArray, error) {
	if !sameShape(cond.shape, x.shape) || !sameShape(cond.shape, y.shape) {
		return nil, fmt.Errorf("%w: where cond %v, x %v, y %v", errs.ErrShapeMismatch, cond.shape, x.Shape(), y.Shape())
	}
	if x.dtype == dtype.String || y.dtype == dtype.String {
		lx, ly := x.stringLoader(), y.stringLoader()
		if lx == nil || ly == nil {
			return nil, fmt.Errorf("%w: where between %s and %s arrays", errs.ErrTypeMismatch, x.dtype, y.dtype)
		}
		out := make([]string, cond.Size())
		fill := func(load func(off int) string, keep func(i int) bool, arr *NDArray) {
			i := 0
			arr.iter(func(off int) {
				if keep(i) {
					out[i] = load(off)
				}
				i++
			})
		}
		fill(lx, func(i int) bool { return cond.data[i] }, x)
		fill(ly, func(i int) bool { return !cond.data[i] }, y)
		return newDense(out, cond.shape, dtype.String), nil
	}
	rdt := dtype.Promote(x.dtype, y.dtype)
	if !dtype.IsNumeric(rdt) && rdt != dtype.Bool {
		return nil, fmt.Errorf("%w: where between %s and %s arrays", errs.ErrTypeMismatch, x.dtype, y.dtype)
	}
	lx := x.mustFloatLoader("where")
	ly := y.mustFloatLoader("where")
	data := allocData(rdt, cond.Size())
	store := floatStore(data)
	xi := 0
	x.iter(func(off int) {
		if cond.data[xi] {
			store(xi, lx(off))
		}
		xi++
	})
	yi := 0
	y.iter(func(off int) {
		if !cond.data[yi] {
			store(yi, ly(off))
		}
		yi++
	})
	return newDense(data, cond.shape, rdt), nil
}

// Compress returns the elements of a (flattened) where mask is true,
// preserving the dtype.
func Compress(mask *BoolArray, a *NDArray) (*NDArray, error) {
	if mask.Size() != a.Size() {
		return nil, fmt.Errorf("%w: mask size %d for array size %d", errs.ErrLengthMismatch, mask.Size(), a.Size())
	}
	var keep []int
	for i, k := range mask.data {
		if k {
			keep = append(keep, i)
		}
	}
	flat := a.Flatten()
	if flat.NDim() != 1 {
		return nil, fmt.Errorf("%w: compress source", errs.ErrShapeMismatch)
	}
	return flat.Take(keep, 0)
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
