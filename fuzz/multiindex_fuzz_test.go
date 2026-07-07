package fuzz_test

import (
	"fmt"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/index"
)

// miArrays derives two deterministic label arrays (strings + ints, with
// NAs) from fuzz input.
func miArrays(seed int8, n int) ([]any, []any) {
	mod := func(x, m int) int { return ((x % m) + m) % m }
	letters := []string{"a", "b", "c", "d"}
	l0 := make([]any, n)
	l1 := make([]any, n)
	for i := 0; i < n; i++ {
		k := mod(i*5+int(seed), 9)
		if k != 8 {
			l0[i] = letters[mod(k, len(letters))]
		}
		if mod(i+int(seed), 7) != 6 {
			l1[i] = mod(i*3, 5)
		}
	}
	return l0, l1
}

// assertMIInvariants checks the core invariants: names/levels/codes
// aligned, codes valid or -1, level values unique, decode matches input.
func assertMIInvariants(t *testing.T, mi *index.MultiIndex, arrays [][]any) {
	t.Helper()
	if len(mi.Names()) != mi.NLevels() || len(mi.Levels()) != mi.NLevels() || len(mi.Codes()) != mi.NLevels() {
		t.Fatal("names/levels/codes size mismatch")
	}
	levels, codes := mi.Levels(), mi.Codes()
	for l := range levels {
		if len(codes[l]) != mi.Len() {
			t.Fatalf("level %d codes length %d != %d", l, len(codes[l]), mi.Len())
		}
		seen := make(map[any]bool, len(levels[l]))
		for _, v := range levels[l] {
			if seen[v] {
				t.Fatalf("level %d duplicate value %v", l, v)
			}
			seen[v] = true
		}
		for i, c := range codes[l] {
			if c == -1 {
				if arrays != nil && arrays[l][i] != nil {
					t.Fatalf("level %d row %d: NA code for value %v", l, i, arrays[l][i])
				}
				continue
			}
			if int(c) < 0 || int(c) >= len(levels[l]) {
				t.Fatalf("level %d row %d: code %d out of range", l, i, c)
			}
			if arrays != nil && levels[l][c] != arrays[l][i] {
				t.Fatalf("level %d row %d: decode %v != %v", l, i, levels[l][c], arrays[l][i])
			}
		}
	}
}

func FuzzMultiIndexFromArrays(f *testing.F) {
	f.Add(int8(1), uint8(12))
	f.Add(int8(-6), uint8(40))
	f.Fuzz(func(t *testing.T, seed int8, size uint8) {
		n := int(size)%64 + 1
		l0, l1 := miArrays(seed, n)
		in0 := append([]any(nil), l0...)
		in1 := append([]any(nil), l1...)
		mi, err := index.NewMultiIndexFromArrays([][]any{l0, l1}, []string{"a", "b"})
		if err != nil {
			t.Fatal(err)
		}
		if mi.Len() != n {
			t.Fatalf("len = %d, want %d", mi.Len(), n)
		}
		assertMIInvariants(t, mi, [][]any{l0, l1})
		for i := range in0 {
			if l0[i] != in0[i] || l1[i] != in1[i] {
				t.Fatal("input arrays mutated")
			}
		}
	})
}

func FuzzMultiIndexTake(f *testing.F) {
	f.Add(int8(3), uint8(20), uint8(15))
	f.Add(int8(-2), uint8(5), uint8(40))
	f.Fuzz(func(t *testing.T, seed int8, size, nTake uint8) {
		mod := func(x, m int) int { return ((x % m) + m) % m }
		n := int(size)%48 + 1
		l0, l1 := miArrays(seed, n)
		mi, err := index.NewMultiIndexFromArrays([][]any{l0, l1}, []string{"a", "b"})
		if err != nil {
			t.Fatal(err)
		}
		before := fmt.Sprint(mi.Tuples())
		positions := make([]int, int(nTake)%64)
		for i := range positions {
			p := mod(i*7+int(seed), n+1)
			if p == n {
				p = -1 // negative -> NA tuple
			}
			positions[i] = p
		}
		taken := mi.Take(positions).(*index.MultiIndex)
		if taken.Len() != len(positions) {
			t.Fatalf("take len = %d, want %d", taken.Len(), len(positions))
		}
		assertMIInvariants(t, taken, nil)
		for i, p := range positions {
			got := taken.Tuple(i)
			if p < 0 {
				if got[0] != nil || got[1] != nil {
					t.Fatalf("negative position must be all-NA: %v", got)
				}
				continue
			}
			want := mi.Tuple(p)
			if got[0] != want[0] || got[1] != want[1] {
				t.Fatalf("take tuple %d: %v != %v", i, got, want)
			}
		}
		if fmt.Sprint(mi.Tuples()) != before {
			t.Fatal("Take mutated the source index")
		}
	})
}

func FuzzMultiIndexSetResetRoundtrip(f *testing.F) {
	f.Add(int8(4), uint8(16))
	f.Add(int8(-9), uint8(50))
	f.Fuzz(func(t *testing.T, seed int8, size uint8) {
		n := int(size)%48 + 1
		l0, l1 := miArrays(seed, n)
		vals := make([]any, n)
		for i := range vals {
			vals[i] = float64(i) + 0.5
		}
		df, err := pd.DataFrameFromMap(
			map[string][]any{"k1": l0, "k2": l1, "v": vals},
			pd.WithColumnOrder("k1", "k2", "v"))
		if err != nil {
			t.Fatal(err)
		}
		beforeRows := fmt.Sprint(df.ToRows())
		indexed, err := df.SetIndex("k1", "k2")
		if err != nil {
			t.Fatal(err)
		}
		if indexed.Len() != n {
			t.Fatalf("SetIndex changed row count: %d", indexed.Len())
		}
		back := indexed.ResetIndex()
		if back.Len() != n {
			t.Fatalf("ResetIndex changed row count: %d", back.Len())
		}
		if fmt.Sprint(back.ToRows()) != beforeRows {
			t.Fatalf("roundtrip rows differ:\n%v\n%v", back.ToRows(), df.ToRows())
		}
		if fmt.Sprint(df.ToRows()) != beforeRows {
			t.Fatal("input frame mutated")
		}
	})
}

func FuzzMultiIndexTupleLookup(f *testing.F) {
	f.Add(int8(7), uint8(24))
	f.Add(int8(-1), uint8(9))
	f.Fuzz(func(t *testing.T, seed int8, size uint8) {
		n := int(size)%48 + 1
		l0, l1 := miArrays(seed, n)
		mi, err := index.NewMultiIndexFromArrays([][]any{l0, l1}, []string{"a", "b"})
		if err != nil {
			t.Fatal(err)
		}
		// Every row's own tuple must be found by lookup, matching a scan.
		for i := 0; i < n; i++ {
			tup := mi.Tuple(i)
			got := mi.PositionsTuple(tup)
			var want []int
			for j := 0; j < n; j++ {
				o := mi.Tuple(j)
				if o[0] == tup[0] && o[1] == tup[1] {
					want = append(want, j)
				}
			}
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Fatalf("tuple %v: lookup %v != scan %v", tup, got, want)
			}
			// Prefix positions are a superset containing i.
			prefix := mi.PositionsPrefix(tup[:1])
			found := false
			for _, p := range prefix {
				if p == i {
					found = true
				}
			}
			if !found {
				t.Fatalf("prefix %v does not contain row %d", tup[:1], i)
			}
		}
		// Unknown label finds nothing.
		if got := mi.PositionsTuple([]any{"zzz", 999}); got != nil {
			t.Fatalf("unknown tuple = %v", got)
		}
	})
}

func FuzzMultiIndexWherePreservesIndex(f *testing.F) {
	f.Add(int8(2), uint8(30))
	f.Add(int8(-5), uint8(6))
	f.Fuzz(func(t *testing.T, seed int8, size uint8) {
		n := int(size)%48 + 1
		l0, l1 := miArrays(seed, n)
		vals := make([]any, n)
		for i := range vals {
			vals[i] = float64((i*int(seed))%11) + 0.5
		}
		df, err := pd.DataFrameFromMap(
			map[string][]any{"k1": l0, "k2": l1, "v": vals},
			pd.WithColumnOrder("k1", "k2", "v"))
		if err != nil {
			t.Fatal(err)
		}
		indexed, err := df.SetIndex("k1", "k2")
		if err != nil {
			t.Fatal(err)
		}
		filtered, err := indexed.Where(pd.Col("v").Gt(5.0))
		if err != nil {
			t.Fatal(err)
		}
		mi, ok := filtered.Index().(*index.MultiIndex)
		if !ok {
			t.Fatalf("filtered index = %T", filtered.Index())
		}
		if mi.Len() != filtered.Len() {
			t.Fatalf("index length %d != frame length %d", mi.Len(), filtered.Len())
		}
		assertMIInvariants(t, mi, nil)
		// Every surviving row's tuple must exist in the original index.
		for i := 0; i < mi.Len(); i++ {
			if got := indexed.Index().(*index.MultiIndex).PositionsTuple(mi.Tuple(i)); len(got) == 0 {
				t.Fatalf("filtered tuple %v missing from source", mi.Tuple(i))
			}
		}
	})
}
