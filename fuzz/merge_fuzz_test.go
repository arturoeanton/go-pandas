package fuzz_test

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

func mergeFixture(t *testing.T, seed int8, n int, withNA bool) (*pd.DataFrame, *pd.DataFrame) {
	t.Helper()
	mod := func(x, m int) int { return ((x % m) + m) % m }
	lids := make([]any, n)
	rids := make([]any, n)
	lv := make([]any, n)
	rv := make([]any, n)
	for i := 0; i < n; i++ {
		if withNA && mod(i+int(seed), 9) == 0 {
			lids[i] = nil
		} else {
			lids[i] = mod(i*int(seed)+i, 8)
		}
		if withNA && mod(i-int(seed), 11) == 0 {
			rids[i] = nil
		} else {
			rids[i] = mod(i+int(seed), 8)
		}
		lv[i] = float64(i)
		rv[i] = "r"
	}
	left, err := pd.DataFrameFromMap(map[string][]any{"id": lids, "lv": lv},
		pd.WithColumnOrder("id", "lv"))
	if err != nil {
		t.Fatal(err)
	}
	right, err := pd.DataFrameFromMap(map[string][]any{"id": rids, "rv": rv},
		pd.WithColumnOrder("id", "rv"))
	if err != nil {
		t.Fatal(err)
	}
	return left, right
}

// FuzzMergeJoinTypes runs every join type over duplicate and NA keys and
// checks row-count invariants plus input immutability.
func FuzzMergeJoinTypes(f *testing.F) {
	f.Add(int8(1), true)
	f.Add(int8(3), false)
	f.Add(int8(-7), true)
	f.Fuzz(func(t *testing.T, seed int8, withNA bool) {
		left, right := mergeFixture(t, seed, 40, withNA)
		lb, rb := left.ToRows(), right.ToRows()

		inner, err := left.Merge(right, pd.MergeOptions{On: []string{"id"}, How: "inner"})
		if err != nil {
			t.Fatal(err)
		}
		lj, err := left.Merge(right, pd.MergeOptions{On: []string{"id"}, How: "left"})
		if err != nil {
			t.Fatal(err)
		}
		rj, err := left.Merge(right, pd.MergeOptions{On: []string{"id"}, How: "right"})
		if err != nil {
			t.Fatal(err)
		}
		outer, err := left.Merge(right, pd.MergeOptions{On: []string{"id"}, How: "outer"})
		if err != nil {
			t.Fatal(err)
		}
		// invariants: left/right joins contain at least their side's rows;
		// inner <= left <= outer
		if lj.Len() < left.Len() {
			t.Fatalf("left join %d < left rows %d", lj.Len(), left.Len())
		}
		if rj.Len() < right.Len() {
			t.Fatalf("right join %d < right rows %d", rj.Len(), right.Len())
		}
		if inner.Len() > lj.Len() || lj.Len() > outer.Len() {
			t.Fatalf("size ordering violated: inner=%d left=%d outer=%d", inner.Len(), lj.Len(), outer.Len())
		}
		// outer = inner + left_only + right_only
		ind, err := left.Merge(right, pd.MergeOptions{On: []string{"id"}, How: "outer", Indicator: true})
		if err != nil {
			t.Fatal(err)
		}
		both, lo, ro := 0, 0, 0
		for _, v := range ind.MustCol("_merge").Values() {
			switch v {
			case "both":
				both++
			case "left_only":
				lo++
			case "right_only":
				ro++
			}
		}
		if both != inner.Len() || both+lo+ro != outer.Len() {
			t.Fatalf("indicator accounting: both=%d lo=%d ro=%d inner=%d outer=%d",
				both, lo, ro, inner.Len(), outer.Len())
		}
		// dtypes preserved
		if outer.StorageDTypes()["lv"] != pd.Float64 || outer.StorageDTypes()["rv"] != pd.String {
			t.Fatalf("merge dtypes = %v", outer.StorageDTypes())
		}
		// inputs untouched
		la, ra := left.ToRows(), right.ToRows()
		for i := range lb {
			for j := range lb[i] {
				if lb[i][j] != la[i][j] {
					t.Fatal("left input mutated")
				}
			}
		}
		for i := range rb {
			for j := range rb[i] {
				if rb[i][j] != ra[i][j] {
					t.Fatal("right input mutated")
				}
			}
		}
	})
}

// FuzzMergeMultiKeyValidate checks composite keys and cardinality
// validation.
func FuzzMergeMultiKeyValidate(f *testing.F) {
	f.Add(int8(2))
	f.Add(int8(5))
	f.Fuzz(func(t *testing.T, seed int8) {
		mod := func(x, m int) int { return ((x % m) + m) % m }
		n := 30
		a := make([]any, n)
		b := make([]any, n)
		v := make([]any, n)
		for i := 0; i < n; i++ {
			a[i] = []string{"x", "y", "z"}[mod(i+int(seed), 3)]
			b[i] = mod(i*7+int(seed), 4)
			v[i] = float64(i)
		}
		left, _ := pd.DataFrameFromMap(map[string][]any{"a": a, "b": b, "v": v},
			pd.WithColumnOrder("a", "b", "v"))
		right, _ := pd.DataFrameFromMap(map[string][]any{"a": a[:12], "b": b[:12], "w": v[:12]},
			pd.WithColumnOrder("a", "b", "w"))
		out, err := left.Merge(right, pd.MergeOptions{On: []string{"a", "b"}, How: "inner"})
		if err != nil {
			t.Fatal(err)
		}
		if out.Len() < 12 { // every right row has at least one left match
			t.Fatalf("multi-key inner rows = %d", out.Len())
		}
		// validation must reject duplicate-key cardinality
		if _, err := left.Merge(right, pd.MergeOptions{On: []string{"a", "b"}, Validate: "one_to_one"}); err == nil {
			t.Fatal("one_to_one should fail with duplicate keys")
		}
	})
}
