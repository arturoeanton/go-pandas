package dataframe

import (
	"errors"
	"testing"

	"github.com/arturoeanton/go-pandas/errs"
)

func TestGroupByBasics(t *testing.T) {
	df := sampleFrame(t)
	sum, err := df.GroupBy("country").Sum("salary")
	if err != nil {
		t.Fatal(err)
	}
	if sum.Len() != 2 {
		t.Fatalf("groups = %d", sum.Len())
	}
	// sorted: AR first
	if v := colValues(t, sum, "country"); v[0] != "AR" || v[1] != "BR" {
		t.Errorf("group keys = %v", v)
	}
	if v := colValues(t, sum, "salary"); v[0] != 3000.0 || v[1] != 1500.0 {
		t.Errorf("group sums = %v", v)
	}
	mean, err := df.GroupBy("country").Mean("salary")
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, mean, "salary"); v[0] != 1500.0 {
		t.Errorf("group means = %v", v)
	}
	count, err := df.GroupBy("country").Count("name")
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, count, "name"); v[0] != 2 || v[1] != 1 {
		t.Errorf("group counts = %v", v)
	}
	size, err := df.GroupBy("country").Size()
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, size, "size"); v[0] != 2 {
		t.Errorf("group sizes = %v", v)
	}
	if _, err := df.GroupBy("nope").Sum(); !errors.Is(err, errs.ErrColumnNotFound) {
		t.Errorf("bad key error = %v", err)
	}
}

func TestGroupByTwoKeysAndAgg(t *testing.T) {
	df, err := DataFrameFromRecords([]map[string]any{
		{"country": "AR", "year": 2023, "sales": 10.0},
		{"country": "AR", "year": 2024, "sales": 20.0},
		{"country": "AR", "year": 2024, "sales": 30.0},
		{"country": "BR", "year": 2023, "sales": 5.0},
	}, WithColumnOrder("country", "year", "sales"))
	if err != nil {
		t.Fatal(err)
	}
	grouped, err := df.GroupBy("country", "year").Sum("sales")
	if err != nil {
		t.Fatal(err)
	}
	if grouped.Len() != 3 {
		t.Fatalf("two-key groups = %d", grouped.Len())
	}
	if v := colValues(t, grouped, "sales"); v[1] != 50.0 {
		t.Errorf("AR/2024 sum = %v", v)
	}
	agg, err := df.GroupBy("country").Agg(map[string]string{"sales": "mean"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := agg.Col("sales_mean"); err != nil {
		t.Errorf("agg column naming: %v (have %v)", err, agg.Columns())
	}
	multi, err := df.GroupBy("country").AggList(map[string][]string{"sales": {"min", "max"}})
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, multi, "sales_max"); v[0] != 30.0 {
		t.Errorf("agg list max = %v", v)
	}
}

func TestGroupByMissingKeys(t *testing.T) {
	df, err := DataFrameFromRecords([]map[string]any{
		{"k": "a", "v": 1.0},
		{"k": nil, "v": 2.0},
		{"k": "a", "v": 3.0},
	}, WithColumnOrder("k", "v"))
	if err != nil {
		t.Fatal(err)
	}
	sum, err := df.GroupBy("k").Sum("v")
	if err != nil {
		t.Fatal(err)
	}
	if sum.Len() != 1 {
		t.Errorf("NA group should be dropped, groups = %d", sum.Len())
	}
	kept, err := df.GroupByOpts([]GroupByOption{GroupDropNA(false)}, "k").Sum("v")
	if err != nil {
		t.Fatal(err)
	}
	if kept.Len() != 2 {
		t.Errorf("NA group kept, groups = %d", kept.Len())
	}
}

func mergeFrames(t *testing.T) (*DataFrame, *DataFrame) {
	t.Helper()
	left, err := DataFrameFromRecords([]map[string]any{
		{"id": 1, "name": "Ana"},
		{"id": 2, "name": "Luis"},
		{"id": 3, "name": "Marta"},
	}, WithColumnOrder("id", "name"))
	if err != nil {
		t.Fatal(err)
	}
	right, err := DataFrameFromRecords([]map[string]any{
		{"id": 1, "salary": 1000.0},
		{"id": 2, "salary": 2000.0},
		{"id": 4, "salary": 4000.0},
	}, WithColumnOrder("id", "salary"))
	if err != nil {
		t.Fatal(err)
	}
	return left, right
}

func TestMergeInnerLeftOuter(t *testing.T) {
	left, right := mergeFrames(t)
	inner, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "inner"})
	if err != nil {
		t.Fatal(err)
	}
	if inner.Len() != 2 {
		t.Fatalf("inner len = %d", inner.Len())
	}
	if v := colValues(t, inner, "salary"); v[1] != 2000.0 {
		t.Errorf("inner salary = %v", v)
	}

	leftJoin, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "left"})
	if err != nil {
		t.Fatal(err)
	}
	if leftJoin.Len() != 3 {
		t.Fatalf("left len = %d", leftJoin.Len())
	}
	if v := colValues(t, leftJoin, "salary"); v[2] != nil {
		t.Errorf("left join unmatched salary = %v", v)
	}

	outer, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "outer"})
	if err != nil {
		t.Fatal(err)
	}
	if outer.Len() != 4 {
		t.Fatalf("outer len = %d", outer.Len())
	}
	if v := colValues(t, outer, "id"); v[3] != 4 {
		t.Errorf("outer right-only id = %v", v)
	}
	if v := colValues(t, outer, "name"); v[3] != nil {
		t.Errorf("outer right-only name = %v", v)
	}

	rightJoin, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "right"})
	if err != nil {
		t.Fatal(err)
	}
	if rightJoin.Len() != 3 {
		t.Fatalf("right len = %d", rightJoin.Len())
	}
}

func TestMergeVariants(t *testing.T) {
	left, right := mergeFrames(t)
	// cross join
	cross, err := left.Merge(right, MergeOptions{How: "cross"})
	if err != nil {
		t.Fatal(err)
	}
	if cross.Len() != 9 {
		t.Errorf("cross len = %d", cross.Len())
	}
	// duplicate non-key columns get suffixes
	l2, _ := DataFrameFromRecords([]map[string]any{{"id": 1, "v": 1}}, WithColumnOrder("id", "v"))
	r2, _ := DataFrameFromRecords([]map[string]any{{"id": 1, "v": 2}}, WithColumnOrder("id", "v"))
	suffixed, err := l2.Merge(r2, MergeOptions{On: []string{"id"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := suffixed.Col("v_x"); err != nil {
		t.Errorf("suffix _x missing: %v", suffixed.Columns())
	}
	if _, err := suffixed.Col("v_y"); err != nil {
		t.Errorf("suffix _y missing: %v", suffixed.Columns())
	}
	// different key names
	r3, _ := DataFrameFromRecords([]map[string]any{{"key": 1, "w": 9}}, WithColumnOrder("key", "w"))
	byNames, err := l2.Merge(r3, MergeOptions{LeftOn: []string{"id"}, RightOn: []string{"key"}})
	if err != nil {
		t.Fatal(err)
	}
	if byNames.Len() != 1 {
		t.Errorf("LeftOn/RightOn len = %d", byNames.Len())
	}
	// missing key column
	if _, err := left.Merge(right, MergeOptions{On: []string{"nope"}}); !errors.Is(err, errs.ErrColumnNotFound) {
		t.Errorf("missing key error = %v", err)
	}
	// validate one_to_one
	dup, _ := DataFrameFromRecords([]map[string]any{{"id": 1, "z": 1}, {"id": 1, "z": 2}}, WithColumnOrder("id", "z"))
	if _, err := left.Merge(dup, MergeOptions{On: []string{"id"}, Validate: "one_to_one"}); !errors.Is(err, errs.ErrInvalidJoin) {
		t.Errorf("validate error = %v", err)
	}
	// no match at all
	far, _ := DataFrameFromRecords([]map[string]any{{"id": 99, "q": 1}}, WithColumnOrder("id", "q"))
	none, err := left.Merge(far, MergeOptions{On: []string{"id"}, How: "inner"})
	if err != nil {
		t.Fatal(err)
	}
	if none.Len() != 0 {
		t.Errorf("no-match inner len = %d", none.Len())
	}
	// indicator
	ind, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "outer", Indicator: true})
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, ind, "_merge"); v[0] != "both" || v[3] != "right_only" {
		t.Errorf("indicator = %v", v)
	}
}

func TestJoinByIndex(t *testing.T) {
	left, err := DataFrameFromRecords([]map[string]any{
		{"v": 1}, {"v": 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	right, err := DataFrameFromRecords([]map[string]any{
		{"w": 10}, {"w": 20},
	})
	if err != nil {
		t.Fatal(err)
	}
	joined, err := left.Join(right, JoinOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if joined.Len() != 2 {
		t.Fatalf("join len = %d", joined.Len())
	}
	if v := colValues(t, joined, "w"); v[1] != 20 {
		t.Errorf("join = %v", v)
	}
	// overlapping columns need suffixes
	right2, _ := DataFrameFromRecords([]map[string]any{{"v": 9}, {"v": 8}})
	if _, err := left.Join(right2, JoinOptions{}); !errors.Is(err, errs.ErrInvalidJoin) {
		t.Errorf("overlap without suffix error = %v", err)
	}
	ok, err := left.Join(right2, JoinOptions{LSuffix: "_l", RSuffix: "_r"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ok.Col("v_l"); err != nil {
		t.Errorf("join suffix columns = %v", ok.Columns())
	}
}

func TestConcat(t *testing.T) {
	a, _ := DataFrameFromRecords([]map[string]any{{"x": 1, "y": "a"}}, WithColumnOrder("x", "y"))
	b, _ := DataFrameFromRecords([]map[string]any{{"x": 2, "y": "b"}}, WithColumnOrder("x", "y"))
	out, err := Concat([]*DataFrame{a, b}, ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 2 {
		t.Fatalf("concat len = %d", out.Len())
	}
	if v := colValues(t, out, "x"); v[1] != 2 {
		t.Errorf("concat = %v", v)
	}
	// different columns fill with NA
	c, _ := DataFrameFromRecords([]map[string]any{{"x": 3, "z": true}}, WithColumnOrder("x", "z"))
	mixed, err := Concat([]*DataFrame{a, c}, ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	if got := mixed.Columns(); len(got) != 3 {
		t.Fatalf("union columns = %v", got)
	}
	if v := colValues(t, mixed, "z"); v[0] != nil || v[1] != true {
		t.Errorf("NA fill = %v", v)
	}
	// inner join keeps common columns only
	inner, err := Concat([]*DataFrame{a, c}, ConcatJoin("inner"), ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	if got := inner.Columns(); len(got) != 1 || got[0] != "x" {
		t.Errorf("inner concat columns = %v", got)
	}
	// empty input
	empty, err := Concat(nil)
	if err != nil || empty.Len() != 0 {
		t.Errorf("empty concat = %v, %v", empty, err)
	}
	// axis=1
	wide, err := Concat([]*DataFrame{a, c}, ConcatAxis(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(wide.Columns()) != 4 {
		t.Errorf("axis=1 columns = %v", wide.Columns())
	}
}

func TestReshape(t *testing.T) {
	df, _ := DataFrameFromRecords([]map[string]any{
		{"name": "Ana", "math": 9.0, "bio": 8.0},
		{"name": "Luis", "math": 7.0, "bio": 6.0},
	}, WithColumnOrder("name", "math", "bio"))
	melted, err := df.Melt(MeltOptions{IDVars: []string{"name"}})
	if err != nil {
		t.Fatal(err)
	}
	if melted.Len() != 4 {
		t.Fatalf("melt len = %d", melted.Len())
	}
	if got := melted.Columns(); got[1] != "variable" || got[2] != "value" {
		t.Errorf("melt columns = %v", got)
	}
	pivoted, err := melted.Pivot(PivotOptions{Index: "name", Columns: "variable", Values: "value"})
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, pivoted, "math"); v[0] != 9.0 {
		t.Errorf("pivot = %v", v)
	}
	// duplicates are an error for Pivot
	dup, _ := DataFrameFromRecords([]map[string]any{
		{"i": "a", "c": "x", "v": 1.0},
		{"i": "a", "c": "x", "v": 2.0},
	}, WithColumnOrder("i", "c", "v"))
	if _, err := dup.Pivot(PivotOptions{Index: "i", Columns: "c", Values: "v"}); err == nil {
		t.Error("pivot with duplicates should fail")
	}
	// ... but PivotTable aggregates them
	pt, err := dup.PivotTable(PivotTableOptions{
		Index: []string{"i"}, Columns: []string{"c"}, Values: []string{"v"}, AggFunc: "mean",
	})
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, pt, "x"); v[0] != 1.5 {
		t.Errorf("pivot table = %v", v)
	}
	if _, err := df.Stack(); !errors.Is(err, errs.ErrNotImplementedBase) {
		t.Errorf("Stack error = %v", err)
	}
	if _, err := df.Unstack(); !errors.Is(err, errs.ErrNotImplementedBase) {
		t.Errorf("Unstack error = %v", err)
	}
	if _, err := df.Resample("1D"); !errors.Is(err, errs.ErrNotImplementedBase) {
		t.Errorf("Resample error = %v", err)
	}
}

func TestDataFrameRolling(t *testing.T) {
	df, _ := DataFrameFromRecords([]map[string]any{
		{"v": 1.0, "name": "a"},
		{"v": 2.0, "name": "b"},
		{"v": 3.0, "name": "c"},
	}, WithColumnOrder("v", "name"))
	rolled, err := df.Rolling(2).Mean()
	if err != nil {
		t.Fatal(err)
	}
	// non-numeric column is dropped
	if got := rolled.Columns(); len(got) != 1 || got[0] != "v" {
		t.Fatalf("rolling columns = %v", got)
	}
	if v := colValues(t, rolled, "v"); v[0] != nil || v[1] != 1.5 || v[2] != 2.5 {
		t.Errorf("rolling mean = %v", v)
	}
}
