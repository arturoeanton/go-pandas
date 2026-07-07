package series_test

import (
	"errors"
	"testing"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/series"
)

func orderedSizes(t *testing.T) *series.Series {
	t.Helper()
	s, err := series.CategoricalSeries("size", []string{"m", "s", "l", "m", "s"},
		series.WithCategories("s", "m", "l"), series.WithOrdered(true))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCategoricalSeriesConstructor(t *testing.T) {
	s := orderedSizes(t)
	if s.DType() != dtype.Category {
		t.Fatalf("dtype = %v", s.DType())
	}
	cat, err := s.Cat()
	if err != nil {
		t.Fatal(err)
	}
	if !cat.Ordered() || cat.Categories()[0] != "s" {
		t.Fatalf("categories = %v ordered=%v", cat.Categories(), cat.Ordered())
	}
	// Strict mode: out-of-list values error.
	_, err = series.CategoricalSeries("size", []string{"xl"}, series.WithCategories("s", "m"))
	if !errors.Is(err, errs.ErrTypeMismatch) {
		t.Fatalf("strict constructor must error, got %v", err)
	}
	// Cat() on non-categorical errors.
	if _, err := series.FloatSeries("f", []float64{1}).Cat(); !errors.Is(err, errs.ErrInvalidDType) {
		t.Fatalf("Cat() on float must error, got %v", err)
	}
}

func TestAstypeCategoryBothWays(t *testing.T) {
	s := series.StringSeries("s", []string{"b", "a", "b"})
	cat, err := s.Astype(dtype.Category)
	if err != nil {
		t.Fatal(err)
	}
	if cat.DType() != dtype.Category {
		t.Fatalf("dtype = %v", cat.DType())
	}
	acc, _ := cat.Cat()
	if acc.Categories()[0] != "a" {
		t.Fatalf("default categories must sort: %v", acc.Categories())
	}
	back, err := cat.Astype(dtype.String)
	if err != nil {
		t.Fatal(err)
	}
	if back.DType() != dtype.String {
		t.Fatalf("back dtype = %v", back.DType())
	}
	for i, v := range s.Values() {
		if back.Values()[i] != v {
			t.Fatalf("round trip mismatch: %v vs %v", back.Values(), s.Values())
		}
	}
}

func TestCategoricalOrderedComparisons(t *testing.T) {
	s := orderedSizes(t)
	wantGt := []bool{false, false, true, false, false}
	for i, got := range s.Gt("m").AsMask() {
		if got != wantGt[i] {
			t.Fatalf("Gt = %v, want %v", s.Gt("m").AsMask(), wantGt)
		}
	}
	cat, _ := s.Cat()
	if _, err := cat.Ge("m"); err != nil {
		t.Fatal(err)
	}
	if _, err := cat.Gt("xl"); !errors.Is(err, errs.ErrTypeMismatch) {
		t.Fatalf("unknown label must error on accessor, got %v", err)
	}

	u, _ := series.CategoricalSeries("u", []string{"a", "b"})
	uc, _ := u.Cat()
	if _, err := uc.Lt("a"); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Fatalf("unordered ordered-compare must error, got %v", err)
	}
	// Series-level fallback: unordered comparisons are all-false.
	for _, v := range u.Gt("a").AsMask() {
		if v {
			t.Fatal("unordered Series.Gt must be all false")
		}
	}
	// Eq / IsIn always work.
	if got := u.Eq("a").AsMask(); !got[0] || got[1] {
		t.Fatalf("Eq = %v", got)
	}
	if got := u.IsIn("b").AsMask(); got[0] || !got[1] {
		t.Fatalf("IsIn = %v", got)
	}
}

func TestCategoricalAccessorOps(t *testing.T) {
	s := orderedSizes(t)
	cat, _ := s.Cat()

	re, err := cat.ReorderCategories([]any{"l", "m", "s"}, true)
	if err != nil {
		t.Fatal(err)
	}
	rc, _ := re.Cat()
	if rc.Categories()[0] != "l" {
		t.Fatalf("reorder = %v", rc.Categories())
	}
	if _, err := cat.ReorderCategories([]any{"l", "m"}, true); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Fatalf("reorder must keep the set, got %v", err)
	}
	if _, err := cat.ReorderCategories([]any{"l", "m", "xl"}, true); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Fatalf("reorder with new category must error, got %v", err)
	}

	added, err := cat.AddCategories("xl")
	if err != nil {
		t.Fatal(err)
	}
	ac, _ := added.Cat()
	if ac.Categories()[3] != "xl" {
		t.Fatalf("add = %v", ac.Categories())
	}

	removed, err := cat.RemoveCategories("s")
	if err != nil {
		t.Fatal(err)
	}
	if removed.Values()[1] != nil {
		t.Fatal("removed category values must become NA")
	}
}

func TestCategoricalSortAndValueCounts(t *testing.T) {
	s := orderedSizes(t)
	sorted := s.SortValues(true)
	want := []any{"s", "s", "m", "m", "l"}
	for i, w := range want {
		if sorted.Values()[i] != w {
			t.Fatalf("sorted = %v, want %v", sorted.Values(), want)
		}
	}
	desc := s.SortValues(false)
	if desc.Values()[0] != "l" {
		t.Fatalf("desc sorted = %v", desc.Values())
	}
	vc := s.ValueCounts()
	if got := vc.Values(); got[0] != 2 || got[1] != 2 || got[2] != 1 {
		t.Fatalf("value_counts = %v", got)
	}
	if lbl := vc.Index().At(0); lbl != "s" {
		t.Fatalf("value_counts first label = %v, want s (tie keeps category order)", lbl)
	}
}
