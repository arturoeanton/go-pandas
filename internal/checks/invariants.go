// Package checks provides test-only invariant validators shared by the
// hardening suites (v0.10.1). Each helper fails the test with a precise
// message when a structural invariant is violated.
package checks

import (
	"testing"

	"github.com/arturoeanton/go-pandas/dataframe"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/internal/column"
	"github.com/arturoeanton/go-pandas/ndarray"
	"github.com/arturoeanton/go-pandas/series"
)

// RequireValidSeries checks the Series structural invariants.
func RequireValidSeries(t testing.TB, s *series.Series) {
	t.Helper()
	if s == nil {
		t.Fatal("nil series")
	}
	n := s.Len()
	if s.Index().Len() != n {
		t.Fatalf("series %q: index length %d != len %d", s.Name(), s.Index().Len(), n)
	}
	if got := len(s.Values()); got != n {
		t.Fatalf("series %q: values length %d != len %d", s.Name(), got, n)
	}
	if s.Storage().Len() != n {
		t.Fatalf("series %q: column length %d != len %d", s.Name(), s.Storage().Len(), n)
	}
	if cc, ok := column.AsCategorical(s.Storage()); ok {
		RequireValidCategoricalColumn(t, cc)
	}
}

// RequireValidCategoricalColumn checks the categorical invariants.
func RequireValidCategoricalColumn(t testing.TB, cc *column.CategoricalColumn) {
	t.Helper()
	cats := cc.Categories()
	seen := make(map[any]bool, len(cats))
	for _, c := range cats {
		if seen[c] {
			t.Fatalf("categorical: duplicate category %v", c)
		}
		seen[c] = true
	}
	codes := cc.Codes()
	for i, code := range codes {
		if (code == -1) != cc.IsNA(i) {
			t.Fatalf("categorical: code %d at %d but IsNA=%v", code, i, cc.IsNA(i))
		}
		if code != -1 && (code < 0 || int(code) >= len(cats)) {
			t.Fatalf("categorical: code %d out of range at %d", code, i)
		}
	}
	for i, c := range cats {
		if got := cc.CodeOf(c); got != int32(i) {
			t.Fatalf("categorical: CodeOf(%v) = %d, want %d", c, got, i)
		}
	}
}

// RequireValidDataFrame checks the DataFrame structural invariants.
func RequireValidDataFrame(t testing.TB, df *dataframe.DataFrame) {
	t.Helper()
	if df == nil {
		t.Fatal("nil dataframe")
	}
	n := df.Len()
	if df.Index().Len() != n {
		t.Fatalf("frame: index length %d != len %d", df.Index().Len(), n)
	}
	for _, name := range df.Columns() {
		c, err := df.Col(name)
		if err != nil {
			t.Fatalf("frame: column %q unreachable: %v", name, err)
		}
		if c == nil {
			t.Fatalf("frame: nil column %q", name)
		}
		if c.Len() != n {
			t.Fatalf("frame: column %q length %d != len %d", name, c.Len(), n)
		}
		RequireValidSeries(t, c)
	}
	RequireValidIndex(t, df.Index())
}

// RequireValidIndex checks generic Index invariants.
func RequireValidIndex(t testing.TB, ix index.Index) {
	t.Helper()
	if ix == nil {
		t.Fatal("nil index")
	}
	if got := len(ix.Values()); got != ix.Len() {
		t.Fatalf("index: values length %d != len %d", got, ix.Len())
	}
	if mi, ok := ix.(*index.MultiIndex); ok {
		RequireValidMultiIndex(t, mi)
	}
}

// RequireValidMultiIndex checks the MultiIndex invariants, including
// that the lookup agrees with a linear scan.
func RequireValidMultiIndex(t testing.TB, mi *index.MultiIndex) {
	t.Helper()
	levels, codes := mi.Levels(), mi.Codes()
	if len(mi.Names()) != mi.NLevels() || len(levels) != mi.NLevels() || len(codes) != mi.NLevels() {
		t.Fatalf("multiindex: names/levels/codes count mismatch")
	}
	for l := range codes {
		if len(codes[l]) != mi.Len() {
			t.Fatalf("multiindex: level %d codes length %d != len %d", l, len(codes[l]), mi.Len())
		}
		for i, c := range codes[l] {
			if c != -1 && (c < 0 || int(c) >= len(levels[l])) {
				t.Fatalf("multiindex: level %d code %d out of range at %d", l, c, i)
			}
		}
	}
	for i := 0; i < mi.Len(); i++ {
		tup := mi.Tuple(i)
		if len(tup) != mi.NLevels() {
			t.Fatalf("multiindex: tuple %d has %d components", i, len(tup))
		}
		// Lookup agrees with a scan for this row's own tuple.
		found := false
		for _, p := range mi.PositionsTuple(tup) {
			if p == i {
				found = true
			}
		}
		if !found {
			t.Fatalf("multiindex: lookup misses row %d tuple %v", i, tup)
		}
	}
}

// RequireValidNDArray checks the NDArray structural invariants.
func RequireValidNDArray(t testing.TB, a *ndarray.NDArray) {
	t.Helper()
	if a == nil {
		t.Fatal("nil ndarray")
	}
	shape := a.Shape()
	prod := 1
	for _, s := range shape {
		if s < 0 {
			t.Fatalf("ndarray: negative dimension in %v", shape)
		}
		prod *= s
	}
	if prod != a.Size() {
		t.Fatalf("ndarray: shape %v product %d != size %d", shape, prod, a.Size())
	}
	if got := len(a.Values()); got != a.Size() {
		t.Fatalf("ndarray: values length %d != size %d", got, a.Size())
	}
}
