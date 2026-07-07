package column

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/arturoeanton/go-pandas/errs"
)

// v0.7.1 policy: implicit categories must come from one label family so
// the sorted default order is total.
func TestCategoricalImplicitMixedTypesPolicy(t *testing.T) {
	if _, err := Factorize([]any{"a", 1}, nil, false); !errors.Is(err, errs.ErrTypeMismatch) {
		t.Fatalf("mixed string+numeric implicit labels must error, got %v", err)
	}
	if _, err := Factorize([]any{true, 1}, nil, false); !errors.Is(err, errs.ErrTypeMismatch) {
		t.Fatalf("mixed bool+numeric implicit labels must error, got %v", err)
	}
	if _, err := Factorize([]any{time.Now(), "x"}, nil, false); !errors.Is(err, errs.ErrTypeMismatch) {
		t.Fatalf("mixed time+string implicit labels must error, got %v", err)
	}
	// One family with NAs mixed in stays fine.
	if _, err := Factorize([]any{"a", nil, "b"}, nil, false); err != nil {
		t.Fatalf("single-family labels with NA must factorize, got %v", err)
	}
	// Numeric widths are ONE family (they order together via AsFloat).
	if _, err := Factorize([]any{1, int64(2), 3.5}, nil, false); err != nil {
		t.Fatalf("mixed numeric widths are one family, got %v", err)
	}
}

// Explicit categories may mix families: the order is user-provided, so
// no total order is required.
func TestCategoricalExplicitMixedCategoriesPolicy(t *testing.T) {
	c, err := Factorize([]any{"a", 1, nil}, []any{"a", 1, true}, false)
	if err != nil {
		t.Fatal(err)
	}
	if c.CodeOf("a") != 0 || c.CodeOf(1) != 1 || c.CodeOf(true) != 2 {
		t.Fatalf("explicit mixed categories must keep given order: %v", c.Categories())
	}
}

func TestCategoricalDefaultNumericCategoryOrder(t *testing.T) {
	c := mustFactorize(t, []any{3, 1, 2, 1}, nil, false)
	want := []any{1, 2, 3}
	for i, w := range want {
		if c.Categories()[i] != w {
			t.Fatalf("numeric categories = %v, want %v", c.Categories(), want)
		}
	}
}

func TestCategoricalDefaultStringCategoryOrder(t *testing.T) {
	c := mustFactorize(t, []any{"m", "s", "l"}, nil, false)
	want := []any{"l", "m", "s"}
	for i, w := range want {
		if c.Categories()[i] != w {
			t.Fatalf("string categories = %v, want %v", c.Categories(), want)
		}
	}
}

func TestCategoricalDefaultTimeCategoryOrder(t *testing.T) {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.AddDate(0, 1, 0)
	t2 := t0.AddDate(0, 2, 0)
	c := mustFactorize(t, []any{t2, t0, t1}, nil, false)
	want := []any{t0, t1, t2}
	for i, w := range want {
		if c.Categories()[i] != w {
			t.Fatalf("time categories = %v, want %v", c.Categories(), want)
		}
	}
}

func highCardinality(t testing.TB, n int) *CategoricalColumn {
	t.Helper()
	values := make([]any, n)
	for i := range values {
		values[i] = fmt.Sprintf("label-%06d", i)
	}
	c, err := Factorize(values, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCategoricalCodeOfHighCardinality(t *testing.T) {
	c := highCardinality(t, 50_000)
	if code := c.CodeOf("label-049999"); code != 49_999 {
		t.Fatalf("CodeOf = %d, want 49999", code)
	}
	if c.CodeOf("missing") != -1 {
		t.Fatal("unknown label must be -1")
	}
	if c.CodeOf([]int{1}) != -1 {
		t.Fatal("unhashable label must be -1, not panic")
	}
	// Derived columns share the same lookup and stay correct.
	taken, err := c.Take([]int{0, 1})
	if err != nil {
		t.Fatal(err)
	}
	tc, _ := AsCategorical(taken)
	if tc.CodeOf("label-000001") != 1 {
		t.Fatal("shared lookup misresolves on derived column")
	}
}

// Concurrent CodeOf across a column and its derived copies must not
// race (the lookup builds once under a sync.Once shared per category
// list); run with -race.
func TestCategoricalLookupDoesNotRace(t *testing.T) {
	// NewCategorical takes the lazy path — the Once really runs here.
	base := mustFactorize(t, []any{"a", "b", "c"}, nil, false)
	c := NewCategorical(base.Codes(), base.Categories(), false, make([]bool, base.Len()))
	sliced, err := c.Slice(0, 2)
	if err != nil {
		t.Fatal(err)
	}
	sc, _ := AsCategorical(sliced)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			target := []any{"a", "b", "c"}[i%3]
			col := c
			if i%2 == 1 {
				col = sc
			}
			if col.CodeOf(target) < 0 {
				t.Errorf("CodeOf(%v) failed", target)
			}
		}(i)
	}
	wg.Wait()
}

func TestCategoricalTakeSharesImmutableCategoriesSafely(t *testing.T) {
	c := mustFactorize(t, []any{"a", "b"}, nil, true)
	taken, err := c.Take([]int{1, 0})
	if err != nil {
		t.Fatal(err)
	}
	tc, _ := AsCategorical(taken)
	// Renaming the derived column must not leak into the source.
	renamed, err := tc.RenameCategories(map[any]any{"a": "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Categories()[0] != "a" || c.CodeOf("a") != 0 {
		t.Fatalf("source categories mutated: %v", c.Categories())
	}
	if renamed.Value(1) != "alpha" || renamed.CodeOf("alpha") != 0 {
		t.Fatalf("rename on derived column wrong: %v", renamed.Values())
	}
}

func TestCategoricalAccessorDoesNotMutateInput(t *testing.T) {
	c := mustFactorize(t, []any{"s", "m", "l"}, []any{"s", "m", "l"}, true)
	wantCodes := c.Codes()
	if _, err := c.WithCategories([]any{"m"}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := c.RenameCategories(map[any]any{"s": "small"}); err != nil {
		t.Fatal(err)
	}
	if c.Categories()[0] != "s" || !c.Ordered() {
		t.Fatalf("input categories mutated: %v ordered=%v", c.Categories(), c.Ordered())
	}
	for i, w := range wantCodes {
		if c.Codes()[i] != w {
			t.Fatalf("input codes mutated: %v", c.Codes())
		}
	}
	if c.CodeOf("small") != -1 || c.CodeOf("s") != 0 {
		t.Fatal("input lookup mutated by accessor operation")
	}
}
