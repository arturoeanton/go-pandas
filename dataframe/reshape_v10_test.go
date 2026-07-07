package dataframe_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arturoeanton/go-pandas/dataframe"
	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/series"
)

func stackInput(t *testing.T) *dataframe.DataFrame {
	t.Helper()
	df, err := dataframe.DataFrameFromMap(map[string][]any{
		"a": {1.0, 2.0}, "b": {3.0, nil},
	}, dataframe.WithColumnOrder("a", "b"),
		dataframe.WithDataFrameIndex(index.NewStringIndex([]string{"x", "y"}, "k")))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

func TestStack(t *testing.T) {
	df := stackInput(t)
	before := fmt.Sprint(df.ToRows())

	s, err := df.Stack()
	if err != nil {
		t.Fatal(err)
	}
	if s.Len() != 4 {
		t.Fatalf("stack len = %d", s.Len())
	}
	mi, ok := s.Index().(*index.MultiIndex)
	if !ok || mi.NLevels() != 2 {
		t.Fatalf("stack index = %T", s.Index())
	}
	// Row-major layout, NA preserved (future-stack behavior).
	want := []any{1.0, 3.0, 2.0, nil}
	for i, w := range want {
		if s.Values()[i] != w {
			t.Fatalf("stack values = %v, want %v", s.Values(), want)
		}
	}
	if got := mi.Tuple(1); got[0] != "x" || got[1] != "b" {
		t.Fatalf("tuple(1) = %v", got)
	}
	// Homogeneous float columns stay typed.
	if s.DType() != dtype.Float64 {
		t.Fatalf("stack dtype = %v", s.DType())
	}
	// RangeIndex frames stack with positional labels.
	plain, _ := dataframe.DataFrameFromMap(map[string][]any{"a": {1.0}})
	ps, err := plain.Stack()
	if err != nil {
		t.Fatal(err)
	}
	if got := ps.Index().(*index.MultiIndex).Tuple(0); got[0] != 0 && got[0] != int64(0) {
		t.Fatalf("range stack tuple = %v", got)
	}
	// Display is stable and shows tuples.
	if !strings.Contains(s.Index().String(), "(x, a)") {
		t.Fatalf("stack index display: %s", s.Index().String())
	}
	if fmt.Sprint(df.ToRows()) != before {
		t.Fatal("Stack mutated the input")
	}
}

func TestStackMultiIndexAppendsLevel(t *testing.T) {
	df, _ := dataframe.DataFrameFromRecords([]map[string]any{
		{"c": "AR", "t": "BA", "v": 1.0},
		{"c": "BR", "t": "SP", "v": 2.0},
	}, dataframe.WithColumnOrder("c", "t", "v"))
	indexed, err := df.SetIndex("c", "t")
	if err != nil {
		t.Fatal(err)
	}
	s, err := indexed.Stack()
	if err != nil {
		t.Fatal(err)
	}
	mi := s.Index().(*index.MultiIndex)
	if mi.NLevels() != 3 {
		t.Fatalf("levels = %d", mi.NLevels())
	}
	if got := mi.Tuple(0); got[0] != "AR" || got[1] != "BA" || got[2] != "v" {
		t.Fatalf("tuple = %v", got)
	}
}

func TestUnstackRoundtripAndErrors(t *testing.T) {
	df := stackInput(t)
	s, err := df.Stack()
	if err != nil {
		t.Fatal(err)
	}
	back, err := dataframe.UnstackSeries(s)
	if err != nil {
		t.Fatal(err)
	}
	if got := back.Columns(); got[0] != "a" || got[1] != "b" {
		t.Fatalf("roundtrip columns = %v", got)
	}
	if v := back.MustCol("b").Values(); v[0] != 3.0 || v[1] != nil {
		t.Fatalf("roundtrip b = %v (NA must survive)", v)
	}
	if dt := back.DTypes()["a"]; dt != dtype.Float64 {
		t.Fatalf("roundtrip dtype = %v", dt)
	}

	// Flat index cannot unstack.
	if _, err := dataframe.UnstackSeries(series.FloatSeries("v", []float64{1})); !errors.Is(err, errs.ErrInvalidIndex) {
		t.Fatalf("flat unstack error = %v", err)
	}
	// Duplicate entries error.
	dup, _ := index.NewMultiIndexFromTuples([][]any{{"a", "x"}, {"a", "x"}}, nil)
	sd, _ := dataframe.DataFrameFromMap(map[string][]any{"v": {1.0, 2.0}},
		dataframe.WithDataFrameIndex(dup))
	if _, err := sd.Unstack(); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Fatalf("duplicate unstack error = %v", err)
	}
}

func TestPivotTableDepth(t *testing.T) {
	sales, _ := dataframe.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "month": "jan", "sales": 10.0, "qty": 1.0},
		{"country": "AR", "month": "feb", "sales": 20.0, "qty": 2.0},
		{"country": "BR", "month": "jan", "sales": 30.0, "qty": nil},
		{"country": "AR", "month": "jan", "sales": 40.0, "qty": 4.0},
	}, dataframe.WithColumnOrder("country", "month", "sales", "qty"))
	before := fmt.Sprint(sales.ToRows())

	// Historical 1x1x1 behavior unchanged.
	one, err := sales.PivotTable(dataframe.PivotTableOptions{
		Index: []string{"country"}, Columns: []string{"month"}, Values: []string{"sales"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := one.Columns(); got[0] != "country" || got[1] != "feb" || got[2] != "jan" {
		t.Fatalf("1x1 columns = %v", got)
	}

	// Multiple values + aggs + fill value.
	multi, err := sales.PivotTable(dataframe.PivotTableOptions{
		Values: []string{"sales", "qty"}, Index: []string{"country"},
		Columns: []string{"month"}, AggFuncs: []string{"sum", "count"},
		FillValue: 0.0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := multi.Columns()[1]; got != "sales_sum_feb" {
		t.Fatalf("naming rule: %v", multi.Columns())
	}
	// BR/feb bucket filled with 0.
	col, err := multi.Col("sales_sum_feb")
	if err != nil {
		t.Fatal(err)
	}
	if v := col.Values(); v[1] != 0.0 {
		t.Fatalf("fill value = %v", v)
	}

	// Multi-key index, no columns dim.
	mk, err := sales.PivotTable(dataframe.PivotTableOptions{
		Values: []string{"sales"}, Index: []string{"country", "month"}, AggFuncs: []string{"sum"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := mk.Columns(); len(got) != 3 || got[2] != "sales" {
		t.Fatalf("multi-key columns = %v", got)
	}

	if _, err := sales.PivotTable(dataframe.PivotTableOptions{
		Values: []string{"sales"}, Index: []string{"country"},
		Columns: []string{"month", "month"},
	}); !errors.Is(err, errs.ErrNotImplementedBase) {
		t.Fatalf("multi columns keys error = %v", err)
	}
	if _, err := sales.PivotTable(dataframe.PivotTableOptions{Values: []string{"sales"}}); err == nil {
		t.Fatal("missing index must error")
	}

	if fmt.Sprint(sales.ToRows()) != before {
		t.Fatal("PivotTable mutated the input")
	}
}

func TestGroupByTransform(t *testing.T) {
	size, err := series.CategoricalSeries("size", []string{"m", "s", "m", "s"})
	if err != nil {
		t.Fatal(err)
	}
	df, err := dataframe.NewDataFrame(
		size,
		series.StringSeries("k", []string{"a", "a", "b", "a"}),
		series.NewSeries("v", []any{1.0, 2.0, nil, 4.0}),
	)
	if err != nil {
		t.Fatal(err)
	}
	before := fmt.Sprint(df.ToRows())

	mean, err := df.GroupBy("k").Transform("v", "mean")
	if err != nil {
		t.Fatal(err)
	}
	if mean.Len() != df.Len() {
		t.Fatalf("transform len = %d", mean.Len())
	}
	// group a: (1+2+4)/3, group b: all NA -> NA.
	want := []any{7.0 / 3.0, 7.0 / 3.0, nil, 7.0 / 3.0}
	for i, w := range want {
		if mean.Values()[i] != w {
			t.Fatalf("transform mean = %v, want %v", mean.Values(), want)
		}
	}
	if mean.DType() != dtype.Float64 {
		t.Fatalf("transform dtype = %v", mean.DType())
	}

	cnt, err := df.GroupBy("size", "k").Transform("v", "count")
	if err != nil {
		t.Fatal(err)
	}
	if cnt.Values()[0] != 1 {
		t.Fatalf("multi-key categorical transform = %v", cnt.Values())
	}
	if _, err := df.GroupBy("k").Transform("v", "bogus"); err == nil {
		t.Fatal("unknown agg must error")
	}
	if fmt.Sprint(df.ToRows()) != before {
		t.Fatal("Transform mutated the input")
	}
}

func TestGroupByFilter(t *testing.T) {
	df, _ := dataframe.DataFrameFromRecords([]map[string]any{
		{"k": "a", "v": 1.0},
		{"k": "b", "v": 2.0},
		{"k": "a", "v": nil},
		{"k": "a", "v": 4.0},
	}, dataframe.WithColumnOrder("k", "v"))
	indexed, err := df.SetIndex("k")
	if err != nil {
		t.Fatal(err)
	}
	_ = indexed

	bySize, err := df.GroupBy("k").Filter(dataframe.GroupSize().Gt(1))
	if err != nil {
		t.Fatal(err)
	}
	if bySize.Len() != 3 {
		t.Fatalf("size filter rows = %d", bySize.Len())
	}
	// Row order preserved.
	if v := bySize.MustCol("v").Values(); v[0] != 1.0 || v[1] != nil || v[2] != 4.0 {
		t.Fatalf("size filter values = %v", v)
	}

	byCount, err := df.GroupBy("k").Filter(dataframe.GroupCount("v").Ge(2))
	if err != nil {
		t.Fatal(err)
	}
	if byCount.Len() != 3 {
		t.Fatalf("count filter rows = %d", byCount.Len())
	}
	none, err := df.GroupBy("k").Filter(dataframe.GroupCount("v").Ge(5))
	if err != nil {
		t.Fatal(err)
	}
	if none.Len() != 0 {
		t.Fatalf("empty filter rows = %d", none.Len())
	}
	if _, err := df.GroupBy("k").Filter(dataframe.GroupCond{}); err == nil {
		t.Fatal("empty condition must error")
	}
}
