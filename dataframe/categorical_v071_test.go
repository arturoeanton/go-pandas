package dataframe_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arturoeanton/go-pandas/dataframe"
	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/series"
)

func sizesWithUnused(t *testing.T) *dataframe.DataFrame {
	t.Helper()
	// "xl" is a declared but unobserved category.
	size, err := series.CategoricalSeries("size", []string{"m", "s", "m"},
		series.WithCategories("s", "m", "l", "xl"), series.WithOrdered(true))
	if err != nil {
		t.Fatal(err)
	}
	df, err := dataframe.NewDataFrame(size, series.FloatSeries("price", []float64{5, 1, 6}))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

// go-pandas groupby is observed-only (pandas observed=True): declared
// but unused categories must not appear as empty groups.
func TestCategoricalGroupByObservedOnly(t *testing.T) {
	g, err := sizesWithUnused(t).GroupBy("size").Mean("price")
	if err != nil {
		t.Fatal(err)
	}
	if g.Len() != 2 {
		t.Fatalf("groups = %d, want 2 observed (no l/xl rows)", g.Len())
	}
	keys := g.MustCol("size").Values()
	if keys[0] != "s" || keys[1] != "m" {
		t.Fatalf("group order = %v, want category order [s m]", keys)
	}
}

func TestCategoricalCSVWritesLabels(t *testing.T) {
	var buf bytes.Buffer
	if err := sizesWithUnused(t).WriteCSV(&buf); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("csv lines = %d: %q", len(lines), buf.String())
	}
	for i, want := range []string{"m", "s", "m"} {
		if got := strings.Split(lines[i+1], ",")[0]; got != want {
			t.Fatalf("row %d wrote %q, want label %q (never a code)", i, got, want)
		}
	}
}

func TestCategoricalJSONWritesLabels(t *testing.T) {
	var buf bytes.Buffer
	if err := sizesWithUnused(t).WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, label := range []string{`"m"`, `"s"`} {
		if !strings.Contains(out, label) {
			t.Fatalf("json output missing label %s: %s", label, out)
		}
	}
	if strings.Contains(out, `"size":0`) || strings.Contains(out, `"size":1`) {
		t.Fatalf("json output leaked codes: %s", out)
	}
}

// Concat of categoricals with different category lists keeps the
// categorical dtype with the union (documented difference vs pandas).
func TestCategoricalConcatUnionPreservesCategory(t *testing.T) {
	a, err := series.CategoricalSeries("size", []string{"s", "m"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := series.CategoricalSeries("size", []string{"m", "xl"})
	if err != nil {
		t.Fatal(err)
	}
	dfa, _ := dataframe.NewDataFrame(a)
	dfb, _ := dataframe.NewDataFrame(b)
	out, err := dataframe.Concat([]*dataframe.DataFrame{dfa, dfb}, dataframe.ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	col := out.MustCol("size")
	if col.DType() != dtype.Category {
		t.Fatalf("concat dtype = %v, want category (union, not object)", col.DType())
	}
	cat, err := col.Cat()
	if err != nil {
		t.Fatal(err)
	}
	if got := cat.Categories(); len(got) != 3 {
		t.Fatalf("union categories = %v, want [m s xl]", got)
	}
	want := []any{"s", "m", "m", "xl"}
	for i, w := range want {
		if col.Values()[i] != w {
			t.Fatalf("values = %v, want %v", col.Values(), want)
		}
	}
}

// SetCategories drops values whose category is removed (they become NA)
// without touching the source series.
func TestCategoricalSetCategoriesRemovedValuesBecomeNA(t *testing.T) {
	s, err := series.CategoricalSeries("size", []string{"s", "m", "l"},
		series.WithCategories("s", "m", "l"), series.WithOrdered(true))
	if err != nil {
		t.Fatal(err)
	}
	cat, _ := s.Cat()
	re, err := cat.SetCategories([]any{"m", "l"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if re.Values()[0] != nil || re.Values()[1] != "m" || re.Values()[2] != "l" {
		t.Fatalf("set_categories values = %v", re.Values())
	}
	if s.Values()[0] != "s" {
		t.Fatal("SetCategories mutated the source series")
	}
	rc, _ := re.Cat()
	if rc.Ordered() {
		t.Fatal("SetCategories must apply the requested ordered flag")
	}
}
