// Package ndarray implements a NumPy-style n-dimensional array with
// shape/strides views, slicing, broadcasting, ufunc-like math, reductions
// and basic linear algebra. v0.1 stores float64 elements.
package ndarray

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// Number constrains the element types accepted by generic constructors.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// NDArray is an n-dimensional, row-major array of float64. Views share the
// underlying data buffer and are materialized only on Copy.
type NDArray struct {
	data    []float64
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

// DType returns the element dtype (Float64 in v0.1).
func (a *NDArray) DType() dtype.DType { return a.dtype }

// IsView reports whether the array shares its buffer with another array.
func (a *NDArray) IsView() bool { return a.view }

// Data returns the flattened elements in logical (row-major) order. For
// contiguous non-view arrays this is the backing slice itself.
func (a *NDArray) Data() []float64 {
	if !a.view && a.offset == 0 && a.isContiguous() {
		return a.data
	}
	out := make([]float64, 0, a.Size())
	a.iter(func(off int) {
		out = append(out, a.data[off])
	})
	return out
}

// Copy returns a compact, contiguous deep copy of the array.
func (a *NDArray) Copy() *NDArray {
	out := &NDArray{
		data:    a.Data(),
		shape:   append([]int(nil), a.shape...),
		dtype:   a.dtype,
		strides: computeStrides(a.shape),
	}
	if !a.view && a.offset == 0 && a.isContiguous() {
		out.data = append([]float64(nil), a.data...)
	}
	return out
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
// physical offset into a.data.
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
	s := strconv.FormatFloat(v, 'g', -1, 64)
	return s
}

// format renders the (sub-)array selected by the given leading coords.
func (a *NDArray) format(coords []int, indent int) string {
	if len(coords) == len(a.shape) {
		off := a.offset
		for d, c := range coords {
			off += c * a.strides[d]
		}
		return formatFloat(a.data[off])
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
