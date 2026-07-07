package dataframe

import (
	"errors"
	"testing"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/expr"
)

func TestRecordsWithMissingKeys(t *testing.T) {
	df, err := DataFrameFromRecords([]map[string]any{
		{"a": 1, "b": "x"},
		{"a": 2}, // b missing -> NA
		{"b": "z"},
	}, WithColumnOrder("a", "b"))
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, df, "b"); v[1] != nil {
		t.Errorf("missing key should be NA, got %v", v[1])
	}
	if v := colValues(t, df, "a"); v[2] != nil {
		t.Errorf("missing key should be NA, got %v", v[2])
	}
	if df.Count()["b"] != 2 {
		t.Errorf("count with NA = %v", df.Count())
	}
}

func TestAssignAndFilterDoNotMutate(t *testing.T) {
	df := sampleFrame(t)
	before := df.ToRows()
	if _, err := df.AssignExpr("x2", expr.Col("age").Mul(2)); err != nil {
		t.Fatal(err)
	}
	if _, err := df.AssignValue("age", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := df.Where(expr.Col("age").Gt(100)); err != nil {
		t.Fatal(err)
	}
	if _, err := df.SortValues("salary", true); err != nil {
		t.Fatal(err)
	}
	_ = df.DropNA()
	after := df.ToRows()
	if len(before) != len(after) {
		t.Fatal("row count changed")
	}
	if len(df.Columns()) != 4 {
		t.Fatalf("columns changed: %v", df.Columns())
	}
	for i := range before {
		for j := range before[i] {
			if before[i][j] != after[i][j] {
				t.Fatalf("cell [%d][%d] mutated: %v -> %v", i, j, before[i][j], after[i][j])
			}
		}
	}
}

func TestMergeDuplicateKeys(t *testing.T) {
	left, _ := DataFrameFromRecords([]map[string]any{
		{"id": 1, "l": "a"},
		{"id": 1, "l": "b"},
		{"id": 2, "l": "c"},
	}, WithColumnOrder("id", "l"))
	right, _ := DataFrameFromRecords([]map[string]any{
		{"id": 1, "r": "x"},
		{"id": 1, "r": "y"},
		{"id": 3, "r": "z"},
	}, WithColumnOrder("id", "r"))

	inner, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "inner"})
	if err != nil {
		t.Fatal(err)
	}
	// 2 left rows x 2 right rows for key 1 = 4 pairs
	if inner.Len() != 4 {
		t.Fatalf("inner duplicate keys len = %d, want 4", inner.Len())
	}
	leftJoin, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "left"})
	if err != nil {
		t.Fatal(err)
	}
	// 4 pairs + unmatched id=2
	if leftJoin.Len() != 5 {
		t.Fatalf("left duplicate keys len = %d, want 5", leftJoin.Len())
	}
	outer, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "outer"})
	if err != nil {
		t.Fatal(err)
	}
	// 4 pairs + left-only id=2 + right-only id=3
	if outer.Len() != 6 {
		t.Fatalf("outer duplicate keys len = %d, want 6", outer.Len())
	}
	// validate catches the fan-out
	if _, err := left.Merge(right, MergeOptions{On: []string{"id"}, Validate: "many_to_one"}); !errors.Is(err, errs.ErrInvalidJoin) {
		t.Errorf("many_to_one with duplicate right keys error = %v", err)
	}
	if _, err := left.Merge(right, MergeOptions{On: []string{"id"}, Validate: "one_to_many"}); !errors.Is(err, errs.ErrInvalidJoin) {
		t.Errorf("one_to_many with duplicate left keys error = %v", err)
	}
}

func TestGroupByNAKeyOrderAndSize(t *testing.T) {
	df, _ := DataFrameFromRecords([]map[string]any{
		{"k": "b", "v": 1.0, "w": nil},
		{"k": nil, "v": 2.0, "w": 1},
		{"k": "a", "v": 3.0, "w": nil},
		{"k": "a", "v": nil, "w": 2},
	}, WithColumnOrder("k", "v", "w"))
	// sorted group keys, NA rows dropped by default
	size, err := df.GroupBy("k").Size()
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, size, "k"); v[0] != "a" || v[1] != "b" {
		t.Errorf("group order = %v", v)
	}
	// Size counts all rows (even those with NA in value columns);
	// Count skips NA values.
	if v := colValues(t, size, "size"); v[0] != 2 {
		t.Errorf("size = %v", v)
	}
	count, err := df.GroupBy("k").Count("v")
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, count, "v"); v[0] != 1 {
		t.Errorf("count skips NA: %v", v)
	}
	// unsorted keeps first-seen order; NA kept as its own group
	unsorted, err := df.GroupByOpts([]GroupByOption{GroupSort(false), GroupDropNA(false)}, "k").Size()
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, unsorted, "k"); v[0] != "b" || v[1] != nil || v[2] != "a" {
		t.Errorf("unsorted group keys = %v", v)
	}
}

func TestCorrWithNA(t *testing.T) {
	df, _ := DataFrameFromMap(map[string][]any{
		"x": {1.0, 2.0, 3.0, nil},
		"y": {2.0, 4.0, 6.0, 100.0}, // the 100 pairs with NA -> ignored
	}, WithColumnOrder("x", "y"))
	corr, err := df.Corr()
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, corr, "y"); v[0].(float64) < 0.999999 {
		t.Errorf("pairwise-complete corr = %v", v)
	}
}
