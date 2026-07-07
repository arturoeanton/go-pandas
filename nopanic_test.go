package pandas_test

import (
	"fmt"
	"testing"
	"time"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/ndarray"
)

// TestNoPanicPublicAPIsInvalidInputs hammers public entry points with
// invalid, empty and degenerate inputs. Every case must return an error
// or a valid (possibly empty) result — never panic. Must* helpers are
// excluded: panicking is their documented contract.
func TestNoPanicPublicAPIsInvalidInputs(t *testing.T) {
	noPanic := func(name string, fn func()) {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("public API panicked: %v", r)
				}
			}()
			fn()
		})
	}

	empty := pd.NewSeries("e", nil)
	emptyDF, _ := pd.NewDataFrame(empty)
	df, _ := pd.DataFrameFromRecords([]map[string]any{
		{"k": "a", "v": 1.0, "s": "x"},
		{"k": "b", "v": 2.0, "s": "y"},
	}, pd.WithColumnOrder("k", "v", "s"))
	strArr := ndarray.ArrayString([]string{"a", "b"})
	numArr := pd.Array([]float64{1, 2, 3})

	noPanic("col not found", func() { _, _ = df.Col("nope") })
	noPanic("drop not found", func() { _, _ = df.Drop("nope") })
	noPanic("select not found", func() { _, _ = df.Select("nope") })
	noPanic("empty series ops", func() {
		_ = empty.Values()
		_, _ = empty.Mean()
		_ = empty.ValueCounts()
		_ = empty.SortValues(true)
		_ = empty.Unique()
	})
	noPanic("empty frame ops", func() {
		_ = emptyDF.Head(5)
		_ = emptyDF.Tail(5)
		_ = emptyDF.ResetIndex()
		_, _ = emptyDF.SortValues("e", true)
		_ = emptyDF.String()
	})
	noPanic("bad take positions", func() { _, _ = df.Take([]int{99}) })
	noPanic("bad iloc", func() { _, _ = df.ILoc().Rows(99).Get() })
	noPanic("bad loc label", func() { _, _ = df.Loc().Rows("nope").Get() })
	noPanic("loc tuple on flat index", func() { _, _ = df.Loc().Tuple("a", "b").Get() })
	noPanic("bad query syntax", func() { _, _ = df.Query("v >>> (((") })
	noPanic("query unknown column", func() { _, _ = df.Query("nope > 1") })
	noPanic("query empty", func() { _, _ = df.Query("") })
	noPanic("query type mismatch", func() { _, _ = df.Query(`v > "text" and s > 5`) })
	noPanic("bad astype", func() { _, _ = df.MustCol("s").Astype(pd.DType(-99)) })
	noPanic("bad datetime format", func() {
		_, _ = pd.ToDatetime(df.MustCol("s"), pd.WithDatetimeFormat("%Q"))
	})
	noPanic("bad datetime values raise", func() { _, _ = pd.ToDatetime(df.MustCol("s")) })
	noPanic("bad resample freq", func() {
		dates, _ := pd.ToDatetime(pd.StringSeries("d", []string{"2026-01-01", "2026-01-02"}))
		f, _ := pd.NewDataFrame(dates, pd.FloatSeries("v", []float64{1, 2}))
		fi, _ := f.SetIndex("d")
		_, _ = fi.Resample("5min").Sum()
	})
	noPanic("resample without datetime index", func() { _, _ = df.Resample("D").Sum() })
	noPanic("bad merge on", func() {
		_, _ = df.Merge(df, pd.MergeOptions{On: []string{"nope"}, How: "inner"})
	})
	noPanic("bad merge how", func() {
		_, _ = df.Merge(df, pd.MergeOptions{On: []string{"k"}, How: "sideways"})
	})
	noPanic("bad merge validate", func() {
		_, _ = df.Merge(df, pd.MergeOptions{On: []string{"k"}, How: "inner", Validate: "one_to_one"})
	})
	noPanic("bad categorical value", func() {
		_, _ = pd.CategoricalSeries("c", []string{"x"}, pd.WithCategories("a", "b"))
	})
	noPanic("cat accessor on numeric", func() { _, _ = df.MustCol("v").Cat() })
	noPanic("bad multiindex tuples", func() {
		_, _ = pd.MultiIndexFromTuples([]string{"a"}, []pd.Tuple{{1, 2}, {1}})
		_, _ = pd.MultiIndexFromTuples(nil, nil)
		_, _ = pd.NewMultiIndexFromArrays([][]any{{func() {}}}, nil)
	})
	noPanic("bad tuple arity loc", func() {
		mi, _ := df.SetIndex("k", "s")
		_, _ = mi.Loc().Tuple("a").Get()
		_, _ = mi.Loc().Tuple("a", "x", "extra").Get()
		_, _ = mi.Loc().TuplePrefix().Get()
	})
	noPanic("stack empty frame", func() {
		zero, _ := pd.DataFrameFromMap(map[string][]any{})
		if zero != nil {
			_, _ = zero.Stack()
		}
	})
	noPanic("unstack flat index", func() { _, _ = df.Unstack() })
	noPanic("bad pivot spec", func() {
		_, _ = df.PivotTable(pd.PivotTableOptions{})
		_, _ = df.PivotTable(pd.PivotTableOptions{Index: []string{"nope"}, Values: []string{"v"}})
		_, _ = df.PivotTable(pd.PivotTableOptions{Index: []string{"k"}, Values: []string{"v"}, AggFuncs: []string{"bogus"}})
	})
	noPanic("bad transform", func() {
		_, _ = df.GroupBy("k").Transform("nope", "mean")
		_, _ = df.GroupBy("k").Transform("v", "bogus")
		_, _ = df.GroupBy("nope").Transform("v", "mean")
	})
	noPanic("bad filter", func() {
		_, _ = df.GroupBy("k").Filter(pd.GroupCond{})
		_, _ = df.GroupBy("k").Filter(pd.GroupCount("nope").Gt(1))
	})
	noPanic("groupby unknown key", func() { _, _ = df.GroupBy("nope").Mean() })
	noPanic("bad ndarray take", func() {
		_, _ = numArr.Take([]int{5}, 0)
		_, _ = numArr.Take([]int{-1}, 0)
		_, _ = numArr.Take(nil, 9)
	})
	noPanic("bad searchsorted", func() {
		_, _ = numArr.SearchSorted([]float64{1}, "middle")
		_, _ = strArr.SearchSorted([]float64{1}, "left")
	})
	noPanic("isin degenerate", func() {
		_ = numArr.IsIn(nil)
		_ = strArr.IsIn([]any{1, nil, func() {}})
		_ = numArr.IsIn([]any{"text"})
	})
	noPanic("bad reshape", func() {
		_, _ = numArr.Reshape(2, 2)
		_, _ = numArr.Reshape(-1, -1)
	})
	noPanic("series eq unhashable", func() {
		_ = df.MustCol("s").Eq([]int{1})
		_ = df.MustCol("s").IsIn([]int{1})
	})
	noPanic("concat empty input", func() { _, _ = pd.Concat(nil) })
	noPanic("unstack NA column level", func() {
		mi, err := pd.MultiIndexFromTuples([]string{"a", "b"}, []pd.Tuple{{"x", nil}})
		if err == nil {
			f, _ := pd.DataFrameFromMap(map[string][]any{"v": {1.0}}, pd.WithDataFrameIndex(mi))
			_, _ = f.Unstack()
		}
	})
	noPanic("ndarray math on string backing", func() {
		_, _ = strArr.Add(numArr)
		_, _ = strArr.Sum(pd.Axis(0))
		_ = strArr.GtScalar(1)
		_, _ = strArr.Mean(pd.Axis(0))
		_ = pd.Sqrt(strArr)
	})
	noPanic("series arithmetic on strings", func() {
		_, _ = df.MustCol("s").AddScalar(1)
		_, _ = df.MustCol("s").Mean()
		_, _ = df.MustCol("s").Cumsum()
	})
	noPanic("datetime series to csv", func() {
		dates, _ := pd.ToDatetime(pd.NewSeries("d", []any{nil, time.Now()}))
		f, _ := pd.NewDataFrame(dates)
		_ = f.String()
		_ = fmt.Sprint(f.ToRows())
	})
}
