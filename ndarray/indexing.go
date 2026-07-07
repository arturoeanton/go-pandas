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

// At returns the element at the given indices as float64. Negative
// indices count from the end, as in NumPy. String arrays return
// ErrTypeMismatch — use ValueAt.
func (a *NDArray) At(indices ...int) (float64, error) {
	off, err := a.offsetOf(indices)
	if err != nil {
		return 0, err
	}
	load := a.floatLoader()
	if load == nil {
		return 0, fmt.Errorf("%w: At on %s array; use ValueAt", errs.ErrTypeMismatch, a.dtype)
	}
	return load(off), nil
}

// ValueAt returns the element at the given indices boxed, for any dtype.
func (a *NDArray) ValueAt(indices ...int) (any, error) {
	off, err := a.offsetOf(indices)
	if err != nil {
		return nil, err
	}
	return a.valueAt(off), nil
}

// MustAt is At that panics on error.
func (a *NDArray) MustAt(indices ...int) float64 {
	v, err := a.At(indices...)
	if err != nil {
		panic(err)
	}
	return v
}

// Set writes an element at the given indices. Writing into an integer
// backing truncates the value (NumPy semantics); string arrays return
// ErrTypeMismatch — use SetValue.
func (a *NDArray) Set(value float64, indices ...int) error {
	off, err := a.offsetOf(indices)
	if err != nil {
		return err
	}
	store := floatStore(a.data)
	if store == nil {
		return fmt.Errorf("%w: Set on %s array; use SetValue", errs.ErrTypeMismatch, a.dtype)
	}
	store(off, value)
	return nil
}

// SetValue writes a boxed element at the given indices, for any dtype.
func (a *NDArray) SetValue(value any, indices ...int) error {
	off, err := a.offsetOf(indices)
	if err != nil {
		return err
	}
	if d, ok := a.data.([]string); ok {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("%w: cannot store %T in string array", errs.ErrTypeMismatch, value)
		}
		d[off] = s
		return nil
	}
	f, ok := toFloat(value)
	if !ok {
		return fmt.Errorf("%w: cannot store %T in %s array", errs.ErrTypeMismatch, value, a.dtype)
	}
	floatStore(a.data)(off, f)
	return nil
}

// Take selects elements along an axis by position, returning a copy.
func (a *NDArray) Take(indices []int, axis int) (*NDArray, error) {
	if err := a.checkAxis(axis); err != nil {
		return nil, err
	}
	outShape := a.Shape()
	outShape[axis] = len(indices)
	out := newDense(allocData(a.dtype, shapeSize(outShape)), outShape, a.dtype)
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
		if err := copyInto(dstView, srcView); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// copyInto copies src's logical elements into dst (equal sizes, same
// dtype family; numeric conversions go through float64).
func copyInto(dst, src *NDArray) error {
	if ds, ok := dst.data.([]string); ok {
		load := src.stringLoader()
		if load == nil {
			return fmt.Errorf("%w: cannot copy %s into string array", errs.ErrTypeMismatch, src.dtype)
		}
		var offs []int
		dst.iter(func(off int) { offs = append(offs, off) })
		i := 0
		src.iter(func(off int) {
			ds[offs[i]] = load(off)
			i++
		})
		return nil
	}
	load := src.floatLoader()
	store := floatStore(dst.data)
	if load == nil || store == nil {
		return fmt.Errorf("%w: cannot copy %s into %s array", errs.ErrTypeMismatch, src.dtype, dst.dtype)
	}
	var offs []int
	dst.iter(func(off int) { offs = append(offs, off) })
	i := 0
	src.iter(func(off int) {
		store(offs[i], load(off))
		i++
	})
	return nil
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	}
	return 0, false
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
