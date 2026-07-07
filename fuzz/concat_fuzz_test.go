package fuzz_test

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

// FuzzConcatAxis0 stresses vertical concat with mixed schemas, NAs and
// numeric promotion.
func FuzzConcatAxis0(f *testing.F) {
	f.Add(int8(1), true, true)
	f.Add(int8(4), false, true)
	f.Add(int8(-3), true, false)
	f.Fuzz(func(t *testing.T, seed int8, extraCol, promote bool) {
		mod := func(x, m int) int { return ((x % m) + m) % m }
		mk := func(n int, float bool, extra bool) *pd.DataFrame {
			v := make([]any, n)
			for i := 0; i < n; i++ {
				if mod(i+int(seed), 6) == 0 {
					continue // nil
				}
				if float {
					v[i] = float64(i) + 0.5
				} else {
					v[i] = i
				}
			}
			cols := map[string][]any{"v": v}
			order := []string{"v"}
			if extra {
				e := make([]any, n)
				for i := range e {
					e[i] = "s"
				}
				cols["e"] = e
				order = append(order, "e")
			}
			df, err := pd.DataFrameFromMap(cols, pd.WithColumnOrder(order...))
			if err != nil {
				t.Fatal(err)
			}
			return df
		}
		a := mk(20+mod(int(seed), 10), false, true)
		b := mk(15, promote, extraCol)
		ab, bb := a.ToRows(), b.ToRows()

		out, err := pd.Concat([]*pd.DataFrame{a, b}, pd.IgnoreIndex(true))
		if err != nil {
			t.Fatal(err)
		}
		if out.Len() != a.Len()+b.Len() {
			t.Fatalf("rows = %d, want %d", out.Len(), a.Len()+b.Len())
		}
		wantV := pd.Int
		if promote {
			wantV = pd.Float64
		}
		if got := out.StorageDTypes()["v"]; got != wantV {
			t.Fatalf("v storage = %v, want %v", got, wantV)
		}
		// masks: every source nil must stay NA
		vOut := out.MustCol("v").Values()
		for i, row := range ab {
			if row[0] == nil && vOut[i] != nil {
				t.Fatalf("NA lost at %d: %v", i, vOut[i])
			}
		}
		// inputs untouched
		aa, ba := a.ToRows(), b.ToRows()
		for i := range ab {
			for j := range ab[i] {
				if ab[i][j] != aa[i][j] {
					t.Fatal("concat mutated first input")
				}
			}
		}
		for i := range bb {
			for j := range bb[i] {
				if bb[i][j] != ba[i][j] {
					t.Fatal("concat mutated second input")
				}
			}
		}
	})
}

// FuzzConcatAxis1 checks horizontal assembly and duplicate-name handling.
func FuzzConcatAxis1(f *testing.F) {
	f.Add(5, int8(1))
	f.Add(1, int8(9))
	f.Fuzz(func(t *testing.T, n int, seed int8) {
		if n < 1 || n > 200 {
			return
		}
		v := make([]any, n)
		for i := range v {
			v[i] = float64(i * int(seed))
		}
		a, _ := pd.DataFrameFromMap(map[string][]any{"v": v})
		b, _ := pd.DataFrameFromMap(map[string][]any{"v": v, "w": v}, pd.WithColumnOrder("v", "w"))
		out, err := pd.Concat([]*pd.DataFrame{a, b}, pd.ConcatAxis(1))
		if err != nil {
			t.Fatal(err)
		}
		if out.Len() != n {
			t.Fatalf("axis1 rows = %d", out.Len())
		}
		if len(out.Columns()) != 3 {
			t.Fatalf("axis1 columns = %v", out.Columns())
		}
		for _, name := range out.Columns() {
			if out.MustCol(name).IsObjectBacked() {
				t.Fatalf("axis1 column %s object-backed", name)
			}
		}
	})
}
