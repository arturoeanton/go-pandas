package fuzz_test

import (
	"fmt"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/ndarray"
)

// FuzzQueryParserNoPanic: any string either parses or errors — never
// panics, and a parse error never comes back as a silent predicate.
func FuzzQueryParserNoPanic(f *testing.F) {
	f.Add("a > 1 and (b + 2) * 3 < c")
	f.Add(`k in ["x", 'y'] or not (v % 2 == 0)`)
	f.Add("((((((((")
	f.Add("\"unclosed")
	f.Add("[1, 2")
	f.Add("date >= \"2026-99-99\"")
	f.Add("- - - 1 > 0")
	f.Add("")
	f.Add("   ")
	f.Fuzz(func(t *testing.T, q string) {
		pred, err := expr.ParseQuery(q)
		if err == nil && pred == nil {
			t.Fatal("nil predicate without error")
		}
		if err != nil && pred != nil {
			t.Fatal("error with non-nil predicate")
		}
	})
}

// FuzzSeriesTakeSlice: gather invariants over a masked typed series.
func FuzzSeriesTakeSlice(f *testing.F) {
	f.Add(int8(1), uint8(20), uint8(7))
	f.Add(int8(-6), uint8(3), uint8(50))
	f.Fuzz(func(t *testing.T, seed int8, size, nTake uint8) {
		mod := func(x, m int) int { return ((x % m) + m) % m }
		n := int(size)%48 + 1
		values := make([]any, n)
		for i := range values {
			if mod(i+int(seed), 5) != 4 {
				values[i] = float64(i)
			}
		}
		s := pd.NewSeries("v", values)
		before := fmt.Sprint(s.Values())

		positions := make([]int, int(nTake)%48)
		for i := range positions {
			positions[i] = mod(i*13+int(seed), n)
		}
		taken, err := s.Take(positions)
		if err != nil {
			t.Fatal(err)
		}
		if taken.Len() != len(positions) {
			t.Fatalf("take len = %d", taken.Len())
		}
		if taken.DType() != s.DType() {
			t.Fatalf("take dtype = %v", taken.DType())
		}
		for i, p := range positions {
			if fmt.Sprint(taken.Values()[i]) != fmt.Sprint(values[p]) {
				t.Fatalf("take[%d] = %v, want %v", i, taken.Values()[i], values[p])
			}
		}
		lo, hi := mod(int(seed), n), n
		sliced, err := s.Slice(lo, hi)
		if err != nil {
			t.Fatal(err)
		}
		if sliced.Len() != hi-lo {
			t.Fatalf("slice len = %d", sliced.Len())
		}
		if fmt.Sprint(s.Values()) != before {
			t.Fatal("input mutated")
		}
	})
}

// FuzzDataFrameWhereQuery: columnar Where and equivalent Query agree.
func FuzzDataFrameWhereQuery(f *testing.F) {
	f.Add(int8(2), uint8(25), uint8(9))
	f.Add(int8(-8), uint8(6), uint8(2))
	f.Fuzz(func(t *testing.T, seed int8, size, threshold uint8) {
		mod := func(x, m int) int { return ((x % m) + m) % m }
		n := int(size)%48 + 1
		th := float64(threshold % 16)
		vals := make([]any, n)
		for i := range vals {
			if mod(i+int(seed), 7) != 6 {
				vals[i] = float64(mod(i*3+int(seed), 16))
			}
		}
		df, err := pd.DataFrameFromMap(map[string][]any{"v": vals})
		if err != nil {
			t.Fatal(err)
		}
		w, err := df.Where(pd.Col("v").Gt(th))
		if err != nil {
			t.Fatal(err)
		}
		q, err := df.Query(fmt.Sprintf("v > %v", th))
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprint(w.ToRows()) != fmt.Sprint(q.ToRows()) {
			t.Fatalf("Where and Query disagree:\n%v\n%v", w.ToRows(), q.ToRows())
		}
	})
}

// FuzzNDArrayBroadcastAdd: broadcast arithmetic keeps the broadcast shape
// and never panics on shape mismatches.
func FuzzNDArrayBroadcastAdd(f *testing.F) {
	f.Add(uint8(3), uint8(4), true)
	f.Add(uint8(5), uint8(5), false)
	f.Fuzz(func(t *testing.T, rows, cols uint8, mismatch bool) {
		r, c := int(rows)%6+1, int(cols)%6+1
		m := make([]float64, r*c)
		for i := range m {
			m[i] = float64(i)
		}
		a, err := ndarray.FromSlice(m, r, c)
		if err != nil {
			t.Fatal(err)
		}
		vlen := c
		if mismatch {
			vlen = c + 1
		}
		vec := make([]float64, vlen)
		for i := range vec {
			vec[i] = float64(i)
		}
		out, err := a.Add(ndarray.Array(vec))
		// (r, c) + (vlen,) broadcasts iff vlen == c, vlen == 1 or c == 1
		// (the 1-sized dimension stretches, NumPy rules).
		valid := vlen == c || vlen == 1 || c == 1
		if !valid {
			if err == nil {
				t.Fatal("shape mismatch must error")
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		wantCols := c
		if vlen > c {
			wantCols = vlen
		}
		if got := out.Shape(); got[0] != r || got[1] != wantCols {
			t.Fatalf("broadcast shape = %v, want [%d %d]", got, r, wantCols)
		}
	})
}

// FuzzNDArrayReshapeTranspose: reshape/transpose preserve size and
// round-trip values.
func FuzzNDArrayReshapeTranspose(f *testing.F) {
	f.Add(uint8(3), uint8(4))
	f.Add(uint8(1), uint8(9))
	f.Fuzz(func(t *testing.T, rows, cols uint8) {
		r, c := int(rows)%8+1, int(cols)%8+1
		data := make([]float64, r*c)
		for i := range data {
			data[i] = float64(i)
		}
		a, err := ndarray.FromSlice(data, r, c)
		if err != nil {
			t.Fatal(err)
		}
		tr, err := a.T()
		if err != nil {
			t.Fatal(err)
		}
		if got := tr.Shape(); got[0] != c || got[1] != r {
			t.Fatalf("transpose shape = %v", got)
		}
		back, err := tr.T()
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprint(back.Values()) != fmt.Sprint(a.Values()) {
			t.Fatal("double transpose changed values")
		}
		flat, err := a.Reshape(r * c)
		if err != nil {
			t.Fatal(err)
		}
		if flat.Size() != r*c {
			t.Fatalf("reshape size = %d", flat.Size())
		}
		if _, err := a.Reshape(r*c + 1); err == nil {
			t.Fatal("bad reshape must error")
		}
	})
}

// FuzzNDArrayReductions: axis reductions agree with manual sums.
func FuzzNDArrayReductions(f *testing.F) {
	f.Add(uint8(3), uint8(4), int8(2))
	f.Add(uint8(6), uint8(2), int8(-3))
	f.Fuzz(func(t *testing.T, rows, cols uint8, seed int8) {
		r, c := int(rows)%6+1, int(cols)%6+1
		data := make([]float64, r*c)
		for i := range data {
			data[i] = float64((i*int(seed))%9) + 0.5
		}
		a, err := ndarray.FromSlice(data, r, c)
		if err != nil {
			t.Fatal(err)
		}
		colSum, err := a.Sum(pd.Axis(0))
		if err != nil {
			t.Fatal(err)
		}
		if colSum.Size() != c {
			t.Fatalf("axis0 sum size = %d", colSum.Size())
		}
		for j := 0; j < c; j++ {
			want := 0.0
			for i := 0; i < r; i++ {
				want += data[i*c+j]
			}
			got, _ := colSum.At(j)
			if got != want {
				t.Fatalf("col %d sum = %v, want %v", j, got, want)
			}
		}
		total := a.SumAll()
		var want float64
		for _, v := range data {
			want += v
		}
		if total != want {
			t.Fatalf("SumAll = %v, want %v", total, want)
		}
	})
}
