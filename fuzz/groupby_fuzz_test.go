package fuzz_test

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

func groupFixture(t *testing.T, seed int8, withNA bool) *pd.DataFrame {
	t.Helper()
	n := 60
	keys := make([]any, n)
	nums := make([]any, n)
	vals := make([]any, n)
	letters := []string{"a", "b", "c", "d"}
	mod := func(x, m int) int { return ((x % m) + m) % m }
	for i := 0; i < n; i++ {
		if withNA && mod(i+int(seed), 7) == 0 {
			keys[i] = nil
		} else {
			keys[i] = letters[mod(i+int(seed), len(letters))]
		}
		nums[i] = mod(i*int(seed), 5)
		if mod(i+int(seed), 5) == 0 {
			vals[i] = nil
		} else {
			vals[i] = float64(i)
		}
	}
	df, err := pd.DataFrameFromMap(map[string][]any{
		"k": keys, "n": nums, "v": vals,
	}, pd.WithColumnOrder("k", "n", "v"))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

// FuzzGroupBySingleKey checks the core invariants: group count bounded by
// rows, sizes sum to the kept row count, count <= size, input untouched.
func FuzzGroupBySingleKey(f *testing.F) {
	f.Add(int8(1), true)
	f.Add(int8(3), false)
	f.Add(int8(-5), true)
	f.Fuzz(func(t *testing.T, seed int8, dropNA bool) {
		df := groupFixture(t, seed, true)
		before := df.ToRows()
		gb := df.GroupByOpts([]pd.GroupByOption{pd.GroupDropNA(dropNA)}, "k")
		size, err := gb.Size()
		if err != nil {
			t.Fatal(err)
		}
		if size.Len() > df.Len() {
			t.Fatalf("more groups than rows: %d", size.Len())
		}
		total := 0
		naKeyRows := 0
		for _, v := range df.MustCol("k").Values() {
			if v == nil {
				naKeyRows++
			}
		}
		for _, v := range size.MustCol("size").Values() {
			total += v.(int)
		}
		want := df.Len()
		if dropNA {
			want -= naKeyRows
		}
		if total != want {
			t.Fatalf("sizes sum to %d, want %d", total, want)
		}
		count, err := df.GroupByOpts([]pd.GroupByOption{pd.GroupDropNA(dropNA)}, "k").Count("v")
		if err != nil {
			t.Fatal(err)
		}
		sizes := size.MustCol("size").Values()
		counts := count.MustCol("v").Values()
		for i := range counts {
			if counts[i].(int) > sizes[i].(int) {
				t.Fatalf("count %v > size %v", counts[i], sizes[i])
			}
		}
		after := df.ToRows()
		for i := range before {
			for j := range before[i] {
				if before[i][j] != after[i][j] {
					t.Fatal("groupby mutated input")
				}
			}
		}
	})
}

// FuzzGroupByMultiKeyAgg checks multi-key grouping plus mixed
// aggregations.
func FuzzGroupByMultiKeyAgg(f *testing.F) {
	f.Add(int8(2))
	f.Add(int8(9))
	f.Fuzz(func(t *testing.T, seed int8) {
		df := groupFixture(t, seed, true)
		out, err := df.GroupBy("k", "n").AggList(map[string][]string{
			"v": {"sum", "mean", "min", "max", "nunique"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if out.Len() > df.Len() {
			t.Fatalf("groups %d > rows %d", out.Len(), df.Len())
		}
		storage := out.StorageDTypes()
		if storage["v_sum"] != pd.Float64 || storage["v_nunique"] != pd.Int {
			t.Fatalf("agg dtypes = %v", storage)
		}
		// min <= max wherever both present
		mins := out.MustCol("v_min").Values()
		maxs := out.MustCol("v_max").Values()
		for i := range mins {
			if mins[i] == nil || maxs[i] == nil {
				continue
			}
			if mins[i].(float64) > maxs[i].(float64) {
				t.Fatalf("min %v > max %v", mins[i], maxs[i])
			}
		}
	})
}
