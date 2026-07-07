package fuzz_test

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

// FuzzTakeGather stresses DataFrame/Series/Index gathering with
// arbitrary positions: no panics, exact lengths, preserved dtypes and
// untouched inputs.
func FuzzTakeGather(f *testing.F) {
	f.Add(0, 1, 2, 3)
	f.Add(3, 3, 3, 3)
	f.Add(4, 0, 4, 0)
	f.Add(-1, 0, 100, 2)
	f.Fuzz(func(t *testing.T, p0, p1, p2, p3 int) {
		df, err := pd.DataFrameFromMap(map[string][]any{
			"i": {10, 20, 30, nil, 50},
			"s": {"a", "b", nil, "d", "e"},
			"f": {1.5, nil, 3.5, 4.5, 5.5},
		}, pd.WithColumnOrder("i", "s", "f"))
		if err != nil {
			t.Fatal(err)
		}
		before := df.ToRows()
		pos := []int{p0, p1, p2, p3}
		out, err := df.Take(pos)
		if err != nil {
			// out-of-range positions must error, never panic
			return
		}
		if out.Len() != len(pos) {
			t.Fatalf("take length = %d, want %d", out.Len(), len(pos))
		}
		storage := out.StorageDTypes()
		if storage["i"] != pd.Int || storage["s"] != pd.String || storage["f"] != pd.Float64 {
			t.Fatalf("dtypes degraded: %v", storage)
		}
		// negative positions must be masked
		for k, p := range pos {
			if p < 0 {
				if v, _ := out.MustCol("i").At(k); v != nil {
					t.Fatalf("negative position produced %v", v)
				}
			}
		}
		// input untouched
		after := df.ToRows()
		for i := range before {
			for j := range before[i] {
				if before[i][j] != after[i][j] {
					t.Fatal("input mutated by Take")
				}
			}
		}
	})
}

// FuzzWhereMask stresses filtering with arbitrary thresholds over data
// containing NAs.
func FuzzWhereMask(f *testing.F) {
	f.Add(25.0, int8(3))
	f.Add(-1e9, int8(0))
	f.Add(1e9, int8(7))
	f.Fuzz(func(t *testing.T, threshold float64, seed int8) {
		values := make([]any, 50)
		for i := range values {
			switch (i + int(seed)) % 5 {
			case 0:
				values[i] = nil
			default:
				values[i] = float64(i * int(seed))
			}
		}
		df, err := pd.DataFrameFromMap(map[string][]any{"v": values})
		if err != nil {
			t.Fatal(err)
		}
		out, err := df.Where(pd.Col("v").Gt(threshold))
		if err != nil {
			t.Fatal(err)
		}
		if out.Len() > df.Len() {
			t.Fatalf("filter grew the frame: %d > %d", out.Len(), df.Len())
		}
		// every selected value must actually satisfy the predicate
		for _, v := range out.MustCol("v").Values() {
			if v == nil {
				t.Fatal("NA row selected")
			}
			if v.(float64) <= threshold {
				t.Fatalf("selected %v <= %v", v, threshold)
			}
		}
	})
}
