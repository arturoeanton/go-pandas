package ndarray

import (
	"fmt"
	"math"
	"sort"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// Typed constructors — since v0.3 these store real typed slices.

// ArrayInt builds a 1-D array backed by []int.
func ArrayInt(data []int) *NDArray { return ArrayOf(data) }

// ArrayInt64 builds a 1-D array backed by []int64.
func ArrayInt64(data []int64) *NDArray { return ArrayOf(data) }

// ArrayFloat32 builds a 1-D array backed by []float32.
func ArrayFloat32(data []float32) *NDArray { return ArrayOf(data) }

// ArrayFloat64 builds a 1-D array backed by []float64.
func ArrayFloat64(data []float64) *NDArray { return Array(data) }

// ArrayBool builds a 1-D array backed by []bool.
func ArrayBool(data []bool) *NDArray { return ArrayOf(data) }

// ArrayString builds a 1-D array backed by []string. String arrays
// support comparisons, Sort, Unique, Take, views and Astype to numeric
// via parsing; arithmetic and numeric ufuncs return errors (or panic for
// the error-free ufunc methods).
func ArrayString(data []string) *NDArray { return ArrayOf(data) }

// Astype converts the array to a new dtype, changing the real backing
// storage (v0.3). Float to integer truncates toward zero; bool targets
// store v != 0; string sources parse numerics; numeric to string
// formats. Invalid conversions return errors.
func (a *NDArray) Astype(dt dtype.DType) (*NDArray, error) {
	switch dt {
	case dtype.Bool, dtype.Int, dtype.Int64, dtype.Float32, dtype.Float64, dtype.String:
	default:
		return nil, fmt.Errorf("%w: NDArray.Astype to %s", errs.ErrInvalidDType, dt)
	}
	data := allocData(dt, a.Size())
	if dt == dtype.String {
		out := data.([]string)
		i := 0
		var castErr error
		a.iter(func(off int) {
			if castErr != nil {
				return
			}
			c, err := dtype.CastValue(a.valueAt(off), dtype.String)
			if err != nil {
				castErr = err
				return
			}
			out[i] = c.(string)
			i++
		})
		if castErr != nil {
			return nil, castErr
		}
		return newDense(data, a.shape, dt), nil
	}
	store := floatStore(data)
	if load := a.floatLoader(); load != nil {
		// numeric -> numeric: no boxing
		i := 0
		a.iter(func(off int) {
			store(i, load(off))
			i++
		})
		return newDense(data, a.shape, dt), nil
	}
	// string -> numeric: parse each element
	loadStr := a.stringLoader()
	i := 0
	var castErr error
	a.iter(func(off int) {
		if castErr != nil {
			return
		}
		c, err := dtype.CastValue(loadStr(off), dtype.Float64)
		if err != nil {
			castErr = err
			return
		}
		store(i, c.(float64))
		i++
	})
	if castErr != nil {
		return nil, castErr
	}
	return newDense(data, a.shape, dt), nil
}

// Sorting ------------------------------------------------------------------

func sortSegment(data any, start, end int) {
	switch d := data.(type) {
	case []float64:
		sort.Float64s(d[start:end])
	case []float32:
		seg := d[start:end]
		sort.Slice(seg, func(i, j int) bool { return seg[i] < seg[j] })
	case []int:
		sort.Ints(d[start:end])
	case []int64:
		seg := d[start:end]
		sort.Slice(seg, func(i, j int) bool { return seg[i] < seg[j] })
	case []string:
		sort.Strings(d[start:end])
	case []bool:
		seg := d[start:end]
		falses := 0
		for _, v := range seg {
			if !v {
				falses++
			}
		}
		for i := range seg {
			seg[i] = i >= falses
		}
	}
}

// Sort returns a copy sorted along the last axis (np.sort default),
// preserving the dtype. Strings sort lexicographically.
func (a *NDArray) Sort() *NDArray {
	out := a.Copy()
	if out.NDim() == 0 || out.Size() == 0 {
		return out
	}
	rowLen := out.shape[len(out.shape)-1]
	if rowLen == 0 {
		return out
	}
	for start := 0; start < out.Size(); start += rowLen {
		sortSegment(out.data, start, start+rowLen)
	}
	return out
}

// ArgSort returns the indices (Int64) that would sort along the last
// axis (stable).
func (a *NDArray) ArgSort() *NDArray {
	c := a.Copy()
	out := make([]int64, c.Size())
	if c.NDim() == 0 || c.Size() == 0 {
		return newDense(out, c.shape, dtype.Int64)
	}
	rowLen := c.shape[len(c.shape)-1]
	var less func(start int, i, j int) bool
	if l := c.stringLoader(); l != nil {
		less = func(start, i, j int) bool { return l(start+i) < l(start+j) }
	} else {
		l := c.mustFloatLoader("argsort")
		less = func(start, i, j int) bool { return l(start+i) < l(start+j) }
	}
	for start := 0; start < c.Size(); start += rowLen {
		idx := make([]int, rowLen)
		for i := range idx {
			idx[i] = i
		}
		sort.SliceStable(idx, func(x, y int) bool { return less(start, idx[x], idx[y]) })
		for i, p := range idx {
			out[start+i] = int64(p)
		}
	}
	return newDense(out, c.shape, dtype.Int64)
}

// Unique returns the sorted distinct values of the flattened array,
// preserving the dtype, like np.unique.
func Unique(a *NDArray) *NDArray {
	sorted := a.Flatten().Sort()
	n := sorted.Size()
	if n == 0 {
		return sorted
	}
	var keep []int
	if l := sorted.stringLoader(); l != nil {
		for i := 0; i < n; i++ {
			if i == 0 || l(i) != l(i-1) {
				keep = append(keep, i)
			}
		}
	} else {
		l := sorted.mustFloatLoader("unique")
		for i := 0; i < n; i++ {
			if i == 0 || l(i) != l(i-1) {
				keep = append(keep, i)
			}
		}
	}
	out, _ := sorted.Take(keep, 0)
	return out
}

// Joining ---------------------------------------------------------------------

// Concatenate joins arrays along an existing axis, like np.concatenate.
// All arrays must share a dtype (numeric mixes promote).
func Concatenate(arrays []*NDArray, axis int) (*NDArray, error) {
	if len(arrays) == 0 {
		return nil, fmt.Errorf("%w: concatenate needs at least one array", errs.ErrInvalidOperation)
	}
	first := arrays[0]
	if axis < 0 {
		axis += first.NDim()
	}
	if err := first.checkAxis(axis); err != nil {
		return nil, err
	}
	outDT := first.dtype
	outShape := first.Shape()
	for _, arr := range arrays[1:] {
		if arr.NDim() != first.NDim() {
			return nil, fmt.Errorf("%w: concatenate with different dimensions", errs.ErrShapeMismatch)
		}
		for d := range outShape {
			if d == axis {
				continue
			}
			if arr.shape[d] != first.shape[d] {
				return nil, fmt.Errorf("%w: concatenate shapes %v and %v on axis %d", errs.ErrShapeMismatch, first.shape, arr.shape, axis)
			}
		}
		outShape[axis] += arr.shape[axis]
		if arr.dtype != outDT {
			p, err := promoteArith(outDT, arr.dtype)
			if err != nil {
				return nil, err
			}
			outDT = p
		}
	}
	out := newDense(allocData(outDT, shapeSize(outShape)), outShape, outDT)
	offset := 0
	for _, arr := range arrays {
		for local := 0; local < arr.shape[axis]; local++ {
			src, err := arr.axisSlice(axis, local)
			if err != nil {
				return nil, err
			}
			dst, err := out.axisSlice(axis, offset+local)
			if err != nil {
				return nil, err
			}
			if err := copyInto(dst, src); err != nil {
				return nil, err
			}
		}
		offset += arr.shape[axis]
	}
	return out, nil
}

// StackArrays joins arrays along a NEW axis, like np.stack.
func StackArrays(arrays []*NDArray, axis int) (*NDArray, error) {
	if len(arrays) == 0 {
		return nil, fmt.Errorf("%w: stack needs at least one array", errs.ErrInvalidOperation)
	}
	expanded := make([]*NDArray, len(arrays))
	for i, arr := range arrays {
		if !sameShape(arr.shape, arrays[0].shape) {
			return nil, fmt.Errorf("%w: stack shapes %v and %v", errs.ErrShapeMismatch, arrays[0].shape, arr.shape)
		}
		e, err := arr.ExpandDims(axis)
		if err != nil {
			return nil, err
		}
		expanded[i] = e
	}
	return Concatenate(expanded, axis)
}

// HStack joins arrays horizontally, like np.hstack: along axis 0 for 1-D
// arrays and axis 1 otherwise.
func HStack(arrays []*NDArray) (*NDArray, error) {
	if len(arrays) > 0 && arrays[0].NDim() == 1 {
		return Concatenate(arrays, 0)
	}
	return Concatenate(arrays, 1)
}

// VStack joins arrays vertically, like np.vstack: 1-D arrays are treated
// as rows.
func VStack(arrays []*NDArray) (*NDArray, error) {
	rows := make([]*NDArray, len(arrays))
	for i, arr := range arrays {
		if arr.NDim() == 1 {
			r, err := arr.Reshape(1, arr.Size())
			if err != nil {
				return nil, err
			}
			rows[i] = r
		} else {
			rows[i] = arr
		}
	}
	return Concatenate(rows, 0)
}

// NaN predicates -------------------------------------------------------------

// IsNaN marks NaN elements, like np.isnan.
func (a *NDArray) IsNaN() *BoolArray {
	return a.cmpScalar(func(x float64) bool { return math.IsNaN(x) })
}

// IsFinite marks finite elements, like np.isfinite.
func (a *NDArray) IsFinite() *BoolArray {
	return a.cmpScalar(func(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) })
}

// IsInf marks infinite elements, like np.isinf.
func (a *NDArray) IsInf() *BoolArray {
	return a.cmpScalar(func(x float64) bool { return math.IsInf(x, 0) })
}

// Masking ----------------------------------------------------------------------

// Mask returns the elements where mask is true as a 1-D array of the
// same dtype (NumPy boolean indexing flattens).
func (a *NDArray) Mask(mask *BoolArray) (*NDArray, error) {
	return Compress(mask, a)
}

// WhereScalar selects from a where mask is true and the scalar elsewhere
// (numeric arrays; the result keeps a's dtype when the scalar is
// integral, else Float64).
func WhereScalar(mask *BoolArray, a *NDArray, other float64) (*NDArray, error) {
	if !sameShape(mask.shape, a.shape) {
		return nil, fmt.Errorf("%w: where mask %v for array %v", errs.ErrShapeMismatch, mask.shape, a.Shape())
	}
	load := a.floatLoader()
	if load == nil {
		return nil, fmt.Errorf("%w: WhereScalar on %s array", errs.ErrTypeMismatch, a.dtype)
	}
	dt := a.scalarResultDType(other, true)
	data := allocData(dt, mask.Size())
	store := floatStore(data)
	i := 0
	a.iter(func(off int) {
		if mask.data[i] {
			store(i, load(off))
		} else {
			store(i, other)
		}
		i++
	})
	return newDense(data, mask.shape, dt), nil
}

// Binary root helpers -------------------------------------------------------------

// Maximum returns the elementwise maximum with broadcasting, like
// np.maximum (dtype-promoting).
func Maximum(a, b *NDArray) (*NDArray, error) {
	p, err := promoteArith(a.dtype, b.dtype)
	if err != nil {
		return nil, err
	}
	return binopAs(a, b, p, math.Max)
}

// Minimum returns the elementwise minimum with broadcasting, like
// np.minimum (dtype-promoting).
func Minimum(a, b *NDArray) (*NDArray, error) {
	p, err := promoteArith(a.dtype, b.dtype)
	if err != nil {
		return nil, err
	}
	return binopAs(a, b, p, math.Min)
}

// Reductions with ddof --------------------------------------------------------------

// VarDDof is Var with an explicit delta-degrees-of-freedom (np.var's
// ddof keyword). ddof=0 matches NumPy's default, ddof=1 matches pandas.
func (a *NDArray) VarDDof(ddof int, axis ...int) (*NDArray, error) {
	if len(axis) == 0 {
		n := a.Size()
		if n-ddof <= 0 {
			return scalarArray(math.NaN()), nil
		}
		load := a.mustFloatLoader("var")
		mean := a.MeanAll()
		acc := 0.0
		a.iter(func(off int) {
			d := load(off) - mean
			acc += d * d
		})
		return scalarArray(acc / float64(n-ddof)), nil
	}
	v, err := a.Var(axis...)
	if err != nil {
		return nil, err
	}
	// rescale population variance (ddof=0) to the requested ddof
	n := float64(a.shape[axis[0]])
	if n-float64(ddof) <= 0 {
		return Full(math.NaN(), v.Shape()...), nil
	}
	return v.MulScalar(n / (n - float64(ddof))), nil
}

// StdDDof is Std with an explicit delta-degrees-of-freedom.
func (a *NDArray) StdDDof(ddof int, axis ...int) (*NDArray, error) {
	v, err := a.VarDDof(ddof, axis...)
	if err != nil {
		return nil, err
	}
	return v.Sqrt(), nil
}

// AsArray copies any supported slice into a 1-D array (np.asarray-ish).
func AsArray[T Element](data []T) *NDArray { return ArrayOf(data) }
