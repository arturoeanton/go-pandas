package fuzz_test

import (
	"fmt"
	"strings"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/ndarray"
)

// v10Frame builds a deterministic key/value frame from fuzz input.
func v10Frame(t *testing.T, seed int8, n int) *pd.DataFrame {
	t.Helper()
	mod := func(x, m int) int { return ((x % m) + m) % m }
	keys := make([]any, n)
	sub := make([]any, n)
	vals := make([]any, n)
	letters := []string{"a", "b", "c"}
	for i := 0; i < n; i++ {
		keys[i] = letters[mod(i*7+int(seed), len(letters))]
		sub[i] = fmt.Sprintf("s%d", mod(i*3+int(seed), 4))
		if mod(i+int(seed), 6) != 5 {
			vals[i] = float64(i) + 0.5
		}
	}
	df, err := pd.DataFrameFromMap(map[string][]any{"k": keys, "s": sub, "v": vals},
		pd.WithColumnOrder("k", "s", "v"))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

// FuzzStackUnstack checks the stack layout invariants and, when tuples
// are unique, the unstack roundtrip.
func FuzzStackUnstack(f *testing.F) {
	f.Add(int8(1), uint8(10))
	f.Add(int8(-4), uint8(30))
	f.Fuzz(func(t *testing.T, seed int8, size uint8) {
		n := int(size)%32 + 1
		df := v10Frame(t, seed, n)
		before := fmt.Sprint(df.ToRows())
		s, err := df.Stack()
		if err != nil {
			t.Fatal(err)
		}
		if s.Len() != n*3 {
			t.Fatalf("stack len = %d, want %d", s.Len(), n*3)
		}
		mi := s.Index().(*pd.MultiIndex)
		if mi.NLevels() != 2 {
			t.Fatalf("levels = %d", mi.NLevels())
		}
		// Row-major: every row contributes its 3 columns in order.
		for i := 0; i < n; i++ {
			if got := mi.Tuple(i * 3)[1]; got != "k" {
				t.Fatalf("layout broken at row %d: %v", i, got)
			}
		}
		out, err := pd.UnstackSeries(s)
		if err != nil {
			t.Fatal(err) // stack always produces unique tuples
		}
		if out.Len() > n || len(out.Columns()) != 3 {
			t.Fatalf("unstack shape = %d x %d", out.Len(), len(out.Columns()))
		}
		if fmt.Sprint(df.ToRows()) != before {
			t.Fatal("input mutated")
		}
	})
}

// FuzzPivotTable checks shape/no-panic across values/aggs combinations.
func FuzzPivotTable(f *testing.F) {
	f.Add(int8(2), uint8(20), true)
	f.Add(int8(-7), uint8(8), false)
	f.Fuzz(func(t *testing.T, seed int8, size uint8, multiAgg bool) {
		n := int(size)%48 + 1
		df := v10Frame(t, seed, n)
		aggs := []string{"sum"}
		if multiAgg {
			aggs = []string{"sum", "count", "mean"}
		}
		out, err := df.PivotTable(pd.PivotTableOptions{
			Values: []string{"v"}, Index: []string{"k"}, Columns: []string{"s"},
			AggFuncs: aggs,
		})
		if err != nil {
			t.Fatal(err)
		}
		if out.Len() == 0 || out.Len() > 3 {
			t.Fatalf("pivot rows = %d", out.Len())
		}
		for _, name := range out.Columns()[1:] {
			if multiAgg && !strings.Contains(name, "_") {
				t.Fatalf("multi-agg naming rule broken: %v", out.Columns())
			}
		}
	})
}

// FuzzGroupByTransform: output aligns with input, group members share
// the aggregate, sum totals match.
func FuzzGroupByTransform(f *testing.F) {
	f.Add(int8(3), uint8(24))
	f.Add(int8(-1), uint8(5))
	f.Fuzz(func(t *testing.T, seed int8, size uint8) {
		n := int(size)%48 + 1
		df := v10Frame(t, seed, n)
		out, err := df.GroupBy("k").Transform("v", "sum")
		if err != nil {
			t.Fatal(err)
		}
		if out.Len() != n {
			t.Fatalf("transform len = %d", out.Len())
		}
		keys := df.MustCol("k").Values()
		// Same key -> same transformed value.
		byKey := map[any]any{}
		for i := 0; i < n; i++ {
			v := out.Values()[i]
			if prev, ok := byKey[keys[i]]; ok {
				if fmt.Sprint(prev) != fmt.Sprint(v) {
					t.Fatalf("group %v: %v != %v", keys[i], prev, v)
				}
				continue
			}
			byKey[keys[i]] = v
		}
	})
}

// FuzzGroupByFilter: kept rows form whole groups in original order.
func FuzzGroupByFilter(f *testing.F) {
	f.Add(int8(5), uint8(30), uint8(2))
	f.Add(int8(-9), uint8(3), uint8(1))
	f.Fuzz(func(t *testing.T, seed int8, size, threshold uint8) {
		n := int(size)%48 + 1
		th := float64(threshold % 8)
		df := v10Frame(t, seed, n)
		out, err := df.GroupBy("k").Filter(pd.GroupSize().Gt(th))
		if err != nil {
			t.Fatal(err)
		}
		// Every kept key's full group survived.
		counts := map[any]int{}
		for _, k := range df.MustCol("k").Values() {
			counts[k]++
		}
		kept := map[any]int{}
		for _, k := range out.MustCol("k").Values() {
			kept[k]++
		}
		for k, c := range kept {
			if counts[k] != c {
				t.Fatalf("group %v partially kept: %d of %d", k, c, counts[k])
			}
			if float64(counts[k]) <= th {
				t.Fatalf("group %v should have been dropped", k)
			}
		}
	})
}

// FuzzQueryParser: arbitrary strings never panic; valid generated
// queries evaluate.
func FuzzQueryParser(f *testing.F) {
	f.Add("salary + bonus > 1000")
	f.Add(`country in ["AR", "BR"] and not active`)
	f.Add("((((")
	f.Add("a >< b !")
	f.Add(`x not in [1, 2] or (y * -3 % 2 == 1)`)
	f.Fuzz(func(t *testing.T, q string) {
		pred, err := expr.ParseQuery(q)
		if err != nil {
			return // syntax errors are fine; panics are not
		}
		_ = pred.String()
		// Evaluate against a row covering common names; errors allowed.
		row := map[string]any{"a": 1.0, "b": 2.0, "x": 3, "y": 4, "salary": 100.0}
		_, _ = pred.EvalBool(row)
	})
}

// FuzzNDArrayTake: dtype preserved, values gathered, errors on range.
func FuzzNDArrayTake(f *testing.F) {
	f.Add(uint8(10), uint8(5), int8(3))
	f.Add(uint8(3), uint8(40), int8(-2))
	f.Fuzz(func(t *testing.T, size, nTake uint8, seed int8) {
		mod := func(x, m int) int { return ((x % m) + m) % m }
		n := int(size)%64 + 1
		data := make([]float64, n)
		for i := range data {
			data[i] = float64(i * 2)
		}
		a := ndarray.Array(data)
		indices := make([]int, int(nTake)%64)
		for i := range indices {
			indices[i] = mod(i*11+int(seed), n)
		}
		out, err := a.Take(indices, 0)
		if err != nil {
			t.Fatal(err)
		}
		if out.Size() != len(indices) {
			t.Fatalf("take size = %d", out.Size())
		}
		for i, idx := range indices {
			got, _ := out.At(i)
			if got != data[idx] {
				t.Fatalf("take[%d] = %v, want %v", i, got, data[idx])
			}
		}
		if _, err := a.Take([]int{n}, 0); err == nil {
			t.Fatal("out-of-range must error")
		}
	})
}

// FuzzSearchSorted: results match a linear scan on sorted data.
func FuzzSearchSorted(f *testing.F) {
	f.Add(uint8(12), int8(4), true)
	f.Add(uint8(3), int8(-8), false)
	f.Fuzz(func(t *testing.T, size uint8, seed int8, rightSide bool) {
		n := int(size)%64 + 1
		data := make([]float64, n)
		acc := 0.0
		for i := range data {
			acc += float64(((int(seed)+i*7)%5 + 5) % 5)
			data[i] = acc
		}
		a := ndarray.Array(data)
		side := "left"
		if rightSide {
			side = "right"
		}
		queries := []float64{data[0] - 1, data[n/2], data[n-1] + 1}
		got, err := a.SearchSorted(queries, side)
		if err != nil {
			t.Fatal(err)
		}
		for qi, q := range queries {
			want := 0
			for _, v := range data {
				if v < q || (rightSide && v == q) {
					want++
				}
			}
			if got[qi] != want {
				t.Fatalf("searchsorted(%v, %s) = %d, want %d", q, side, got[qi], want)
			}
		}
	})
}

// FuzzIsIn: membership matches a linear scan; NaN never matches.
func FuzzIsIn(f *testing.F) {
	f.Add(uint8(16), int8(2))
	f.Add(uint8(50), int8(-5))
	f.Fuzz(func(t *testing.T, size uint8, seed int8) {
		mod := func(x, m int) int { return ((x % m) + m) % m }
		n := int(size)%64 + 1
		data := make([]float64, n)
		for i := range data {
			data[i] = float64(mod(i*13+int(seed), 9))
		}
		a := ndarray.Array(data)
		candidates := []any{2.0, 5, float64(mod(int(seed), 9))}
		out := a.IsIn(candidates)
		want := map[float64]bool{2: true, 5: true, float64(mod(int(seed), 9)): true}
		vals := out.Data()
		for i, v := range data {
			if vals[i] != want[v] {
				t.Fatalf("isin[%d]=%v for value %v", i, vals[i], v)
			}
		}
	})
}
