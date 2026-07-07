package dataframe_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/arturoeanton/go-pandas/dataframe"
	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/series"
)

func miTestFrame(t *testing.T) *dataframe.DataFrame {
	t.Helper()
	df, err := dataframe.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "city": "BA", "year": 2023, "salary": 1000.0},
		{"country": "AR", "city": "CO", "year": 2023, "salary": 800.0},
		{"country": "BR", "city": "SP", "year": 2024, "salary": 1500.0},
		{"country": "AR", "city": "BA", "year": 2024, "salary": 1200.0},
	}, dataframe.WithColumnOrder("country", "city", "year", "salary"))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

func miIndexedFrame(t *testing.T) *dataframe.DataFrame {
	t.Helper()
	out, err := miTestFrame(t).SetIndex("country", "city")
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestSetIndexMultiColumn(t *testing.T) {
	df := miTestFrame(t)
	before := fmt.Sprint(df.ToRows())

	two, err := df.SetIndex("country", "city")
	if err != nil {
		t.Fatal(err)
	}
	mi, ok := two.Index().(*index.MultiIndex)
	if !ok {
		t.Fatalf("index = %T", two.Index())
	}
	if names := mi.Names(); names[0] != "country" || names[1] != "city" {
		t.Fatalf("names = %v", names)
	}
	if got := two.Columns(); len(got) != 2 || got[0] != "year" {
		t.Fatalf("index columns must be removed: %v", got)
	}
	// Duplicate tuples allowed.
	if got := mi.PositionsTuple([]any{"AR", "BA"}); len(got) != 2 {
		t.Fatalf("duplicate tuple positions = %v", got)
	}

	three, err := df.SetIndex("country", "city", "year")
	if err != nil {
		t.Fatal(err)
	}
	if three.Index().(*index.MultiIndex).NLevels() != 3 {
		t.Fatal("three-level index")
	}

	// Single column keeps the historical simple-index behavior.
	one, err := df.SetIndex("country")
	if err != nil {
		t.Fatal(err)
	}
	if _, isMI := one.Index().(*index.MultiIndex); isMI {
		t.Fatal("single-column SetIndex must not build a MultiIndex")
	}

	if fmt.Sprint(df.ToRows()) != before || len(df.Columns()) != 4 {
		t.Fatal("SetIndex mutated the input frame")
	}
}

func TestSetIndexWithNAAndCategorical(t *testing.T) {
	size, err := series.CategoricalSeries("size", []string{"m", "s", "m"})
	if err != nil {
		t.Fatal(err)
	}
	df, err := dataframe.NewDataFrame(
		series.NewSeries("k", []any{"a", nil, "b"}),
		size,
		series.FloatSeries("v", []float64{1, 2, 3}),
	)
	if err != nil {
		t.Fatal(err)
	}
	indexed, err := df.SetIndex("k", "size")
	if err != nil {
		t.Fatal(err)
	}
	mi := indexed.Index().(*index.MultiIndex)
	if !mi.IsNA(1, 0) {
		t.Fatal("NA key must be an NA tuple component")
	}
	// Categorical column contributes its labels as level values.
	if lv := mi.Levels()[1]; lv[0] != "m" || lv[1] != "s" {
		t.Fatalf("categorical level = %v", lv)
	}
	if got := mi.Tuple(1); got[0] != nil || got[1] != "s" {
		t.Fatalf("tuple(1) = %v", got)
	}
}

func TestResetIndexMultiIndex(t *testing.T) {
	indexed := miIndexedFrame(t)
	before := fmt.Sprint(indexed.ToRows())

	back := indexed.ResetIndex()
	if got := back.Columns(); got[0] != "country" || got[1] != "city" || got[2] != "year" {
		t.Fatalf("reset columns = %v", got)
	}
	if _, isRange := back.Index().(*index.RangeIndex); !isRange {
		t.Fatalf("reset index = %T", back.Index())
	}
	// Level dtypes restore typed columns.
	if dt := back.DTypes()["country"]; dt != dtype.String {
		t.Fatalf("country dtype = %v", dt)
	}
	if v := back.MustCol("country").Values(); v[3] != "AR" {
		t.Fatalf("country values = %v", v)
	}
	if fmt.Sprint(indexed.ToRows()) != before {
		t.Fatal("ResetIndex mutated the input")
	}

	// Roundtrip equals the original data frame ordering.
	orig := miTestFrame(t)
	if fmt.Sprint(back.ToRows()) != fmt.Sprint(orig.ToRows()) {
		t.Fatalf("roundtrip rows differ:\n%v\n%v", back.ToRows(), orig.ToRows())
	}
}

func TestResetIndexUnnamedLevelsAndNA(t *testing.T) {
	mi, err := index.NewMultiIndexFromTuples([][]any{{"a", 1}, {nil, 2}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	indexed, err := dataframe.DataFrameFromMap(
		map[string][]any{"v": {1.0, 2.0}},
		dataframe.WithDataFrameIndex(mi),
	)
	if err != nil {
		t.Fatal(err)
	}
	back := indexed.ResetIndex()
	if got := back.Columns(); got[0] != "level_0" || got[1] != "level_1" {
		t.Fatalf("unnamed level columns = %v", got)
	}
	if v := back.MustCol("level_0").Values(); v[1] != nil {
		t.Fatalf("NA component must reset as NA: %v", v)
	}
}

func TestLocTuple(t *testing.T) {
	indexed := miIndexedFrame(t)
	before := fmt.Sprint(indexed.ToRows())

	// Full tuple, duplicate rows.
	rows, err := indexed.Loc().Tuple("AR", "BA").Get()
	if err != nil {
		t.Fatal(err)
	}
	if rows.Len() != 2 {
		t.Fatalf("full tuple rows = %d", rows.Len())
	}
	if _, ok := rows.Index().(*index.MultiIndex); !ok {
		t.Fatal("Loc result must keep the MultiIndex")
	}
	if dt := rows.DTypes()["salary"]; dt != dtype.Float64 {
		t.Fatalf("salary dtype = %v", dt)
	}

	// pd.Tuple form.
	rows2, err := indexed.Loc().Tuple(index.Tuple{"BR", "SP"}).Get()
	if err != nil || rows2.Len() != 1 {
		t.Fatalf("tuple arg form: %v len=%d", err, rows2.Len())
	}

	// Unknown tuple errors like unknown labels do.
	if _, err := indexed.Loc().Tuple("XX", "BA").Get(); err == nil {
		t.Fatal("unknown tuple must error")
	}

	// Prefix.
	prefix, err := indexed.Loc().TuplePrefix("AR").Get()
	if err != nil || prefix.Len() != 3 {
		t.Fatalf("prefix: %v len=%d", err, prefix.Len())
	}
	if _, err := indexed.Loc().TuplePrefix("XX").Get(); err == nil {
		t.Fatal("unknown prefix must error")
	}

	// Tuple selection needs a MultiIndex.
	if _, err := miTestFrame(t).Loc().Tuple("AR", "BA").Get(); err == nil {
		t.Fatal("tuple Loc on a flat index must error")
	}

	if fmt.Sprint(indexed.ToRows()) != before {
		t.Fatal("Loc mutated the input")
	}
}

func TestEnginesPreserveMultiIndex(t *testing.T) {
	indexed := miIndexedFrame(t)

	where, err := indexed.Where(expr.Col("salary").Gt(900.0))
	if err != nil {
		t.Fatal(err)
	}
	wmi, ok := where.Index().(*index.MultiIndex)
	if !ok || where.Len() != 3 {
		t.Fatalf("where index = %T len=%d", where.Index(), where.Len())
	}
	if got := wmi.Tuple(0); got[0] != "AR" || got[1] != "BA" {
		t.Fatalf("where tuple 0 = %v", got)
	}

	taken, err := indexed.Take([]int{2, 0})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := taken.Index().(*index.MultiIndex); !ok {
		t.Fatal("Take must preserve MultiIndex")
	}
	if _, ok := indexed.Head(2).Index().(*index.MultiIndex); !ok {
		t.Fatal("Head must preserve MultiIndex")
	}
	if _, ok := indexed.Tail(2).Index().(*index.MultiIndex); !ok {
		t.Fatal("Tail must preserve MultiIndex")
	}
}

func TestConcatPreservesMultiIndex(t *testing.T) {
	indexed := miIndexedFrame(t)
	out, err := dataframe.Concat([]*dataframe.DataFrame{indexed, indexed})
	if err != nil {
		t.Fatal(err)
	}
	mi, ok := out.Index().(*index.MultiIndex)
	if !ok {
		t.Fatalf("concat index = %T", out.Index())
	}
	if mi.Len() != 8 || mi.NLevels() != 2 {
		t.Fatalf("concat index shape: %d x %d", mi.Len(), mi.NLevels())
	}
	if got := mi.Tuple(4); got[0] != "AR" || got[1] != "BA" {
		t.Fatalf("second half tuple = %v", got)
	}
}

func TestJoinByMultiIndex(t *testing.T) {
	l, _ := dataframe.DataFrameFromRecords([]map[string]any{
		{"a": "x", "b": 1, "v": 1.0},
		{"a": "y", "b": 2, "v": 2.0},
	}, dataframe.WithColumnOrder("a", "b", "v"))
	r, _ := dataframe.DataFrameFromRecords([]map[string]any{
		{"a": "x", "b": 1, "w": 10.0},
	}, dataframe.WithColumnOrder("a", "b", "w"))
	li, err := l.SetIndex("a", "b")
	if err != nil {
		t.Fatal(err)
	}
	ri, err := r.SetIndex("a", "b")
	if err != nil {
		t.Fatal(err)
	}
	// v0.8: index joins align MultiIndexes through boxed tuple keys
	// (documented; no typed fast path yet).
	out, err := li.Join(ri, dataframe.JoinOptions{How: "left"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 2 {
		t.Fatalf("join rows = %d", out.Len())
	}
	w := out.MustCol("w").Values()
	if w[0] != 10.0 || w[1] != nil {
		t.Fatalf("join values = %v", w)
	}
}

func TestGroupByAsIndex(t *testing.T) {
	df := miTestFrame(t)

	// Default unchanged: keys stay columns over a RangeIndex.
	def, err := df.GroupBy("country", "city").Mean("salary")
	if err != nil {
		t.Fatal(err)
	}
	if got := def.Columns(); len(got) != 3 || got[0] != "country" {
		t.Fatalf("default columns = %v", got)
	}
	if _, isRange := def.Index().(*index.RangeIndex); !isRange {
		t.Fatalf("default index = %T", def.Index())
	}

	// AsIndex: group keys become a MultiIndex, not columns.
	g, err := df.GroupBy("country", "city").AsIndex(true).Mean("salary")
	if err != nil {
		t.Fatal(err)
	}
	if got := g.Columns(); len(got) != 1 || got[0] != "salary" {
		t.Fatalf("asindex columns = %v", got)
	}
	mi, ok := g.Index().(*index.MultiIndex)
	if !ok {
		t.Fatalf("asindex index = %T", g.Index())
	}
	if names := mi.Names(); names[0] != "country" || names[1] != "city" {
		t.Fatalf("index names = %v", names)
	}
	if got := mi.Tuple(0); got[0] != "AR" || got[1] != "BA" {
		t.Fatalf("first group tuple = %v", got)
	}

	// Roundtrip through ResetIndex equals the default output.
	back := g.ResetIndex()
	if fmt.Sprint(back.ToRows()) != fmt.Sprint(def.ToRows()) {
		t.Fatalf("roundtrip mismatch:\n%v\n%v", back.ToRows(), def.ToRows())
	}

	// Single key: plain index, not MultiIndex.
	one, err := df.GroupBy("country").AsIndex(true).Mean("salary")
	if err != nil {
		t.Fatal(err)
	}
	if _, isMI := one.Index().(*index.MultiIndex); isMI {
		t.Fatal("single-key AsIndex must use a plain index")
	}
	if one.Index().At(0) != "AR" {
		t.Fatalf("single-key label = %v", one.Index().At(0))
	}

	// Size() honors AsIndex too.
	sz, err := df.GroupBy("country", "city").AsIndex(true).Size()
	if err != nil {
		t.Fatal(err)
	}
	if got := sz.Columns(); len(got) != 1 || got[0] != "size" {
		t.Fatalf("size columns = %v", got)
	}
}

func TestGroupByAsIndexCategoricalAndNA(t *testing.T) {
	size, err := series.CategoricalSeries("size", []string{"m", "s", "m", "s"})
	if err != nil {
		t.Fatal(err)
	}
	df, err := dataframe.NewDataFrame(
		size,
		series.NewSeries("k", []any{"a", "a", nil, "b"}),
		series.FloatSeries("v", []float64{1, 2, 3, 4}),
	)
	if err != nil {
		t.Fatal(err)
	}
	g, err := df.GroupByOpts([]dataframe.GroupByOption{
		dataframe.GroupAsIndex(true), dataframe.GroupDropNA(false),
	}, "size", "k").Mean("v")
	if err != nil {
		t.Fatal(err)
	}
	mi, ok := g.Index().(*index.MultiIndex)
	if !ok {
		t.Fatalf("index = %T", g.Index())
	}
	// The NA key group keeps an NA tuple component.
	foundNA := false
	for i := 0; i < mi.Len(); i++ {
		if mi.IsNA(i, 1) {
			foundNA = true
		}
	}
	if !foundNA {
		t.Fatalf("NA key group missing: %v", mi.Tuples())
	}
}

func TestDataFrameStringWithMultiIndex(t *testing.T) {
	out := miIndexedFrame(t).String()
	if !strings.Contains(out, "(AR, BA)") || !strings.Contains(out, "(BR, SP)") {
		t.Fatalf("display missing tuple labels:\n%s", out)
	}
	// Duplicate tuple rows both display.
	if strings.Count(out, "(AR, BA)") != 2 {
		t.Fatalf("duplicate tuples must both display:\n%s", out)
	}
}
