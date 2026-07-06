package ndarray

import (
	"fmt"
	"math"
	"sort"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// Typed constructors ------------------------------------------------------
// v0.2 keeps float64 storage; these constructors record the logical dtype
// while converting values, so a.DType() round-trips (see known
// differences).

func typedArray[T Number](data []T, dt dtype.DType) *NDArray {
	a := ArrayOf(data)
	a.dtype = dt
	return a
}

// ArrayInt builds a 1-D array from ints.
func ArrayInt(data []int) *NDArray { return typedArray(data, dtype.Int) }

// ArrayInt64 builds a 1-D array from int64s.
func ArrayInt64(data []int64) *NDArray { return typedArray(data, dtype.Int64) }

// ArrayFloat32 builds a 1-D array from float32s.
func ArrayFloat32(data []float32) *NDArray { return typedArray(data, dtype.Float32) }

// ArrayFloat64 builds a 1-D array from float64s.
func ArrayFloat64(data []float64) *NDArray { return Array(data) }

// ArrayBool builds a 1-D array from bools (true -> 1, false -> 0).
func ArrayBool(data []bool) *NDArray {
	out := make([]float64, len(data))
	for i, v := range data {
		if v {
			out[i] = 1
		}
	}
	a := Array(out)
	a.dtype = dtype.Bool
	return a
}

// Astype returns a copy with the target logical dtype; integer targets
// truncate values. Storage remains float64 in v0.2.
func (a *NDArray) Astype(dt dtype.DType) (*NDArray, error) {
	if !dtype.IsNumeric(dt) && !dtype.IsBool(dt) {
		return nil, fmt.Errorf("%w: NDArray.Astype to %s", errs.ErrInvalidDType, dt)
	}
	out := a.Copy()
	out.dtype = dt
	if dtype.IsInteger(dt) {
		for i := range out.data {
			out.data[i] = math.Trunc(out.data[i])
		}
	}
	if dt == dtype.Bool {
		for i := range out.data {
			if out.data[i] != 0 {
				out.data[i] = 1
			}
		}
	}
	return out, nil
}

// Sorting ------------------------------------------------------------------

// Sort returns a copy sorted along the last axis (np.sort default). For a
// 1-D array this is a plain ascending sort.
func (a *NDArray) Sort() *NDArray {
	out := a.Copy()
	if out.NDim() == 0 {
		return out
	}
	rowLen := out.shape[len(out.shape)-1]
	if rowLen == 0 {
		return out
	}
	for start := 0; start < len(out.data); start += rowLen {
		sort.Float64s(out.data[start : start+rowLen])
	}
	return out
}

// ArgSort returns the indices that would sort along the last axis.
func (a *NDArray) ArgSort() *NDArray {
	c := a.Copy()
	out := Zeros(c.shape...)
	if c.NDim() == 0 || c.Size() == 0 {
		return out
	}
	rowLen := c.shape[len(c.shape)-1]
	for start := 0; start < len(c.data); start += rowLen {
		row := c.data[start : start+rowLen]
		idx := make([]int, rowLen)
		for i := range idx {
			idx[i] = i
		}
		sort.SliceStable(idx, func(x, y int) bool { return row[idx[x]] < row[idx[y]] })
		for i, p := range idx {
			out.data[start+i] = float64(p)
		}
	}
	return out
}

// Unique returns the sorted distinct values of the flattened array, like
// np.unique.
func Unique(a *NDArray) *NDArray {
	// Data() may alias the backing buffer for contiguous arrays; copy
	// before sorting so the input array is never mutated.
	data := append([]float64(nil), a.Data()...)
	sort.Float64s(data)
	var out []float64
	for i, v := range data {
		if i == 0 || v != data[i-1] {
			out = append(out, v)
		}
	}
	return Array(out)
}

// Joining -------------------------------------------------------------------

// Concatenate joins arrays along an existing axis, like np.concatenate.
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
	}
	out := Zeros(outShape...)
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
			data := src.Data()
			i := 0
			dst.iter(func(off int) {
				dst.data[off] = data[i]
				i++
			})
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

// Mask returns the elements where mask is true as a 1-D array (NumPy
// boolean indexing flattens).
func (a *NDArray) Mask(mask *BoolArray) (*NDArray, error) {
	return Compress(mask, a)
}

// WhereScalar selects from a where mask is true and the scalar elsewhere.
func WhereScalar(mask *BoolArray, a *NDArray, other float64) (*NDArray, error) {
	if !sameShape(mask.shape, a.shape) {
		return nil, fmt.Errorf("%w: where mask %v for array %v", errs.ErrShapeMismatch, mask.shape, a.Shape())
	}
	out := Zeros(mask.shape...)
	data := a.Data()
	for i := range out.data {
		if mask.data[i] {
			out.data[i] = data[i]
		} else {
			out.data[i] = other
		}
	}
	return out, nil
}

// Binary root helpers -------------------------------------------------------------

// Maximum returns the elementwise maximum with broadcasting, like
// np.maximum.
func Maximum(a, b *NDArray) (*NDArray, error) { return binop(a, b, math.Max) }

// Minimum returns the elementwise minimum with broadcasting, like
// np.minimum.
func Minimum(a, b *NDArray) (*NDArray, error) { return binop(a, b, math.Min) }

// Reductions with ddof --------------------------------------------------------------

// VarDDof is Var with an explicit delta-degrees-of-freedom (np.var's
// ddof keyword). ddof=0 matches NumPy's default, ddof=1 matches pandas.
func (a *NDArray) VarDDof(ddof int, axis ...int) (*NDArray, error) {
	if len(axis) == 0 {
		n := a.Size()
		if n-ddof <= 0 {
			return scalarArray(math.NaN()), nil
		}
		mean := a.MeanAll()
		acc := 0.0
		a.iter(func(off int) {
			d := a.data[off] - mean
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

// AsArray copies any numeric slice into a 1-D array (np.asarray-ish).
func AsArray[T Number](data []T) *NDArray { return ArrayOf(data) }
