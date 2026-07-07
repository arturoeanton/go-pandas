package fuzz_test

import (
	"fmt"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

// catLabels derives a deterministic label list with NAs from fuzz input.
func catLabels(seed int8, n int) []any {
	mod := func(x, m int) int { return ((x % m) + m) % m }
	letters := []string{"a", "b", "c", "d", "e"}
	out := make([]any, n)
	for i := 0; i < n; i++ {
		k := mod(i*7+int(seed), 11)
		if k == 10 {
			continue // nil (NA)
		}
		out[i] = letters[mod(k, len(letters))]
	}
	return out
}

// FuzzCategoricalFactorize checks the codes/categories invariants: codes
// index the sorted distinct label set, -1 iff NA, and decoding restores
// the input.
func FuzzCategoricalFactorize(f *testing.F) {
	f.Add(int8(1), uint8(20))
	f.Add(int8(-7), uint8(3))
	f.Add(int8(50), uint8(60))
	f.Fuzz(func(t *testing.T, seed int8, size uint8) {
		values := catLabels(seed, int(size)%64+1)
		s, err := pd.NewCategoricalSeries("v", values)
		if err != nil {
			t.Fatal(err)
		}
		cat, err := s.Cat()
		if err != nil {
			t.Fatal(err)
		}
		cats, codes := cat.Categories(), cat.Codes()
		for i := 1; i < len(cats); i++ {
			if cats[i-1].(string) >= cats[i].(string) {
				t.Fatalf("categories not sorted/unique: %v", cats)
			}
		}
		for i, v := range values {
			if v == nil {
				if codes[i] != -1 {
					t.Fatalf("NA row %d has code %d", i, codes[i])
				}
				continue
			}
			if codes[i] < 0 || int(codes[i]) >= len(cats) {
				t.Fatalf("code out of range at %d: %d", i, codes[i])
			}
			if cats[codes[i]] != v {
				t.Fatalf("decode mismatch at %d: %v != %v", i, cats[codes[i]], v)
			}
		}
	})
}

// FuzzCategoricalAstypeRoundtrip converts string -> category -> string
// and requires exact value/NA preservation.
func FuzzCategoricalAstypeRoundtrip(f *testing.F) {
	f.Add(int8(0), uint8(10))
	f.Add(int8(9), uint8(40))
	f.Fuzz(func(t *testing.T, seed int8, size uint8) {
		values := catLabels(seed, int(size)%64+1)
		s := pd.NewSeries("v", values)
		cat, err := s.Astype(pd.Category)
		if err != nil {
			t.Fatal(err)
		}
		back, err := cat.Astype(pd.String)
		if err != nil {
			t.Fatal(err)
		}
		for i, v := range values {
			if back.Values()[i] != v {
				t.Fatalf("round trip mismatch at %d: %v != %v", i, back.Values()[i], v)
			}
		}
	})
}

// FuzzCategoricalGroupByEquivalence requires the categorical code fast
// path to produce exactly the string engine's aggregation.
func FuzzCategoricalGroupByEquivalence(f *testing.F) {
	f.Add(int8(3), uint8(30))
	f.Add(int8(-1), uint8(7))
	f.Fuzz(func(t *testing.T, seed int8, size uint8) {
		n := int(size)%64 + 2
		labels := catLabels(seed, n)
		nums := make([]float64, n)
		for i := range nums {
			nums[i] = float64((i*int(seed))%13) + 0.5
		}
		asStr, _ := pd.NewDataFrame(pd.NewSeries("k", labels), pd.FloatSeries("v", nums))
		kc, err := pd.NewSeries("k", labels).Astype(pd.Category)
		if err != nil {
			t.Fatal(err)
		}
		asCat, _ := pd.NewDataFrame(kc, pd.FloatSeries("v", nums))

		gs, err := asStr.GroupBy("k").Mean()
		if err != nil {
			t.Fatal(err)
		}
		gc, err := asCat.GroupBy("k").Mean()
		if err != nil {
			t.Fatal(err)
		}
		if gs.Len() != gc.Len() {
			t.Fatalf("group counts differ: %d vs %d", gs.Len(), gc.Len())
		}
		sr, cr := gs.ToRows(), gc.ToRows()
		for i := range sr {
			if fmt.Sprint(sr[i]) != fmt.Sprint(cr[i]) {
				t.Fatalf("row %d: string %v vs categorical %v", i, sr[i], cr[i])
			}
		}
	})
}

// FuzzCategoricalMergeEquivalence requires the code-based join to match
// the string join pair-for-pair.
func FuzzCategoricalMergeEquivalence(f *testing.F) {
	f.Add(int8(2), uint8(18), uint8(9))
	f.Add(int8(-5), uint8(5), uint8(25))
	f.Fuzz(func(t *testing.T, seed int8, ln, rn uint8) {
		mkFrames := func(labels []any, valName string, categorical bool) *pd.DataFrame {
			key := pd.NewSeries("k", labels)
			if categorical {
				var err error
				if key, err = key.Astype(pd.Category); err != nil {
					t.Fatal(err)
				}
			}
			vals := make([]int, len(labels))
			for i := range vals {
				vals[i] = i
			}
			df, err := pd.NewDataFrame(key, pd.IntSeries(valName, vals))
			if err != nil {
				t.Fatal(err)
			}
			return df
		}
		llab := catLabels(seed, int(ln)%32+1)
		rlab := catLabels(seed+1, int(rn)%32+1)
		for _, how := range []string{"inner", "left", "outer"} {
			ms, err := mkFrames(llab, "l", false).Merge(mkFrames(rlab, "r", false),
				pd.MergeOptions{On: []string{"k"}, How: how})
			if err != nil {
				t.Fatal(err)
			}
			mc, err := mkFrames(llab, "l", true).Merge(mkFrames(rlab, "r", true),
				pd.MergeOptions{On: []string{"k"}, How: how})
			if err != nil {
				t.Fatal(err)
			}
			if ms.Len() != mc.Len() {
				t.Fatalf("%s rows differ: %d vs %d", how, ms.Len(), mc.Len())
			}
			sr, cr := ms.ToRows(), mc.ToRows()
			for i := range sr {
				if fmt.Sprint(sr[i]) != fmt.Sprint(cr[i]) {
					t.Fatalf("%s row %d: %v vs %v", how, i, sr[i], cr[i])
				}
			}
		}
	})
}

// FuzzCategoricalSortConcat requires categorical sort to equal the string
// sort (default categories are sorted labels) and concat to preserve
// every value with the categorical dtype.
func FuzzCategoricalSortConcat(f *testing.F) {
	f.Add(int8(4), uint8(22), uint8(11))
	f.Add(int8(-9), uint8(6), uint8(2))
	f.Fuzz(func(t *testing.T, seed int8, an, bn uint8) {
		alab := catLabels(seed, int(an)%48+1)
		blab := catLabels(seed*3+1, int(bn)%48+1)
		catA, err := pd.NewSeries("v", alab).Astype(pd.Category)
		if err != nil {
			t.Fatal(err)
		}
		strSorted := pd.NewSeries("v", alab).SortValues(true).Values()
		catSorted := catA.SortValues(true).Values()
		for i := range strSorted {
			if strSorted[i] != catSorted[i] {
				t.Fatalf("sort mismatch at %d: %v vs %v", i, strSorted[i], catSorted[i])
			}
		}
		catB, err := pd.NewSeries("v", blab).Astype(pd.Category)
		if err != nil {
			t.Fatal(err)
		}
		dfA, _ := pd.NewDataFrame(catA)
		dfB, _ := pd.NewDataFrame(catB)
		out, err := pd.Concat([]*pd.DataFrame{dfA, dfB}, pd.IgnoreIndex(true))
		if err != nil {
			t.Fatal(err)
		}
		col := out.MustCol("v")
		if col.DType() != pd.Category {
			t.Fatalf("concat dtype = %v", col.DType())
		}
		joined := append(append([]any{}, alab...), blab...)
		for i, w := range joined {
			if col.Values()[i] != w {
				t.Fatalf("concat value %d: %v != %v", i, col.Values()[i], w)
			}
		}
	})
}
