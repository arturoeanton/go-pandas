package dataframe

import (
	"testing"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/series"
)

// TestGroupByKeyDTypes groups by one key of every supported dtype.
func TestGroupByKeyDTypes(t *testing.T) {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	df, err := DataFrameFromMap(map[string][]any{
		"str":  {"b", "a", "b", "a"},
		"int":  {2, 1, 2, 1},
		"i64":  {int64(20), int64(10), int64(20), int64(10)},
		"f64":  {2.5, 1.5, 2.5, 1.5},
		"bool": {true, false, true, false},
		"time": {t0.AddDate(0, 1, 0), t0, t0.AddDate(0, 1, 0), t0},
		"v":    {10.0, 20.0, 30.0, 40.0},
	}, WithColumnOrder("str", "int", "i64", "f64", "bool", "time", "v"))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"str", "int", "i64", "f64", "bool", "time"} {
		out, err := df.GroupBy(key).Sum("v")
		if err != nil {
			t.Fatalf("groupby %s: %v", key, err)
		}
		if out.Len() != 2 {
			t.Fatalf("groupby %s groups = %d", key, out.Len())
		}
		// sorted ascending: group of rows {1,3} first (sum 60), then {0,2} (sum 40)
		sums := colValues(t, out, "v")
		if sums[0] != 60.0 || sums[1] != 40.0 {
			t.Errorf("groupby %s sums = %v", key, sums)
		}
		// key label column keeps its dtype
		keyDT := df.MustCol(key).StorageDType()
		if got := out.MustCol(key).StorageDType(); got != keyDT {
			t.Errorf("groupby %s key dtype = %v, want %v", key, got, keyDT)
		}
	}
}

func TestGroupByMultiKeyCombos(t *testing.T) {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	df, _ := DataFrameFromMap(map[string][]any{
		"c":  {"x", "x", "y", "y", "x"},
		"n":  {1, 2, 1, 1, 1},
		"tm": {t0, t0, t0, t0.AddDate(0, 0, 1), t0},
		"v":  {1.0, 2.0, 3.0, 4.0, 5.0},
	}, WithColumnOrder("c", "n", "tm", "v"))

	ss, err := df.GroupBy("c", "n").Sum("v")
	if err != nil {
		t.Fatal(err)
	}
	if ss.Len() != 3 { // (x,1) (x,2) (y,1)
		t.Fatalf("string+int groups = %d", ss.Len())
	}
	if v := colValues(t, ss, "v"); v[0] != 6.0 { // (x,1): rows 0,4
		t.Errorf("string+int sums = %v", v)
	}
	st, err := df.GroupBy("c", "tm").Size()
	if err != nil {
		t.Fatal(err)
	}
	if st.Len() != 3 { // (x,t0) (y,t0) (y,t0+1)
		t.Fatalf("string+time groups = %d", st.Len())
	}
}

func TestGroupByAggregationsTyped(t *testing.T) {
	df, _ := DataFrameFromMap(map[string][]any{
		"k": {"a", "a", "a", "b", "b"},
		"v": {1.0, nil, 3.0, 4.0, 6.0},
	}, WithColumnOrder("k", "v"))
	gb := func() *GroupBy { return df.GroupBy("k") }

	check := func(name string, got *DataFrame, err error, want []any) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		v := colValues(t, got, "v")
		for i := range want {
			if v[i] != want[i] {
				t.Errorf("%s = %v, want %v", name, v, want)
				return
			}
		}
	}
	sum, err := gb().Sum("v")
	check("sum", sum, err, []any{4.0, 10.0})
	mean, err := gb().Mean("v")
	check("mean", mean, err, []any{2.0, 5.0})
	median, err := gb().Median("v")
	check("median", median, err, []any{2.0, 5.0})
	min, err := gb().Min("v")
	check("min", min, err, []any{1.0, 4.0})
	max, err := gb().Max("v")
	check("max", max, err, []any{3.0, 6.0})
	va, err := gb().Var("v")
	check("var", va, err, []any{2.0, 2.0})
	count, err := gb().Count("v")
	check("count", count, err, []any{2, 2})
	size, err := gb().Size()
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, size, "size"); v[0] != 3 || v[1] != 2 {
		t.Errorf("size = %v (size counts NA rows, count does not)", v)
	}
	first, err := gb().First("v")
	check("first", first, err, []any{1.0, 4.0})
	last, err := gb().Last("v")
	check("last", last, err, []any{3.0, 6.0})
	nu, err := gb().NUnique("v")
	check("nunique", nu, err, []any{2, 2})
}

func TestGroupByTypedOutputStorage(t *testing.T) {
	df := sampleFrame(t)
	out, err := df.GroupBy("country").AggList(map[string][]string{
		"salary": {"mean", "max"},
		"age":    {"min"},
	})
	if err != nil {
		t.Fatal(err)
	}
	storage := out.StorageDTypes()
	if storage["country"] != dtype.String {
		t.Errorf("key storage = %v", storage["country"])
	}
	if storage["salary_mean"] != dtype.Float64 || storage["salary_max"] != dtype.Float64 {
		t.Errorf("salary storage = %v / %v", storage["salary_mean"], storage["salary_max"])
	}
	// min of an int column stays int (index-selector gather)
	if storage["age_min"] != dtype.Int {
		t.Errorf("age_min storage = %v", storage["age_min"])
	}
	for name := range storage {
		if out.MustCol(name).IsObjectBacked() {
			t.Errorf("column %s is object-backed", name)
		}
	}
}

func TestGroupByNAKeySortedLast(t *testing.T) {
	df, _ := DataFrameFromMap(map[string][]any{
		"k": {"b", nil, "a", nil},
		"v": {1.0, 2.0, 3.0, 4.0},
	}, WithColumnOrder("k", "v"))
	// dropNA=false + sorted: NA group last, like pandas dropna=False
	out, err := df.GroupByOpts([]GroupByOption{GroupDropNA(false)}, "k").Size()
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 3 {
		t.Fatalf("groups = %d", out.Len())
	}
	keys := colValues(t, out, "k")
	if keys[0] != "a" || keys[1] != "b" || keys[2] != nil {
		t.Errorf("sorted keys with NA last = %v", keys)
	}
	if v := colValues(t, out, "size"); v[2] != 2 {
		t.Errorf("NA group size = %v", v)
	}
	// dropNA=true drops those rows entirely
	dropped, err := df.GroupBy("k").Size()
	if err != nil {
		t.Fatal(err)
	}
	if dropped.Len() != 2 {
		t.Errorf("dropna groups = %d", dropped.Len())
	}
}

func TestGroupByObjectFallback(t *testing.T) {
	obj := series.NewSeries("k", []any{"a", "b", "a"}, series.WithDType(dtype.Object))
	vals := series.NewSeries("v", []any{1.0, 2.0, 3.0}, series.WithDType(dtype.Object))
	df, err := NewDataFrame(obj, vals)
	if err != nil {
		t.Fatal(err)
	}
	if !df.MustCol("k").IsObjectBacked() || !df.MustCol("v").IsObjectBacked() {
		t.Fatal("fixture should be object-backed")
	}
	out, err := df.GroupBy("k").Sum("v")
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 2 {
		t.Fatalf("object fallback groups = %d", out.Len())
	}
	if v := colValues(t, out, "v"); v[0] != 4.0 || v[1] != 2.0 {
		t.Errorf("object fallback sums = %v", v)
	}
}

func TestGroupByImmutability(t *testing.T) {
	df := sampleFrame(t)
	before := df.ToRows()
	if _, err := df.GroupBy("country").Mean("salary"); err != nil {
		t.Fatal(err)
	}
	if _, err := df.GroupBy("country", "name").Size(); err != nil {
		t.Fatal(err)
	}
	after := df.ToRows()
	for i := range before {
		for j := range before[i] {
			if before[i][j] != after[i][j] {
				t.Fatalf("groupby mutated input at [%d][%d]", i, j)
			}
		}
	}
	// mutating the result must not touch the source key column
	out, _ := df.GroupBy("country").Size()
	_ = out.MustCol("country").Set(0, "XX")
	if v, _ := df.MustCol("country").At(0); v == "XX" {
		t.Fatal("group label column aliases the source")
	}
}

func TestGroupByUnsupportedAgg(t *testing.T) {
	df := sampleFrame(t)
	if _, err := df.GroupBy("country").Agg(map[string]string{"salary": "wat"}); err == nil {
		t.Error("unknown aggregation should error")
	}
	// string column with numeric agg errors clearly
	if _, err := df.GroupBy("country").Agg(map[string]string{"name": "mean"}); err == nil {
		t.Error("mean of string column should error")
	}
}
