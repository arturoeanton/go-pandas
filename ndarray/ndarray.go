// Package ndarray implements a NumPy-style n-dimensional array with
// shape/strides views, slicing, broadcasting, ufunc-like math, reductions
// and basic linear algebra. Since v0.3 storage is typed: bool, int,
// int64, float32, float64 and string backings.
package ndarray

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// Number constrains the element types accepted by numeric constructors.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// NDArray is an n-dimensional, row-major array over a typed backing
// slice. Views share the underlying buffer and are materialized only on
// Copy.
type NDArray struct {
	data    any // []bool | []int | []int64 | []float32 | []float64 | []string
	shape   []int
	strides []int
	offset  int
	dtype   dtype.DType
	view    bool
}

// computeStrides returns the row-major strides for a shape.
func computeStrides(shape []int) []int {
	strides := make([]int, len(shape))
	acc := 1
	for i := len(shape) - 1; i >= 0; i-- {
		strides[i] = acc
		acc *= shape[i]
	}
	return strides
}

func shapeSize(shape []int) int {
	n := 1
	for _, d := range shape {
		n *= d
	}
	return n
}

// Shape returns a copy of the array shape.
func (a *NDArray) Shape() []int { return append([]int(nil), a.shape...) }

// Strides returns a copy of the array strides (in elements, not bytes).
func (a *NDArray) Strides() []int { return append([]int(nil), a.strides...) }

// Size returns the total number of elements.
func (a *NDArray) Size() int { return shapeSize(a.shape) }

// NDim returns the number of dimensions.
func (a *NDArray) NDim() int { return len(a.shape) }

// DType returns the element dtype.
func (a *NDArray) DType() dtype.DType { return a.dtype }

// StorageDType returns the dtype of the physical backing slice. Since
// v0.3 it always equals DType().
func (a *NDArray) StorageDType() dtype.DType { return dtypeOfData(a.data) }

// IsView reports whether the array shares its buffer with another array.
func (a *NDArray) IsView() bool { return a.view }

// RawData returns the typed backing slice ([]int, []float64, ...) of the
// whole buffer. For views it includes elements outside the view; use
// Copy().RawData() for a dense, view-free backing. Treat it as
// read-only unless you own the array.
func (a *NDArray) RawData() any { return a.data }

// Data returns the elements converted to float64 in logical (row-major)
// order. For contiguous, non-view Float64 arrays this is the backing
// slice itself (treat it as read-only); other numeric dtypes are
// converted copies. String arrays return nil — use Values().
func (a *NDArray) Data() []float64 {
	if d, ok := a.data.([]float64); ok && !a.view && a.offset == 0 && a.isContiguous() {
		return d
	}
	load := a.floatLoader()
	if load == nil {
		return nil
	}
	out := make([]float64, 0, a.Size())
	a.iter(func(off int) {
		out = append(out, load(off))
	})
	return out
}

// Values returns the elements boxed as []any in logical order, for any
// dtype (including strings).
func (a *NDArray) Values() []any {
	out := make([]any, 0, a.Size())
	a.iter(func(off int) {
		out = append(out, a.valueAt(off))
	})
	return out
}

// Copy returns a compact, contiguous deep copy of the array.
func (a *NDArray) Copy() *NDArray {
	return newDense(a.materialize(), a.shape, a.dtype)
}

// Clone is an alias of Copy.
func (a *NDArray) Clone() *NDArray { return a.Copy() }

func (a *NDArray) isContiguous() bool {
	expected := computeStrides(a.shape)
	for i, s := range a.strides {
		if a.shape[i] > 1 && s != expected[i] {
			return false
		}
	}
	return true
}

// iter walks every element in row-major logical order, calling f with the
// physical offset into the backing slice.
func (a *NDArray) iter(f func(offset int)) {
	if len(a.shape) == 0 {
		f(a.offset)
		return
	}
	size := a.Size()
	if size == 0 {
		return
	}
	coords := make([]int, len(a.shape))
	for {
		off := a.offset
		for d, c := range coords {
			off += c * a.strides[d]
		}
		f(off)
		// increment coords
		d := len(coords) - 1
		for d >= 0 {
			coords[d]++
			if coords[d] < a.shape[d] {
				break
			}
			coords[d] = 0
			d--
		}
		if d < 0 {
			return
		}
	}
}

// checkAxis validates an axis for this array.
func (a *NDArray) checkAxis(axis int) error {
	if axis < 0 || axis >= len(a.shape) {
		return fmt.Errorf("%w: axis %d for array of dimension %d", errs.ErrInvalidAxis, axis, len(a.shape))
	}
	return nil
}

// String renders the array in a NumPy-like "array([...])" form.
func (a *NDArray) String() string {
	var b strings.Builder
	b.WriteString("array(")
	b.WriteString(a.format(nil, len("array(")))
	b.WriteString(")")
	return b.String()
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

func (a *NDArray) formatElem(off int) string {
	switch v := a.valueAt(off).(type) {
	case float64:
		return formatFloat(v)
	case float32:
		return formatFloat(float64(v))
	case string:
		return strconv.Quote(v)
	default:
		return fmt.Sprint(v)
	}
}

// format renders the (sub-)array selected by the given leading coords.
func (a *NDArray) format(coords []int, indent int) string {
	if len(coords) == len(a.shape) {
		off := a.offset
		for d, c := range coords {
			off += c * a.strides[d]
		}
		return a.formatElem(off)
	}
	dim := a.shape[len(coords)]
	parts := make([]string, dim)
	for i := 0; i < dim; i++ {
		parts[i] = a.format(append(coords, i), indent+1)
	}
	sep := ", "
	if len(coords) < len(a.shape)-1 {
		sep = ",\n" + strings.Repeat(" ", indent+1)
	}
	return "[" + strings.Join(parts, sep) + "]"
}
