package ndarray

import (
	"fmt"
	"math"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// Element constrains the Go types an NDArray can store (v0.3 typed
// storage).
type Element interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 | ~bool | ~string
}

// allocData allocates a dense typed backing for a dtype.
func allocData(dt dtype.DType, n int) any {
	switch dt {
	case dtype.Bool:
		return make([]bool, n)
	case dtype.Int:
		return make([]int, n)
	case dtype.Int64:
		return make([]int64, n)
	case dtype.Float32:
		return make([]float32, n)
	case dtype.String:
		return make([]string, n)
	default:
		return make([]float64, n)
	}
}

// dtypeOfData maps a backing slice to its dtype.
func dtypeOfData(data any) dtype.DType {
	switch data.(type) {
	case []bool:
		return dtype.Bool
	case []int:
		return dtype.Int
	case []int64:
		return dtype.Int64
	case []float32:
		return dtype.Float32
	case []float64:
		return dtype.Float64
	case []string:
		return dtype.String
	}
	return dtype.Invalid
}

func dataLen(data any) int {
	switch d := data.(type) {
	case []bool:
		return len(d)
	case []int:
		return len(d)
	case []int64:
		return len(d)
	case []float32:
		return len(d)
	case []float64:
		return len(d)
	case []string:
		return len(d)
	}
	return 0
}

// floatLoader returns a closure reading elements as float64 (nil for
// string backings). The closure form keeps the type switch out of the
// per-element hot loop.
func (a *NDArray) floatLoader() func(off int) float64 {
	switch d := a.data.(type) {
	case []float64:
		return func(off int) float64 { return d[off] }
	case []float32:
		return func(off int) float64 { return float64(d[off]) }
	case []int:
		return func(off int) float64 { return float64(d[off]) }
	case []int64:
		return func(off int) float64 { return float64(d[off]) }
	case []bool:
		return func(off int) float64 {
			if d[off] {
				return 1
			}
			return 0
		}
	}
	return nil
}

// mustFloatLoader panics with a descriptive error for string arrays; the
// numeric-only entry points document this.
func (a *NDArray) mustFloatLoader(op string) func(off int) float64 {
	l := a.floatLoader()
	if l == nil {
		panic(fmt.Errorf("%w: %s on %s array", errs.ErrTypeMismatch, op, a.dtype))
	}
	return l
}

// stringLoader reads elements as strings (nil for non-string backings).
func (a *NDArray) stringLoader() func(off int) string {
	if d, ok := a.data.([]string); ok {
		return func(off int) string { return d[off] }
	}
	return nil
}

// valueAt boxes the element at a physical offset (used by formatting and
// Values()).
func (a *NDArray) valueAt(off int) any {
	switch d := a.data.(type) {
	case []bool:
		return d[off]
	case []int:
		return d[off]
	case []int64:
		return d[off]
	case []float32:
		return d[off]
	case []float64:
		return d[off]
	case []string:
		return d[off]
	}
	return nil
}

// floatStore returns a closure writing float64 values into a typed
// backing (integers truncate, bools store v != 0). Nil for strings.
func floatStore(data any) func(pos int, v float64) {
	switch d := data.(type) {
	case []float64:
		return func(pos int, v float64) { d[pos] = v }
	case []float32:
		return func(pos int, v float64) { d[pos] = float32(v) }
	case []int:
		return func(pos int, v float64) { d[pos] = int(math.Trunc(v)) }
	case []int64:
		return func(pos int, v float64) { d[pos] = int64(math.Trunc(v)) }
	case []bool:
		return func(pos int, v float64) { d[pos] = v != 0 }
	}
	return nil
}

// materialize gathers the logical elements of a (view or not) into a new
// dense typed backing of the same dtype.
func (a *NDArray) materialize() any {
	n := a.Size()
	switch d := a.data.(type) {
	case []bool:
		out := make([]bool, 0, n)
		a.iter(func(off int) { out = append(out, d[off]) })
		return out
	case []int:
		out := make([]int, 0, n)
		a.iter(func(off int) { out = append(out, d[off]) })
		return out
	case []int64:
		out := make([]int64, 0, n)
		a.iter(func(off int) { out = append(out, d[off]) })
		return out
	case []float32:
		out := make([]float32, 0, n)
		a.iter(func(off int) { out = append(out, d[off]) })
		return out
	case []float64:
		out := make([]float64, 0, n)
		a.iter(func(off int) { out = append(out, d[off]) })
		return out
	case []string:
		out := make([]string, 0, n)
		a.iter(func(off int) { out = append(out, d[off]) })
		return out
	}
	return make([]float64, n)
}

// promoteArith computes the result dtype of an arithmetic operation
// following the documented NumPy-style rules. String operands are an
// error; bool arithmetic promotes to Int.
func promoteArith(a, b dtype.DType) (dtype.DType, error) {
	if a == dtype.String || b == dtype.String {
		return dtype.Invalid, fmt.Errorf("%w: arithmetic on string arrays", errs.ErrTypeMismatch)
	}
	p := dtype.Promote(a, b)
	if p == dtype.Bool {
		p = dtype.Int
	}
	if !dtype.IsNumeric(p) {
		return dtype.Invalid, fmt.Errorf("%w: arithmetic between %s and %s", errs.ErrTypeMismatch, a, b)
	}
	return p, nil
}

// divDType maps a promoted dtype to the true-division result dtype:
// integer and bool divisions produce Float64, floats keep their width.
func divDType(p dtype.DType) dtype.DType {
	if dtype.IsFloat(p) {
		return p
	}
	return dtype.Float64
}

// newDense builds a dense array around a typed backing.
func newDense(data any, shape []int, dt dtype.DType) *NDArray {
	return &NDArray{
		data:    data,
		shape:   append([]int(nil), shape...),
		strides: computeStrides(shape),
		dtype:   dt,
	}
}
