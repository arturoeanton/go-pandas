package dataframe

import (
	"testing"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/index"
)

// typedFrame has one column per typed backing.
func typedFrame(t *testing.T) *DataFrame {
	t.Helper()
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	df, err := DataFrameFromMap(map[string][]any{
		"age":        {31, 30, 20, 40, nil},
		"salary":     {1000.0, 800.0, 700.0, nil, 900.0},
		"name":       {"Ana", "Bia", nil, "Dan", "Eva"},
		"active":     {true, false, true, true, nil},
		"created_at": {t0, t0.AddDate(0, 1, 0), t0.AddDate(0, 2, 0), nil, t0},
	}, WithColumnOrder("age", "salary", "name", "active", "created_at"))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

var wantStorage = map[string]dtype.DType{
	"age": dtype.Int, "salary": dtype.Float64, "name": dtype.String,
	"active": dtype.Bool, "created_at": dtype.Time,
}

func assertTypedStorage(t *testing.T, df *DataFrame, context string) {
	t.Helper()
	storage := df.StorageDTypes()
	for name, want := range wantStorage {
		if storage[name] != want {
			t.Errorf("%s: column %s storage = %v, want %v", context, name, storage[name], want)
		}
	}
}

func TestWherePreservesTypedStorage(t *testing.T) {
	df := typedFrame(t)
	// selects rows 0, 1, 3 — an irregular position pattern
	out, err := df.Where(expr.Col("age").Gt(29))
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 3 {
		t.Fatalf("filtered rows = %d", out.Len())
	}
	assertTypedStorage(t, out, "Where")
	// masks survive the gather: salary at original row 3 was NA
	if v, _ := out.MustCol("salary").At(2); v != nil {
		t.Errorf("mask lost through Where: %v", v)
	}
	// an irregular selection over a RangeIndex becomes a typed
	// Int64Index, labels preserved
	if _, ok := out.Index().(*index.Int64Index); !ok {
		t.Errorf("filtered index type = %T, want *index.Int64Index", out.Index())
	}
	if out.Index().At(2) != 3 {
		t.Errorf("filtered label = %v, want 3", out.Index().At(2))
	}
}

func TestTakePreservesTypedStorage(t *testing.T) {
	df := typedFrame(t)
	out, err := df.Take([]int{4, 0, 2})
	if err != nil {
		t.Fatal(err)
	}
	assertTypedStorage(t, out, "Take")
	if v, _ := out.MustCol("name").At(2); v != nil {
		t.Errorf("string NA lost: %v", v)
	}
	if v, _ := out.MustCol("age").At(1); v != 31 {
		t.Errorf("take order: %v", v)
	}
}

func TestSlicePreservesTypedStorage(t *testing.T) {
	df := typedFrame(t)
	out, err := df.Slice(1, 4)
	if err != nil {
		t.Fatal(err)
	}
	assertTypedStorage(t, out, "Slice")
	// contiguous slice of a RangeIndex stays a RangeIndex
	if _, ok := out.Index().(*index.RangeIndex); !ok {
		t.Errorf("sliced index type = %T, want *index.RangeIndex", out.Index())
	}
	if out.Index().At(0) != 1 {
		t.Errorf("sliced label = %v, want 1", out.Index().At(0))
	}
	assertTypedStorage(t, df.Head(2), "Head")
	assertTypedStorage(t, df.Tail(2), "Tail")
	assertTypedStorage(t, df.DropNA(), "DropNA")
}

func TestSeriesTakePreservesTypedStorage(t *testing.T) {
	df := typedFrame(t)
	for name, want := range wantStorage {
		s := df.MustCol(name)
		taken, err := s.Take([]int{2, 0})
		if err != nil {
			t.Fatal(err)
		}
		if taken.StorageDType() != want || taken.IsObjectBacked() {
			t.Errorf("series %s take storage = %v", name, taken.StorageDType())
		}
		sliced, err := s.Slice(1, 3)
		if err != nil {
			t.Fatal(err)
		}
		if sliced.StorageDType() != want {
			t.Errorf("series %s slice storage = %v", name, sliced.StorageDType())
		}
	}
	// negative take positions become NA without degrading dtype
	withNA, err := df.MustCol("age").Take([]int{0, -1})
	if err != nil {
		t.Fatal(err)
	}
	if withNA.StorageDType() != dtype.Int {
		t.Errorf("negative take storage = %v", withNA.StorageDType())
	}
	if v, _ := withNA.At(1); v != nil {
		t.Errorf("negative take value = %v", v)
	}
}

func TestGatherImmutability(t *testing.T) {
	df := typedFrame(t)
	before := df.ToRows()
	beforeIdx := df.Index().Values()

	out, err := df.Take([]int{0, 1})
	if err != nil {
		t.Fatal(err)
	}
	// mutate every cell of the output
	for _, name := range out.Columns() {
		_ = out.MustCol(name).Set(0, nil)
	}
	filtered, err := df.Where(expr.Col("age").Gt(0))
	if err != nil {
		t.Fatal(err)
	}
	_ = filtered.MustCol("age").Set(0, 999)

	after := df.ToRows()
	for i := range before {
		for j := range before[i] {
			if before[i][j] != after[i][j] {
				t.Fatalf("input mutated at [%d][%d]: %v -> %v", i, j, before[i][j], after[i][j])
			}
		}
	}
	afterIdx := df.Index().Values()
	for i := range beforeIdx {
		if beforeIdx[i] != afterIdx[i] {
			t.Fatalf("input index mutated at %d", i)
		}
	}
}

func TestIndexTakeTyped(t *testing.T) {
	r := index.NewRangeIndex(10)
	// contiguous -> RangeIndex
	if got := index.Take(r, []int{3, 4, 5}); got.At(0) != 3 {
		t.Errorf("contiguous take label = %v", got.At(0))
	} else if _, ok := got.(*index.RangeIndex); !ok {
		t.Errorf("contiguous take type = %T", got)
	}
	// constant step -> RangeIndex
	stepped := index.Take(r, []int{1, 3, 5})
	if _, ok := stepped.(*index.RangeIndex); !ok {
		t.Errorf("stepped take type = %T", stepped)
	}
	if stepped.Len() != 3 || stepped.At(2) != 5 {
		t.Errorf("stepped take = %v (len %d)", stepped.At(2), stepped.Len())
	}
	// irregular -> Int64Index with int labels
	irregular := index.Take(r, []int{0, 4, 5})
	i64, ok := irregular.(*index.Int64Index)
	if !ok {
		t.Fatalf("irregular take type = %T", irregular)
	}
	if i64.At(1) != 4 {
		t.Errorf("irregular label = %v (%T)", i64.At(1), i64.At(1))
	}
	if p, ok := i64.Pos(5); !ok || p != 2 {
		t.Errorf("Int64Index Pos = %d, %v", p, ok)
	}
	// string index stays typed
	s := index.NewStringIndex([]string{"a", "b", "c"})
	st := index.Take(s, []int{2, 0})
	if _, ok := st.(*index.StringIndex); !ok {
		t.Errorf("string take type = %T", st)
	}
	if st.At(0) != "c" {
		t.Errorf("string take label = %v", st.At(0))
	}
	// negative positions fall back to the boxed index with nil labels
	neg := index.Take(r, []int{0, -1})
	if neg.At(1) != nil {
		t.Errorf("negative take label = %v", neg.At(1))
	}
}
