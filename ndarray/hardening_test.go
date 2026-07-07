package ndarray

import (
	"errors"
	"testing"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// TestOperationsDoNotMutateInputs locks in the copy semantics of every
// value-producing operation.
func TestOperationsDoNotMutateInputs(t *testing.T) {
	snapshot := func() (*NDArray, *NDArray) {
		return Array([]float64{3, 1, 2}), Array([]float64{10, 20, 30})
	}
	check := func(name string, a, b *NDArray) {
		t.Helper()
		assertData(t, a, []float64{3, 1, 2})
		assertData(t, b, []float64{10, 20, 30})
		_ = name
	}

	a, b := snapshot()
	if _, err := a.Add(b); err != nil {
		t.Fatal(err)
	}
	check("Add", a, b)
	if _, err := a.Sub(b); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Mul(b); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Div(b); err != nil {
		t.Fatal(err)
	}
	check("arith", a, b)

	_ = a.Sort()
	_ = a.ArgSort()
	_ = Unique(a)
	_ = a.Flatten()
	_ = a.AddScalar(5)
	_ = a.Sqrt()
	check("unary", a, b)

	if _, err := a.Astype(dtype.Int64); err != nil {
		t.Fatal(err)
	}
	check("astype", a, b)

	mask := a.GtScalar(1)
	if _, err := a.Mask(mask); err != nil {
		t.Fatal(err)
	}
	if _, err := Where(mask, a, b); err != nil {
		t.Fatal(err)
	}
	if _, err := WhereScalar(mask, a, 0); err != nil {
		t.Fatal(err)
	}
	check("mask/where", a, b)

	// broadcasted op against a view
	m := MustFromSlice([]float64{1, 2, 3, 4, 5, 6}, 2, 3)
	row := Array([]float64{10, 20, 30})
	if _, err := m.Add(row); err != nil {
		t.Fatal(err)
	}
	assertData(t, m, []float64{1, 2, 3, 4, 5, 6})
	assertData(t, row, []float64{10, 20, 30})
}

// TestFlattenIndependence is the regression test for the v0.2.0 buffer
// aliasing bug: mutating a Flatten() result must not touch the source.
func TestFlattenIndependence(t *testing.T) {
	a := Array([]float64{1, 2, 3})
	f := a.Flatten()
	if err := f.Set(99, 0); err != nil {
		t.Fatal(err)
	}
	if a.MustAt(0) != 1 {
		t.Fatalf("Flatten shares the source buffer: source[0] = %v", a.MustAt(0))
	}
	// Ravel of a contiguous array IS a view by contract.
	r := a.Ravel()
	if !r.IsView() {
		t.Error("Ravel of a contiguous array should be a view")
	}
}

func TestEdgeShapes(t *testing.T) {
	// empty (0,)
	empty := Array(nil)
	assertShape(t, empty, 0)
	if empty.SumAll() != 0 {
		t.Errorf("empty sum = %v", empty.SumAll())
	}
	if got := empty.Sort(); got.Size() != 0 {
		t.Errorf("empty sort size = %d", got.Size())
	}
	if got := Unique(empty); got.Size() != 0 {
		t.Errorf("empty unique size = %d", got.Size())
	}
	sum, err := empty.Add(Array(nil))
	if err != nil || sum.Size() != 0 {
		t.Errorf("empty add = %v, %v", sum, err)
	}

	// (1,) broadcasts against anything
	one := Array([]float64{5})
	m := MustFromSlice([]float64{1, 2, 3, 4, 5, 6}, 2, 3)
	out, err := m.Add(one)
	if err != nil {
		t.Fatal(err)
	}
	assertData(t, out, []float64{6, 7, 8, 9, 10, 11})

	// (1,1) + (2,3)
	oneone := MustFromSlice([]float64{10}, 1, 1)
	out, err = m.Add(oneone)
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, out, 2, 3)

	// (2,1) + (1,2) -> (2,2)
	col := MustFromSlice([]float64{1, 2}, 2, 1)
	row := MustFromSlice([]float64{10, 20}, 1, 2)
	out, err = col.Add(row)
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, out, 2, 2)
	assertData(t, out, []float64{11, 21, 12, 22})

	// incompatible
	if _, err := MustFromSlice([]float64{1, 2}, 2, 1).Add(Zeros(3, 2)); !errors.Is(err, errs.ErrBroadcastMismatch) {
		t.Errorf("(2,1)+(3,2) error = %v", err)
	}

	// reductions on (1,1)
	s, err := oneone.Sum(0)
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, s, 1)
	assertData(t, s, []float64{10})
}

// TestViewWriteThrough documents the view contract explicitly: slices,
// reshapes of contiguous arrays and transposes share the buffer.
func TestViewWriteThrough(t *testing.T) {
	base := Arange(6)
	view, err := base.Reshape(2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := view.Set(99, 0, 0); err != nil {
		t.Fatal(err)
	}
	if base.MustAt(0) != 99 {
		t.Error("reshape view should write through")
	}
	tr, _ := view.T()
	if err := tr.Set(-1, 2, 1); err != nil { // tr[2,1] == view[1,2]
		t.Fatal(err)
	}
	if view.MustAt(1, 2) != -1 {
		t.Error("transpose view should write through")
	}
	// Copy detaches.
	c := view.Copy()
	_ = c.Set(1234, 0, 0)
	if base.MustAt(0) == 1234 {
		t.Error("Copy must not write through")
	}
}
