package fuzz_test

import (
	"fmt"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

// assertCatInvariants checks the core categorical invariants: unique
// categories, codes in range, code -1 exactly on NA values.
func assertCatInvariants(t *testing.T, s *pd.Series) {
	t.Helper()
	cat, err := s.Cat()
	if err != nil {
		t.Fatal(err)
	}
	cats, codes := cat.Categories(), cat.Codes()
	seen := make(map[any]bool, len(cats))
	for _, c := range cats {
		if seen[c] {
			t.Fatalf("duplicate category %v in %v", c, cats)
		}
		seen[c] = true
	}
	if len(codes) != s.Len() {
		t.Fatalf("codes length %d != series length %d", len(codes), s.Len())
	}
	for i, code := range codes {
		na := s.Values()[i] == nil
		if na != (code == -1) {
			t.Fatalf("row %d: code %d vs NA %v", i, code, na)
		}
		if code != -1 && int(code) >= len(cats) {
			t.Fatalf("row %d: code %d out of range (%d categories)", i, code, len(cats))
		}
	}
}

// FuzzCategoricalExplicitCategories stresses strict explicit-category
// construction: given categories keep order and are not mutated, values
// outside the list error instead of panicking.
func FuzzCategoricalExplicitCategories(f *testing.F) {
	f.Add(int8(1), uint8(12), uint8(3))
	f.Add(int8(-4), uint8(40), uint8(5))
	f.Fuzz(func(t *testing.T, seed int8, size, ncats uint8) {
		mod := func(x, m int) int { return ((x % m) + m) % m }
		k := int(ncats)%5 + 1
		letters := []string{"a", "b", "c", "d", "e"}
		explicit := make([]any, k)
		for i := range explicit {
			explicit[i] = letters[i]
		}
		given := append([]any(nil), explicit...)

		values := catLabels(seed, int(size)%48+1)
		inList := true
		for _, v := range values {
			if v == nil {
				continue
			}
			if mod(int(v.(string)[0]-'a'), len(letters)) >= k {
				inList = false
			}
		}
		s, err := pd.NewCategoricalSeries("v", values,
			pd.WithCategories(explicit...), pd.WithOrdered(seed%2 == 0))
		if !inList {
			if err == nil {
				t.Fatal("out-of-list value must error in strict mode")
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		assertCatInvariants(t, s)
		cat, _ := s.Cat()
		for i, c := range cat.Categories() {
			if c != given[i] {
				t.Fatalf("explicit category order changed: %v vs %v", cat.Categories(), given)
			}
		}
		for i, c := range explicit {
			if c != given[i] {
				t.Fatal("input categories slice mutated")
			}
		}
	})
}

// FuzzCategoricalSetCategories checks that SetCategories keeps values
// whose category survives and turns removed ones into NA, with the
// source series untouched.
func FuzzCategoricalSetCategories(f *testing.F) {
	f.Add(int8(2), uint8(20), uint8(2))
	f.Add(int8(-8), uint8(6), uint8(0))
	f.Fuzz(func(t *testing.T, seed int8, size, keep uint8) {
		values := catLabels(seed, int(size)%48+1)
		s, err := pd.NewCategoricalSeries("v", values)
		if err != nil {
			t.Fatal(err)
		}
		before := fmt.Sprint(s.Values())
		cat, _ := s.Cat()
		full := cat.Categories()
		kept := full[:int(keep)%(len(full)+1)]
		re, err := cat.SetCategories(kept, false)
		if err != nil {
			t.Fatal(err)
		}
		assertCatInvariants(t, re)
		surviving := make(map[any]bool, len(kept))
		for _, c := range kept {
			surviving[c] = true
		}
		for i, v := range s.Values() {
			got := re.Values()[i]
			if v == nil || !surviving[v] {
				if got != nil {
					t.Fatalf("row %d: removed category kept value %v", i, got)
				}
				continue
			}
			if got != v {
				t.Fatalf("row %d: surviving value changed %v -> %v", i, v, got)
			}
		}
		if fmt.Sprint(s.Values()) != before {
			t.Fatal("SetCategories mutated the source series")
		}
	})
}

// FuzzCategoricalConcatUnion checks that concatenating categoricals
// keeps the dtype, unions categories without duplicates, and preserves
// every value.
func FuzzCategoricalConcatUnion(f *testing.F) {
	f.Add(int8(5), uint8(14), uint8(9))
	f.Add(int8(-2), uint8(3), uint8(30))
	f.Fuzz(func(t *testing.T, seed int8, an, bn uint8) {
		alab := catLabels(seed, int(an)%48+1)
		blab := catLabels(seed*5+3, int(bn)%48+1)
		a, err := pd.NewCategoricalSeries("v", alab)
		if err != nil {
			t.Fatal(err)
		}
		b, err := pd.NewCategoricalSeries("v", blab)
		if err != nil {
			t.Fatal(err)
		}
		dfa, _ := pd.NewDataFrame(a)
		dfb, _ := pd.NewDataFrame(b)
		out, err := pd.Concat([]*pd.DataFrame{dfa, dfb}, pd.IgnoreIndex(true))
		if err != nil {
			t.Fatal(err)
		}
		col := out.MustCol("v")
		if col.DType() != pd.Category {
			t.Fatalf("concat dtype = %v", col.DType())
		}
		assertCatInvariants(t, col)
		joined := append(append([]any{}, alab...), blab...)
		if col.Len() != len(joined) {
			t.Fatalf("concat length %d, want %d", col.Len(), len(joined))
		}
		for i, w := range joined {
			if col.Values()[i] != w {
				t.Fatalf("row %d: %v != %v", i, col.Values()[i], w)
			}
		}
	})
}
