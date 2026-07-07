package ndarray

import (
	"errors"
	"math"
	"testing"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

func TestTypedConstructorsAndAstype(t *testing.T) {
	a := ArrayInt([]int{1, 2, 3})
	if a.DType() != dtype.Int {
		t.Errorf("ArrayInt dtype = %v", a.DType())
	}
	assertData(t, a, []float64{1, 2, 3})
	b := ArrayBool([]bool{true, false, true})
	if b.DType() != dtype.Bool {
		t.Errorf("ArrayBool dtype = %v", b.DType())
	}
	assertData(t, b, []float64{1, 0, 1})
	if ArrayFloat32([]float32{1.5}).DType() != dtype.Float32 {
		t.Error("ArrayFloat32 dtype")
	}
	casted, err := Array([]float64{1.7, -2.7}).Astype(dtype.Int64)
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, casted, []float64{1, -2})
	if casted.DType() != dtype.Int64 {
		t.Errorf("Astype dtype = %v", casted.DType())
	}
	if _, ok := casted.RawData().([]int64); !ok {
		t.Errorf("Astype(Int64) backing = %T, want []int64", casted.RawData())
	}
	// v0.3: string conversion is real
	asStr, err := Array([]float64{1.5}).Astype(dtype.String)
	if err != nil {
		t.Fatal(err)
	}
	if vals := asStr.Values(); vals[0] != "1.5" {
		t.Errorf("Astype(String) = %v", vals)
	}
	parsed, err := ArrayString([]string{"42", "7"}).Astype(dtype.Int)
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, parsed, []float64{42, 7})
	if _, err := ArrayString([]string{"abc"}).Astype(dtype.Int64); err == nil {
		t.Error("Astype of non-numeric string should error")
	}
	if _, err := Array([]float64{1}).Astype(dtype.Time); !errors.Is(err, errs.ErrInvalidDType) {
		t.Errorf("Astype to unsupported dtype error = %v", err)
	}
}

func TestSortArgSortUnique(t *testing.T) {
	a := Array([]float64{3, 1, 2, 3, 1})
	assertData(t, a.Sort(), []float64{1, 1, 2, 3, 3})
	assertData(t, a.ArgSort(), []float64{1, 4, 2, 0, 3})
	assertData(t, Unique(a), []float64{1, 2, 3})
	m := MustFromSlice([]float64{3, 1, 2, 9, 7, 8}, 2, 3)
	assertData(t, m.Sort(), []float64{1, 2, 3, 7, 8, 9})
	assertData(t, m.ArgSort(), []float64{1, 2, 0, 1, 2, 0})
	// the source array is untouched
	assertData(t, a, []float64{3, 1, 2, 3, 1})
}

func TestJoining(t *testing.T) {
	v := Array([]float64{1, 2})
	w := Array([]float64{3, 4})
	cat, err := Concatenate([]*NDArray{v, w}, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, cat, []float64{1, 2, 3, 4})
	st, err := StackArrays([]*NDArray{v, w}, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, st, 2, 2)
	vs, err := VStack([]*NDArray{v, w})
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, vs, 2, 2)
	hs, err := HStack([]*NDArray{v, w})
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, hs, 4)
	m := MustFromSlice([]float64{1, 2, 3, 4}, 2, 2)
	wide, err := HStack([]*NDArray{m, m})
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, wide, 2, 4)
	if _, err := Concatenate([]*NDArray{v, m}, 0); !errors.Is(err, errs.ErrShapeMismatch) {
		t.Errorf("mismatched concatenate error = %v", err)
	}
	if _, err := StackArrays([]*NDArray{v, Array([]float64{1, 2, 3})}, 0); !errors.Is(err, errs.ErrShapeMismatch) {
		t.Errorf("mismatched stack error = %v", err)
	}
}

func TestNaNPredicatesAndMask(t *testing.T) {
	a := Array([]float64{1, math.NaN(), math.Inf(1)})
	if got := a.IsNaN().Data(); got[0] || !got[1] || got[2] {
		t.Errorf("IsNaN = %v", got)
	}
	if got := a.IsFinite().Data(); !got[0] || got[1] || got[2] {
		t.Errorf("IsFinite = %v", got)
	}
	if got := a.IsInf().Data(); got[0] || got[1] || !got[2] {
		t.Errorf("IsInf = %v", got)
	}
	m := MustFromSlice([]float64{1, 5, 3, 7}, 2, 2)
	masked, err := m.Mask(m.GtScalar(3))
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, masked, []float64{5, 7})
	sel, err := WhereScalar(m.GtScalar(3), m, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, sel, []float64{0, 5, 0, 7})
}

func TestMinMaxBinaryAndDDof(t *testing.T) {
	a := Array([]float64{1, 5})
	b := Array([]float64{3, 2})
	mx, err := Maximum(a, b)
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, mx, []float64{3, 5})
	mn, _ := Minimum(a, b)
	assertData(t, mn, []float64{1, 2})

	m := MustFromSlice([]float64{0, 1, 2, 3, 4, 5}, 2, 3)
	v1, err := m.VarDDof(1)
	if err != nil {
		t.Fatal(err)
	}
	if !almostEqual(v1.MustAt(), 3.5) {
		t.Errorf("VarDDof(1) = %v", v1.MustAt())
	}
	s1, _ := m.StdDDof(1, 1)
	assertData(t, s1, []float64{1, 1})
}
