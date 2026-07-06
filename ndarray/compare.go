package ndarray

import (
	"fmt"
	"strings"
)

// BoolArray is the result of elementwise comparisons: an n-dimensional
// boolean mask.
type BoolArray struct {
	data  []bool
	shape []int
}

// Shape returns a copy of the mask shape.
func (b *BoolArray) Shape() []int { return append([]int(nil), b.shape...) }

// Size returns the number of elements.
func (b *BoolArray) Size() int { return shapeSize(b.shape) }

// Data returns the flattened boolean values in row-major order.
func (b *BoolArray) Data() []bool { return append([]bool(nil), b.data...) }

// At returns the element at flat position i.
func (b *BoolArray) At(i int) bool { return b.data[i] }

// CountTrue returns how many elements are true.
func (b *BoolArray) CountTrue() int {
	n := 0
	for _, v := range b.data {
		if v {
			n++
		}
	}
	return n
}

// Any reports whether at least one element is true.
func (b *BoolArray) Any() bool {
	for _, v := range b.data {
		if v {
			return true
		}
	}
	return false
}

// All reports whether every element is true.
func (b *BoolArray) All() bool {
	for _, v := range b.data {
		if !v {
			return false
		}
	}
	return true
}

// Not returns the elementwise negation.
func (b *BoolArray) Not() *BoolArray {
	out := make([]bool, len(b.data))
	for i, v := range b.data {
		out[i] = !v
	}
	return &BoolArray{data: out, shape: append([]int(nil), b.shape...)}
}

func (b *BoolArray) String() string {
	parts := make([]string, len(b.data))
	for i, v := range b.data {
		parts[i] = fmt.Sprint(v)
	}
	return "array([" + strings.Join(parts, ", ") + "])"
}

// cmpOp applies a boolean predicate over the broadcast of a and b.
func cmpOp(a, b *NDArray, f func(x, y float64) bool) (*BoolArray, error) {
	shape, err := BroadcastShapes(a.shape, b.shape)
	if err != nil {
		return nil, err
	}
	out := &BoolArray{data: make([]bool, shapeSize(shape)), shape: shape}
	iter2(a, b, shape, func(pos, offA, offB int) {
		out.data[pos] = f(a.data[offA], b.data[offB])
	})
	return out, nil
}

// Eq returns a == b elementwise.
func (a *NDArray) Eq(b *NDArray) (*BoolArray, error) {
	return cmpOp(a, b, func(x, y float64) bool { return x == y })
}

// Ne returns a != b elementwise.
func (a *NDArray) Ne(b *NDArray) (*BoolArray, error) {
	return cmpOp(a, b, func(x, y float64) bool { return x != y })
}

// Gt returns a > b elementwise.
func (a *NDArray) Gt(b *NDArray) (*BoolArray, error) {
	return cmpOp(a, b, func(x, y float64) bool { return x > y })
}

// Ge returns a >= b elementwise.
func (a *NDArray) Ge(b *NDArray) (*BoolArray, error) {
	return cmpOp(a, b, func(x, y float64) bool { return x >= y })
}

// Lt returns a < b elementwise.
func (a *NDArray) Lt(b *NDArray) (*BoolArray, error) {
	return cmpOp(a, b, func(x, y float64) bool { return x < y })
}

// Le returns a <= b elementwise.
func (a *NDArray) Le(b *NDArray) (*BoolArray, error) {
	return cmpOp(a, b, func(x, y float64) bool { return x <= y })
}

// GtScalar returns a > v elementwise (and siblings below).
func (a *NDArray) GtScalar(v float64) *BoolArray {
	return a.cmpScalar(func(x float64) bool { return x > v })
}
func (a *NDArray) GeScalar(v float64) *BoolArray {
	return a.cmpScalar(func(x float64) bool { return x >= v })
}
func (a *NDArray) LtScalar(v float64) *BoolArray {
	return a.cmpScalar(func(x float64) bool { return x < v })
}
func (a *NDArray) LeScalar(v float64) *BoolArray {
	return a.cmpScalar(func(x float64) bool { return x <= v })
}
func (a *NDArray) EqScalar(v float64) *BoolArray {
	return a.cmpScalar(func(x float64) bool { return x == v })
}
func (a *NDArray) NeScalar(v float64) *BoolArray {
	return a.cmpScalar(func(x float64) bool { return x != v })
}

func (a *NDArray) cmpScalar(f func(x float64) bool) *BoolArray {
	out := &BoolArray{data: make([]bool, a.Size()), shape: a.Shape()}
	i := 0
	a.iter(func(off int) {
		out.data[i] = f(a.data[off])
		i++
	})
	return out
}
