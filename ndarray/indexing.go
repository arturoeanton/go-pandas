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
// 1-D contiguous arrays gather through a typed buffer loop (v0.10.1 —
// previously every element went through the boxed per-slice copier).
func (a *NDArray) Take(indices []int, axis int) (*NDArray, error) {
	if err := a.checkAxis(axis); err != nil {
		return nil, err
	}
	if len(a.shape) == 1 && a.isContiguous() && a.offset == 0 {
		return a.take1DTyped(indices)
	}
	// Contiguous N-D arrays gather typed slabs along the axis
	// (v1.0-rc — previously boxed per slice); views keep the generic
	// copier below.
	if a.isContiguous() && a.offset == 0 {
		return a.takeAxisTyped(indices, axis)
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

// take1DTyped gathers a 1-D contiguous array into one typed output
// buffer with no boxing: one bounds-checked loop per backing type.
// Negative indices wrap once (NumPy convention, like the generic path).
func (a *NDArray) take1DTyped(indices []int) (*NDArray, error) {
	n := a.shape[0]
	resolved := make([]int, len(indices))
	for i, src := range indices {
		if src < 0 {
			src += n
		}
		if src < 0 || src >= n {
			return nil, fmt.Errorf("%w: take index %d out of range for axis 0 with size %d", errs.ErrIndexOutOfBounds, indices[i], n)
		}
		resolved[i] = src
	}
	gather := func() any {
		switch d := a.data.(type) {
		case []bool:
			out := make([]bool, len(resolved))
			for i, p := range resolved {
				out[i] = d[p]
			}
			return out
		case []int:
			out := make([]int, len(resolved))
			for i, p := range resolved {
				out[i] = d[p]
			}
			return out
		case []int64:
			out := make([]int64, len(resolved))
			for i, p := range resolved {
				out[i] = d[p]
			}
			return out
		case []float32:
			out := make([]float32, len(resolved))
			for i, p := range resolved {
				out[i] = d[p]
			}
			return out
		case []float64:
			out := make([]float64, len(resolved))
			for i, p := range resolved {
				out[i] = d[p]
			}
			return out
		case []string:
			out := make([]string, len(resolved))
			for i, p := range resolved {
				out[i] = d[p]
			}
			return out
		}
		return nil
	}
	data := gather()
	if data == nil {
		return nil, fmt.Errorf("%w: take on %s array", errs.ErrTypeMismatch, a.dtype)
	}
	return newDense(data, []int{len(resolved)}, a.dtype), nil
}

// takeAxisTyped gathers a contiguous N-D array along an axis by
// copying inner-stride slabs — one copy() per (outer, index) pair, no
// per-value boxing (v1.0-rc). Negative indices wrap once.
func (a *NDArray) takeAxisTyped(indices []int, axis int) (*NDArray, error) {
	dim := a.shape[axis]
	resolved := make([]int, len(indices))
	for i, src := range indices {
		if src < 0 {
			src += dim
		}
		if src < 0 || src >= dim {
			return nil, fmt.Errorf("%w: take index %d out of range for axis %d with size %d", errs.ErrIndexOutOfBounds, indices[i], axis, dim)
		}
		resolved[i] = src
	}
	outer, inner := 1, 1
	for d := 0; d < axis; d++ {
		outer *= a.shape[d]
	}
	for d := axis + 1; d < len(a.shape); d++ {
		inner *= a.shape[d]
	}
	outShape := a.Shape()
	outShape[axis] = len(resolved)

	var data any
	switch src := a.data.(type) {
	case []bool:
		data = takeAxisSlabs(src, outer, dim, inner, resolved)
	case []int:
		data = takeAxisSlabs(src, outer, dim, inner, resolved)
	case []int64:
		data = takeAxisSlabs(src, outer, dim, inner, resolved)
	case []float32:
		data = takeAxisSlabs(src, outer, dim, inner, resolved)
	case []float64:
		data = takeAxisSlabs(src, outer, dim, inner, resolved)
	case []string:
		data = takeAxisSlabs(src, outer, dim, inner, resolved)
	default:
		return nil, fmt.Errorf("%w: take on %s array", errs.ErrTypeMismatch, a.dtype)
	}
	return newDense(data, outShape, a.dtype), nil
}

func takeAxisSlabs[T any](src []T, outer, dim, inner int, resolved []int) []T {
	out := make([]T, outer*len(resolved)*inner)
	pos := 0
	for o := 0; o < outer; o++ {
		base := o * dim * inner
		for _, idx := range resolved {
			copy(out[pos:pos+inner], src[base+idx*inner:base+(idx+1)*inner])
			pos += inner
		}
	}
	return out
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
