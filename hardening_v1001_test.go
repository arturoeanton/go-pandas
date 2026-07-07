package pandas_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/internal/checks"
)

// hardFrame builds one frame covering every dtype family, with NAs.
func hardFrame(t *testing.T) *pd.DataFrame {
	t.Helper()
	cat, err := pd.CategoricalSeries("cat", []string{"m", "s", "m", "l", "s", "m"})
	if err != nil {
		t.Fatal(err)
	}
	dates, err := pd.ToDatetime(pd.NewSeries("when", []any{
		"2026-01-03", "2026-01-01", nil, "2026-01-02", "2026-01-01", "2026-01-04",
	}), pd.WithDatetimeErrors("coerce"))
	if err != nil {
		t.Fatal(err)
	}
	df, err := pd.NewDataFrame(
		pd.StringSeries("k", []string{"a", "b", "a", "c", "b", "a"}),
		pd.NewSeries("f", []any{1.5, nil, 3.5, 4.5, 5.5, 6.5}),
		pd.IntSeries("i", []int{10, 20, 30, 40, 50, 60}),
		pd.BoolSeries("b", []bool{true, false, true, false, true, false}),
		cat,
		dates,
	)
	if err != nil {
		t.Fatal(err)
	}
	return df
}

// TestDTypePreservationTakeWhereQuery: row-selection operations keep
// every column dtype and the frame invariants.
func TestDTypePreservationTakeWhereQuery(t *testing.T) {
	df := hardFrame(t)
	want := df.DTypes()
	ops := map[string]func() (*pd.DataFrame, error){
		"take":  func() (*pd.DataFrame, error) { return df.Take([]int{5, 0, 0, 3}) },
		"where": func() (*pd.DataFrame, error) { return df.Where(pd.Col("i").Gt(15)) },
		"query": func() (*pd.DataFrame, error) { return df.Query("i > 15 and k != \"c\"") },
		"sort":  func() (*pd.DataFrame, error) { return df.SortValues("i", false) },
		"head":  func() (*pd.DataFrame, error) { return df.Head(3), nil },
		"dropna": func() (*pd.DataFrame, error) {
			return df.DropNA(), nil
		},
	}
	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			out, err := op()
			if err != nil {
				t.Fatal(err)
			}
			checks.RequireValidDataFrame(t, out)
			for col, dt := range out.DTypes() {
				if dt != want[col] {
					t.Fatalf("%s: column %q dtype %v -> %v", name, col, want[col], dt)
				}
			}
		})
	}
}

// TestDTypePreservationConcatMerge: same-schema concat and merges keep
// dtypes; fallbacks are column-local.
func TestDTypePreservationConcatMerge(t *testing.T) {
	df := hardFrame(t)
	want := df.DTypes()

	cc, err := pd.Concat([]*pd.DataFrame{df, df}, pd.IgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	checks.RequireValidDataFrame(t, cc)
	for col, dt := range cc.DTypes() {
		if dt != want[col] {
			t.Fatalf("concat: column %q dtype %v -> %v", col, want[col], dt)
		}
	}

	right, err := pd.NewDataFrame(
		pd.StringSeries("k", []string{"a", "b"}),
		pd.FloatSeries("extra", []float64{1, 2}),
	)
	if err != nil {
		t.Fatal(err)
	}
	merged, err := df.Merge(right, pd.MergeOptions{On: []string{"k"}, How: "left"})
	if err != nil {
		t.Fatal(err)
	}
	checks.RequireValidDataFrame(t, merged)
	for col, dt := range want {
		if merged.DTypes()[col] != dt {
			t.Fatalf("merge: column %q dtype %v -> %v", col, dt, merged.DTypes()[col])
		}
	}
}

// TestDTypePreservationGroupByTransformFilter: transform/filter keep
// value dtypes and (for filter) the whole schema.
func TestDTypePreservationGroupByTransformFilter(t *testing.T) {
	df := hardFrame(t)
	tr, err := df.GroupBy("k").Transform("i", "max")
	if err != nil {
		t.Fatal(err)
	}
	checks.RequireValidSeries(t, tr)
	if tr.DType() != pd.Int {
		t.Fatalf("transform max dtype = %v (int expected via first/last-style gather)", tr.DType())
	}
	fl, err := df.GroupBy("k").Filter(pd.GroupSize().Ge(2))
	if err != nil {
		t.Fatal(err)
	}
	checks.RequireValidDataFrame(t, fl)
	for col, dt := range fl.DTypes() {
		if dt != df.DTypes()[col] {
			t.Fatalf("filter: column %q dtype changed to %v", col, dt)
		}
	}
}

// TestDTypePreservationPivotStackUnstack: reshape keeps typed values
// for homogeneous inputs.
func TestDTypePreservationPivotStackUnstack(t *testing.T) {
	df := hardFrame(t)
	numeric, err := df.Select("f", "i")
	if err != nil {
		t.Fatal(err)
	}
	s, err := numeric.Stack()
	if err != nil {
		t.Fatal(err)
	}
	checks.RequireValidSeries(t, s)
	if s.DType() != pd.Float64 {
		t.Fatalf("stack of float+int should promote to float64 storage via Infer, got %v", s.DType())
	}
	back, err := pd.UnstackSeries(s)
	if err != nil {
		t.Fatal(err)
	}
	checks.RequireValidDataFrame(t, back)
	if dt := back.DTypes()["f"]; dt != pd.Float64 {
		t.Fatalf("unstack f dtype = %v", dt)
	}
	pt, err := df.PivotTable(pd.PivotTableOptions{
		Index: []string{"k"}, Values: []string{"f"}, AggFuncs: []string{"sum", "count"},
	})
	if err != nil {
		t.Fatal(err)
	}
	checks.RequireValidDataFrame(t, pt)
	if dt := pt.DTypes()["sum"]; dt != pd.Float64 {
		t.Fatalf("pivot sum dtype = %v", dt)
	}
	if dt := pt.DTypes()["count"]; dt != pd.Int {
		t.Fatalf("pivot count dtype = %v", dt)
	}
}

// TestDTypePreservationResample: resample aggregations keep numeric
// dtypes; first/last keep the source dtype.
func TestDTypePreservationResample(t *testing.T) {
	dates, _ := pd.ToDatetime(pd.StringSeries("d", []string{
		"2026-01-01 01:00:00", "2026-01-01 02:00:00", "2026-01-02 01:00:00",
	}))
	df, _ := pd.NewDataFrame(dates,
		pd.FloatSeries("f", []float64{1, 2, 3}),
		pd.StringSeries("s", []string{"x", "y", "z"}))
	indexed, err := df.SetIndex("d")
	if err != nil {
		t.Fatal(err)
	}
	sum, err := indexed.Resample("D").Sum()
	if err != nil {
		t.Fatal(err)
	}
	checks.RequireValidDataFrame(t, sum)
	if dt := sum.DTypes()["f"]; dt != pd.Float64 {
		t.Fatalf("resample sum dtype = %v", dt)
	}
	first, err := indexed.Resample("D").First()
	if err != nil {
		t.Fatal(err)
	}
	if dt := first.DTypes()["s"]; dt != pd.String {
		t.Fatalf("resample first string dtype = %v", dt)
	}
}

// TestIndexPreservationRowOps: every row-changing operation keeps the
// index aligned with the surviving rows, per index type.
func TestIndexPreservationRowOps(t *testing.T) {
	build := func(idx index.Index) *pd.DataFrame {
		df, err := pd.DataFrameFromMap(map[string][]any{
			"v": {10.0, 20.0, 30.0, 40.0},
		}, pd.WithDataFrameIndex(idx))
		if err != nil {
			t.Fatal(err)
		}
		return df
	}
	mi, _ := pd.MultiIndexFromTuples([]string{"a", "b"},
		[]pd.Tuple{{"x", 1}, {"x", 2}, {"y", 1}, {"y", 2}})
	indexes := map[string]index.Index{
		"range":    index.NewRangeIndex(4),
		"string":   index.NewStringIndex([]string{"p", "q", "r", "s"}, ""),
		"int64":    index.NewInt64Index([]int64{7, 8, 9, 10}, ""),
		"datetime": index.NewDatetimeIndex([]time.Time{day(1), day(2), day(3), day(4)}, ""),
		"multi":    mi,
	}
	for name, idx := range indexes {
		t.Run(name, func(t *testing.T) {
			df := build(idx)
			// Filter keeps the selected labels aligned.
			out, err := df.Where(expr.Col("v").Gt(15.0))
			if err != nil {
				t.Fatal(err)
			}
			checks.RequireValidDataFrame(t, out)
			if out.Len() != 3 {
				t.Fatalf("rows = %d", out.Len())
			}
			if got, want := fmt.Sprint(out.Index().At(0)), fmt.Sprint(idx.At(1)); got != want {
				t.Fatalf("filtered label = %v, want %v", got, want)
			}
			// Sort reorders labels with rows.
			sorted, err := df.SortValues("v", false)
			if err != nil {
				t.Fatal(err)
			}
			checks.RequireValidDataFrame(t, sorted)
			if got, want := fmt.Sprint(sorted.Index().At(0)), fmt.Sprint(idx.At(3)); got != want {
				t.Fatalf("sorted label = %v, want %v", got, want)
			}
			// Take with repeated positions.
			taken, err := df.Take([]int{2, 2, 0})
			if err != nil {
				t.Fatal(err)
			}
			checks.RequireValidDataFrame(t, taken)
			if got, want := fmt.Sprint(taken.Index().At(1)), fmt.Sprint(idx.At(2)); got != want {
				t.Fatalf("taken label = %v, want %v", got, want)
			}
			// Transform preserves the original index verbatim.
			tr, err := df.GroupBy("v").Transform("v", "sum")
			if err != nil {
				t.Fatal(err)
			}
			if !tr.Index().Equals(df.Index()) {
				t.Fatal("transform changed the index")
			}
		})
	}
}

// TestCopyAliasing: Copy/Take/Slice never alias mutable buffers.
func TestCopyAliasing(t *testing.T) {
	df := hardFrame(t)
	orig := fmt.Sprint(df.ToRows())

	cp := df.Copy()
	if err := cp.MustCol("i").Set(0, 999); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(df.ToRows()) != orig {
		t.Fatal("Copy aliases column data")
	}
	taken, _ := df.Take([]int{0, 1})
	if err := taken.MustCol("i").Set(0, 888); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(df.ToRows()) != orig {
		t.Fatal("Take aliases column data")
	}
	sliced, _ := df.Slice(0, 2)
	if err := sliced.MustCol("i").Set(0, 777); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(df.ToRows()) != orig {
		t.Fatal("Slice aliases column data")
	}
}

// TestConcurrentSharedLookups: shared categorical and MultiIndex lookup
// structures are race-safe (run with -race).
func TestConcurrentSharedLookups(t *testing.T) {
	cat, err := pd.CategoricalSeries("c", []string{"a", "b", "c", "a"})
	if err != nil {
		t.Fatal(err)
	}
	mi, err := pd.MultiIndexFromTuples([]string{"x", "y"},
		[]pd.Tuple{{"a", 1}, {"b", 2}, {"a", 2}})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			acc, err := cat.Cat()
			if err != nil {
				t.Error(err)
				return
			}
			_ = acc.Codes()
			_ = cat.Eq("b")
			_ = mi.PositionsTuple([]any{"a", 1})
			_ = mi.PositionsPrefix([]any{"b"})
		}(i)
	}
	wg.Wait()
}

func day(d int) time.Time { return time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC) }
