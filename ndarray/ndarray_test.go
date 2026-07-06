package ndarray

import (
	"errors"
	"math"
	"testing"

	"github.com/arturoeanton/go-pandas/errs"
)

const tol = 1e-9

func almostEqual(a, b float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	return math.Abs(a-b) <= tol
}

func assertData(t *testing.T, a *NDArray, want []float64) {
	t.Helper()
	got := a.Data()
	if len(got) != len(want) {
		t.Fatalf("data length %d, want %d (%v vs %v)", len(got), len(want), got, want)
	}
	for i := range want {
		if !almostEqual(got[i], want[i]) {
			t.Fatalf("data[%d] = %v, want %v (full: %v)", i, got[i], want[i], got)
		}
	}
}

func assertShape(t *testing.T, a *NDArray, want ...int) {
	t.Helper()
	got := a.Shape()
	if len(got) != len(want) {
		t.Fatalf("shape %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("shape %v, want %v", got, want)
		}
	}
}

func TestConstructors(t *testing.T) {
	a := Array([]float64{1, 2, 3})
	assertShape(t, a, 3)
	if a.Size() != 3 || a.NDim() != 1 {
		t.Errorf("size/ndim: %d, %d", a.Size(), a.NDim())
	}
	z := Zeros(2, 3)
	assertShape(t, z, 2, 3)
	assertData(t, z, []float64{0, 0, 0, 0, 0, 0})
	o := Ones(2, 2)
	assertData(t, o, []float64{1, 1, 1, 1})
	f := Full(7, 3)
	assertData(t, f, []float64{7, 7, 7})
	assertData(t, Arange(5), []float64{0, 1, 2, 3, 4})
	assertData(t, Arange(2, 8, 2), []float64{2, 4, 6})
	assertData(t, Linspace(0, 1, 5), []float64{0, 0.25, 0.5, 0.75, 1})
	e := Eye(3)
	assertData(t, e, []float64{1, 0, 0, 0, 1, 0, 0, 0, 1})
	d, err := Diag(Array([]float64{2, 3}))
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, d, []float64{2, 0, 0, 3})
	if _, err := FromSlice([]float64{1, 2, 3}, 2, 2); !errors.Is(err, errs.ErrShapeMismatch) {
		t.Errorf("FromSlice mismatch error = %v", err)
	}
	g := ArrayOf([]int{1, 2, 3})
	assertData(t, g, []float64{1, 2, 3})
}

func TestStrides(t *testing.T) {
	a := Zeros(2, 3, 4)
	s := a.Strides()
	if s[0] != 12 || s[1] != 4 || s[2] != 1 {
		t.Errorf("strides = %v", s)
	}
}

func TestAtSet(t *testing.T) {
	a := MustFromSlice([]float64{1, 2, 3, 4, 5, 6}, 2, 3)
	if v, _ := a.At(1, 2); v != 6 {
		t.Errorf("At(1,2) = %v", v)
	}
	if v, _ := a.At(-1, -1); v != 6 {
		t.Errorf("At(-1,-1) = %v", v)
	}
	if err := a.Set(9, 0, 1); err != nil {
		t.Fatal(err)
	}
	if v := a.MustAt(0, 1); v != 9 {
		t.Errorf("after Set, At(0,1) = %v", v)
	}
	if _, err := a.At(5, 0); !errors.Is(err, errs.ErrIndexOutOfBounds) {
		t.Errorf("At out of bounds error = %v", err)
	}
	if _, err := a.At(0); !errors.Is(err, errs.ErrIndexOutOfBounds) {
		t.Errorf("At with wrong arity error = %v", err)
	}
}

func TestReshapeFlattenRavel(t *testing.T) {
	a := Arange(6)
	b, err := a.Reshape(2, 3)
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, b, 2, 3)
	if !b.IsView() {
		t.Error("contiguous reshape should be a view")
	}
	c, err := a.Reshape(3, -1)
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, c, 3, 2)
	if _, err := a.Reshape(4, 2); !errors.Is(err, errs.ErrShapeMismatch) {
		t.Errorf("bad reshape error = %v", err)
	}
	assertData(t, b.Flatten(), []float64{0, 1, 2, 3, 4, 5})
	assertShape(t, b.Ravel(), 6)
}

func TestTranspose(t *testing.T) {
	a := MustFromSlice([]float64{1, 2, 3, 4, 5, 6}, 2, 3)
	tr, err := a.T()
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, tr, 3, 2)
	assertData(t, tr, []float64{1, 4, 2, 5, 3, 6})
	// transpose of transpose is the original
	tt, _ := tr.T()
	assertData(t, tt, []float64{1, 2, 3, 4, 5, 6})
}

func TestSlice(t *testing.T) {
	a := MustFromSlice([]float64{
		0, 1, 2, 3,
		4, 5, 6, 7,
		8, 9, 10, 11,
	}, 3, 4)
	s, err := a.Slice(Slice(0, 2), Slice(1, 3))
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, s, 2, 2)
	assertData(t, s, []float64{1, 2, 5, 6})
	if !s.IsView() {
		t.Error("slice should be a view")
	}
	// mutating the view mutates the base
	if err := s.Set(99, 0, 0); err != nil {
		t.Fatal(err)
	}
	if a.MustAt(0, 1) != 99 {
		t.Error("view mutation did not propagate")
	}
	// stepped slice
	st, err := a.Slice(SliceStep(0, 3, 2))
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, st, 2, 4)
	assertData(t, st, []float64{0, 99, 2, 3, 8, 9, 10, 11})
}

func TestScalarOps(t *testing.T) {
	a := Array([]float64{1, 2, 3})
	assertData(t, a.AddScalar(10), []float64{11, 12, 13})
	assertData(t, a.SubScalar(1), []float64{0, 1, 2})
	assertData(t, a.MulScalar(2), []float64{2, 4, 6})
	assertData(t, a.DivScalar(2), []float64{0.5, 1, 1.5})
	assertData(t, a.PowScalar(2), []float64{1, 4, 9})
}

func TestBroadcasting(t *testing.T) {
	// (2,3) + (3,)
	a := MustFromSlice([]float64{1, 2, 3, 4, 5, 6}, 2, 3)
	b := Array([]float64{10, 20, 30})
	c, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, c, 2, 3)
	assertData(t, c, []float64{11, 22, 33, 14, 25, 36})

	// (2,1) + (1,3)
	x := MustFromSlice([]float64{1, 2}, 2, 1)
	y := MustFromSlice([]float64{10, 20, 30}, 1, 3)
	z, err := x.Add(y)
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, z, 2, 3)
	assertData(t, z, []float64{11, 21, 31, 12, 22, 32})

	// (5,1) + (6,) -> (5,6)
	p := Zeros(5, 1)
	q := Zeros(6)
	r, err := p.Add(q)
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, r, 5, 6)

	// (8,1,6,1) + (7,1,5) -> (8,7,6,5)
	shape, err := BroadcastShapes([]int{8, 1, 6, 1}, []int{7, 1, 5})
	if err != nil {
		t.Fatal(err)
	}
	if shape[0] != 8 || shape[1] != 7 || shape[2] != 6 || shape[3] != 5 {
		t.Errorf("broadcast shape = %v", shape)
	}

	// incompatible: (3,) + (4,)
	if _, err := Array([]float64{1, 2, 3}).Add(Zeros(4)); !errors.Is(err, errs.ErrBroadcastMismatch) {
		t.Errorf("(3,)+(4,) error = %v", err)
	}
	// incompatible: (4,3) + (4,)
	if _, err := Zeros(4, 3).Add(Zeros(4)); !errors.Is(err, errs.ErrBroadcastMismatch) {
		t.Errorf("(4,3)+(4,) error = %v", err)
	}
}

func TestElementwiseOps(t *testing.T) {
	a := Array([]float64{4, 9})
	b := Array([]float64{2, 3})
	got, _ := a.Mul(b)
	assertData(t, got, []float64{8, 27})
	got, _ = a.Sub(b)
	assertData(t, got, []float64{2, 6})
	got, _ = a.Div(b)
	assertData(t, got, []float64{2, 3})
	got, _ = a.Pow(b)
	assertData(t, got, []float64{16, 729})
}

func TestUfuncs(t *testing.T) {
	assertData(t, Array([]float64{1, 4, 9}).Sqrt(), []float64{1, 2, 3})
	assertData(t, Array([]float64{-1, 2}).Abs(), []float64{1, 2})
	assertData(t, Array([]float64{0, 1}).Exp(), []float64{1, math.E})
	assertData(t, Array([]float64{1, math.E}).Log(), []float64{0, 1})
	assertData(t, Array([]float64{1.4, 1.6}).Round(), []float64{1, 2})
	assertData(t, Array([]float64{-5, 0, 5}).Clip(-1, 1), []float64{-1, 0, 1})
}

func TestReductions(t *testing.T) {
	a := MustFromSlice([]float64{1, 2, 3, 4, 5, 6}, 2, 3)
	if got := a.SumAll(); got != 21 {
		t.Errorf("SumAll = %v", got)
	}
	if got := a.MeanAll(); got != 3.5 {
		t.Errorf("MeanAll = %v", got)
	}
	if got := a.MinAll(); got != 1 {
		t.Errorf("MinAll = %v", got)
	}
	if got := a.MaxAll(); got != 6 {
		t.Errorf("MaxAll = %v", got)
	}
	if got := a.StdAll(); !almostEqual(got, 1.707825127659933) {
		t.Errorf("StdAll = %v", got)
	}
	if got := a.VarAll(); !almostEqual(got, 2.9166666666666665) {
		t.Errorf("VarAll = %v", got)
	}

	sum0, err := a.Sum(0)
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, sum0, []float64{5, 7, 9})
	sum1, _ := a.Sum(1)
	assertData(t, sum1, []float64{6, 15})
	mean0, _ := a.Mean(0)
	assertData(t, mean0, []float64{2.5, 3.5, 4.5})
	min1, _ := a.Min(1)
	assertData(t, min1, []float64{1, 4})
	max0, _ := a.Max(0)
	assertData(t, max0, []float64{4, 5, 6})
	if _, err := a.Sum(2); !errors.Is(err, errs.ErrInvalidAxis) {
		t.Errorf("Sum(2) error = %v", err)
	}
	am, _ := a.ArgMax()
	if am.MustAt() != 5 {
		t.Errorf("ArgMax = %v", am)
	}
}

func TestLinalg(t *testing.T) {
	v1 := Array([]float64{1, 2, 3})
	v2 := Array([]float64{4, 5, 6})
	d, err := Dot(v1, v2)
	if err != nil {
		t.Fatal(err)
	}
	if d.MustAt() != 32 {
		t.Errorf("dot = %v", d)
	}
	m1 := MustFromSlice([]float64{1, 2, 3, 4}, 2, 2)
	m2 := MustFromSlice([]float64{5, 6, 7, 8}, 2, 2)
	mm, err := MatMul(m1, m2)
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, mm, []float64{19, 22, 43, 50})
	mv, err := m1.Dot(Array([]float64{1, 1}))
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, mv, []float64{3, 7})
	tr, err := m1.Trace()
	if err != nil || tr != 5 {
		t.Errorf("trace = %v, %v", tr, err)
	}
	if _, err := MatMul(m1, Zeros(3, 3)); !errors.Is(err, errs.ErrShapeMismatch) {
		t.Errorf("bad matmul error = %v", err)
	}
}

func TestComparisonsAndWhere(t *testing.T) {
	a := Array([]float64{1, 5, 3})
	b := Array([]float64{2, 2, 3})
	gt, err := a.Gt(b)
	if err != nil {
		t.Fatal(err)
	}
	if got := gt.Data(); got[0] || !got[1] || got[2] {
		t.Errorf("Gt = %v", got)
	}
	eq, _ := a.Eq(b)
	if got := eq.Data(); got[0] || got[1] || !got[2] {
		t.Errorf("Eq = %v", got)
	}
	w, err := Where(gt, a, b)
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, w, []float64{2, 5, 3})
	mask := a.GtScalar(2)
	if mask.CountTrue() != 2 {
		t.Errorf("GtScalar count = %d", mask.CountTrue())
	}
	c, err := Compress(mask, a)
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, c, []float64{5, 3})
}

func TestRandom(t *testing.T) {
	Seed(42)
	r := Rand(2, 3)
	assertShape(t, r, 2, 3)
	for _, v := range r.Data() {
		if v < 0 || v >= 1 {
			t.Errorf("Rand out of range: %v", v)
		}
	}
	n := Randn(100)
	if n.Size() != 100 {
		t.Errorf("Randn size = %d", n.Size())
	}
}

func TestSqueezeExpandDims(t *testing.T) {
	a := Zeros(1, 3, 1)
	s, err := a.Squeeze()
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, s, 3)
	e, err := s.ExpandDims(0)
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, e, 1, 3)
}

func TestTake(t *testing.T) {
	a := MustFromSlice([]float64{1, 2, 3, 4, 5, 6}, 2, 3)
	got, err := a.Take([]int{2, 0}, 1)
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, got, 2, 2)
	assertData(t, got, []float64{3, 1, 6, 4})
	rows, err := a.Take([]int{1}, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, rows, []float64{4, 5, 6})
}

func TestCopyIndependence(t *testing.T) {
	a := Array([]float64{1, 2, 3})
	b := a.Copy()
	_ = b.Set(9, 0)
	if a.MustAt(0) == 9 {
		t.Error("Copy should not share data")
	}
	if b.IsView() {
		t.Error("Copy should not be a view")
	}
}
