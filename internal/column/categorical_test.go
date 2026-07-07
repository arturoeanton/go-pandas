package column

import (
	"errors"
	"testing"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

func mustFactorize(t *testing.T, values []any, explicit []any, ordered bool) *CategoricalColumn {
	t.Helper()
	c, err := Factorize(values, explicit, ordered)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestFactorizeDefaultSortedCategories(t *testing.T) {
	c := mustFactorize(t, []any{"m", "s", "l", "m", nil, "s"}, nil, false)
	wantCats := []any{"l", "m", "s"}
	gotCats := c.Categories()
	if len(gotCats) != len(wantCats) {
		t.Fatalf("categories = %v, want %v", gotCats, wantCats)
	}
	for i := range wantCats {
		if gotCats[i] != wantCats[i] {
			t.Fatalf("categories = %v, want %v", gotCats, wantCats)
		}
	}
	wantCodes := []int32{1, 2, 0, 1, -1, 2}
	for i, want := range wantCodes {
		if c.Codes()[i] != want {
			t.Fatalf("codes = %v, want %v", c.Codes(), wantCodes)
		}
	}
	if !c.IsNA(4) || c.Value(4) != nil {
		t.Fatal("NA row must stay masked")
	}
	if c.DType() != dtype.Category {
		t.Fatalf("dtype = %v, want category", c.DType())
	}
}

func TestFactorizeExplicitStrict(t *testing.T) {
	c := mustFactorize(t, []any{"b", "a"}, []any{"c", "b", "a"}, true)
	if c.CodeOf("b") != 1 || c.CodeOf("c") != 0 {
		t.Fatal("explicit categories must keep their given order")
	}
	if !c.Ordered() {
		t.Fatal("ordered flag lost")
	}
	if _, err := Factorize([]any{"x"}, []any{"a", "b"}, false); !errors.Is(err, errs.ErrTypeMismatch) {
		t.Fatalf("unknown value must be strict, got %v", err)
	}
	if _, err := Factorize(nil, []any{"a", "a"}, false); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Fatalf("duplicate categories must error, got %v", err)
	}
	if _, err := Factorize([]any{[]int{1}}, nil, false); !errors.Is(err, errs.ErrTypeMismatch) {
		t.Fatalf("unhashable label must error, got %v", err)
	}
}

func TestCategoricalSetAppendTake(t *testing.T) {
	c := mustFactorize(t, []any{"a", "b", "a"}, nil, false)
	if err := c.SetValue(0, "b"); err != nil {
		t.Fatal(err)
	}
	if c.Value(0) != "b" {
		t.Fatalf("SetValue: got %v", c.Value(0))
	}
	if err := c.SetValue(1, "zzz"); !errors.Is(err, errs.ErrTypeMismatch) {
		t.Fatalf("non-category SetValue must error, got %v", err)
	}
	if err := c.SetValue(1, nil); err != nil || !c.IsNA(1) {
		t.Fatalf("NA SetValue: err=%v na=%v", err, c.IsNA(1))
	}
	if err := c.AppendValue("a"); err != nil || c.Len() != 4 {
		t.Fatalf("AppendValue: err=%v len=%d", err, c.Len())
	}
	taken, err := c.Take([]int{3, -1, 0})
	if err != nil {
		t.Fatal(err)
	}
	if taken.Value(0) != "a" || !taken.IsNA(1) || taken.Value(2) != "b" {
		t.Fatalf("Take mismatch: %v", taken.Values())
	}
	tc, _ := AsCategorical(taken)
	if tc.CategoryCount() != 2 {
		t.Fatal("Take must share the category list")
	}
}

func TestWithCategoriesRemapAndNA(t *testing.T) {
	c := mustFactorize(t, []any{"s", "m", "l"}, []any{"s", "m", "l"}, false)
	re, err := c.WithCategories([]any{"m", "l"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !re.IsNA(0) || re.Value(1) != "m" || re.Value(2) != "l" {
		t.Fatalf("remap mismatch: %v", re.Values())
	}
	if !re.Ordered() {
		t.Fatal("ordered flag not applied")
	}
	if _, err := c.WithCategories([]any{"m", "m"}, false); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Fatalf("duplicate must error, got %v", err)
	}
}

func TestRenameCategories(t *testing.T) {
	c := mustFactorize(t, []any{"s", "m"}, []any{"s", "m"}, true)
	re, err := c.RenameCategories(map[any]any{"s": "small"})
	if err != nil {
		t.Fatal(err)
	}
	if re.Value(0) != "small" || re.Value(1) != "m" || !re.Ordered() {
		t.Fatalf("rename mismatch: %v ordered=%v", re.Values(), re.Ordered())
	}
	if re.Codes()[0] != c.Codes()[0] {
		t.Fatal("rename must keep codes")
	}
	if _, err := c.RenameCategories(map[any]any{"s": "m"}); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Fatalf("rename collision must error, got %v", err)
	}
}

func TestConcatCategoricalUnion(t *testing.T) {
	a := mustFactorize(t, []any{"a", "b"}, nil, false)
	b := mustFactorize(t, []any{"b", "c"}, nil, false)
	out := ConcatParts([]ConcatPart{{Col: a, Len: 2}, {Col: b, Len: 2}, {Len: 1}})
	cc, ok := AsCategorical(out)
	if !ok {
		t.Fatalf("concat lost categorical dtype: %T", out)
	}
	if cc.CategoryCount() != 3 {
		t.Fatalf("union categories = %v", cc.Categories())
	}
	want := []any{"a", "b", "b", "c", nil}
	for i, w := range want {
		if cc.Value(i) != w {
			t.Fatalf("values = %v, want %v", cc.Values(), want)
		}
	}
	// Ordered survives only for identical ordered category lists.
	oa := mustFactorize(t, []any{"a"}, []any{"a", "b"}, true)
	ob := mustFactorize(t, []any{"b"}, []any{"a", "b"}, true)
	same, _ := AsCategorical(ConcatParts([]ConcatPart{{Col: oa, Len: 1}, {Col: ob, Len: 1}}))
	if !same.Ordered() {
		t.Fatal("identical ordered parts must stay ordered")
	}
	mixed, _ := AsCategorical(ConcatParts([]ConcatPart{{Col: oa, Len: 1}, {Col: b, Len: 2}}))
	if mixed.Ordered() {
		t.Fatal("differing parts must drop ordered")
	}
}

func TestGatherCoalesceCategorical(t *testing.T) {
	a := mustFactorize(t, []any{"a", "b"}, nil, false)
	b := mustFactorize(t, []any{"c", "b"}, nil, false)
	out, ok := GatherCoalesce(a, b, []int{0, -1, 1}, []int{-1, 0, 1})
	if !ok {
		t.Fatal("categorical coalesce must take the typed path")
	}
	cc, _ := AsCategorical(out)
	if cc.Value(0) != "a" || cc.Value(1) != "c" || cc.Value(2) != "b" {
		t.Fatalf("coalesce values = %v", cc.Values())
	}
	if cc.CategoryCount() != 3 {
		t.Fatalf("coalesce categories = %v", cc.Categories())
	}
}

func TestFromAnyCategory(t *testing.T) {
	c := FromAny([]any{"b", "a", nil}, dtype.Category)
	cc, ok := AsCategorical(c)
	if !ok {
		t.Fatalf("FromAny(Category) = %T", c)
	}
	if cc.Categories()[0] != "a" {
		t.Fatalf("categories = %v, want sorted", cc.Categories())
	}
	// Unhashable values fall back to object storage, like other dtypes.
	if _, ok := AsCategorical(FromAny([]any{[]int{1}}, dtype.Category)); ok {
		t.Fatal("unhashable values must downgrade to object")
	}
}
