package dataframe

import (
	"testing"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/series"
)

func concatFixture(t *testing.T) (*DataFrame, *DataFrame) {
	t.Helper()
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	a, err := DataFrameFromMap(map[string][]any{
		"i": {1, nil}, "f": {1.5, 2.5}, "s": {"x", nil},
		"b": {true, false}, "t": {t0, nil},
	}, WithColumnOrder("i", "f", "s", "b", "t"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := DataFrameFromMap(map[string][]any{
		"i": {3, 4}, "f": {nil, 4.5}, "s": {"y", "z"},
		"b": {nil, true}, "t": {t0.AddDate(0, 1, 0), t0},
	}, WithColumnOrder("i", "f", "s", "b", "t"))
	if err != nil {
		t.Fatal(err)
	}
	return a, b
}

func TestConcatSameSchemaTypedAndMasks(t *testing.T) {
	a, b := concatFixture(t)
	out, err := Concat([]*DataFrame{a, b}, ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 4 {
		t.Fatalf("rows = %d", out.Len())
	}
	want := map[string]dtype.DType{
		"i": dtype.Int, "f": dtype.Float64, "s": dtype.String,
		"b": dtype.Bool, "t": dtype.Time,
	}
	storage := out.StorageDTypes()
	for name, dt := range want {
		if storage[name] != dt {
			t.Errorf("column %s storage = %v, want %v", name, storage[name], dt)
		}
		if out.MustCol(name).IsObjectBacked() {
			t.Errorf("column %s object-backed", name)
		}
	}
	// masks survive: i[1], f[2], s[1], b[2], t[1] are NA
	checks := map[string]int{"i": 1, "f": 2, "s": 1, "b": 2, "t": 1}
	for name, pos := range checks {
		if v, _ := out.MustCol(name).At(pos); v != nil {
			t.Errorf("column %s NA at %d lost: %v", name, pos, v)
		}
	}
}

func TestConcatNumericPromotion(t *testing.T) {
	ints, _ := DataFrameFromMap(map[string][]any{"v": {1, 2}})
	floats, _ := DataFrameFromMap(map[string][]any{"v": {2.5, nil}})
	i64s, _ := DataFrameFromMap(map[string][]any{"v": {int64(7)}})

	// int + float64 -> Float64
	out, err := Concat([]*DataFrame{ints, floats}, ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	if out.StorageDTypes()["v"] != dtype.Float64 {
		t.Errorf("int+float storage = %v", out.StorageDTypes()["v"])
	}
	if v := colValues(t, out, "v"); v[0] != 1.0 || v[2] != 2.5 || v[3] != nil {
		t.Errorf("promoted values = %v", v)
	}
	// int + int64 -> Int64
	out, err = Concat([]*DataFrame{ints, i64s}, ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	if out.StorageDTypes()["v"] != dtype.Int64 {
		t.Errorf("int+int64 storage = %v", out.StorageDTypes()["v"])
	}
	// bool + int -> Int
	bools, _ := DataFrameFromMap(map[string][]any{"v": {true, false}})
	out, err = Concat([]*DataFrame{bools, ints}, ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	if out.StorageDTypes()["v"] != dtype.Int {
		t.Errorf("bool+int storage = %v", out.StorageDTypes()["v"])
	}
	if v := colValues(t, out, "v"); v[0] != 1 || v[1] != 0 {
		t.Errorf("bool promoted values = %v", v)
	}
	// string + int -> Object (only that column)
	strs, _ := DataFrameFromMap(map[string][]any{"v": {"a"}, "ok": {9.5}}, WithColumnOrder("v", "ok"))
	nums, _ := DataFrameFromMap(map[string][]any{"v": {1}, "ok": {1.5}}, WithColumnOrder("v", "ok"))
	out, err = Concat([]*DataFrame{strs, nums}, ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	if !out.MustCol("v").IsObjectBacked() {
		t.Error("incompatible column should be object-backed")
	}
	if out.MustCol("ok").IsObjectBacked() || out.StorageDTypes()["ok"] != dtype.Float64 {
		t.Error("compatible sibling column degraded")
	}
	// time + string -> Object
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	times, _ := DataFrameFromMap(map[string][]any{"v": {t0}})
	out, err = Concat([]*DataFrame{times, strs.Head(1)}, ConcatJoin("inner"), ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	if !out.MustCol("v").IsObjectBacked() {
		t.Error("time+string should be object-backed")
	}
}

func TestConcatMissingColumnsTyped(t *testing.T) {
	a, _ := DataFrameFromMap(map[string][]any{"x": {1, 2}, "only_a": {1.5, 2.5}},
		WithColumnOrder("x", "only_a"))
	b, _ := DataFrameFromMap(map[string][]any{"x": {3}, "only_b": {"s"}},
		WithColumnOrder("x", "only_b"))
	out, err := Concat([]*DataFrame{a, b}, ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Columns(); len(got) != 3 || got[0] != "x" || got[1] != "only_a" || got[2] != "only_b" {
		t.Fatalf("union columns = %v", got)
	}
	// gaps stay typed with NA masks
	if out.StorageDTypes()["only_a"] != dtype.Float64 || out.StorageDTypes()["only_b"] != dtype.String {
		t.Errorf("gap columns degraded: %v", out.StorageDTypes())
	}
	if v := colValues(t, out, "only_a"); v[2] != nil {
		t.Errorf("gap value = %v", v[2])
	}
	if v := colValues(t, out, "only_b"); v[0] != nil || v[2] != "s" {
		t.Errorf("gap values = %v", v)
	}
	inner, err := Concat([]*DataFrame{a, b}, ConcatJoin("inner"), ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	if got := inner.Columns(); len(got) != 1 || got[0] != "x" {
		t.Errorf("inner columns = %v", got)
	}
}

func TestConcatIndexBehavior(t *testing.T) {
	a, _ := DataFrameFromMap(map[string][]any{"v": {1, 2}})
	b, _ := DataFrameFromMap(map[string][]any{"v": {3}})
	// preserve-index: integer labels stay a typed integer index
	out, err := Concat([]*DataFrame{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.Index().(*index.Int64Index); !ok {
		t.Errorf("integer label concat index = %T", out.Index())
	}
	if out.Index().At(2) != 0 { // b's first label
		t.Errorf("labels = %v", out.Index().Values())
	}
	// string indexes stay string
	sa, _ := DataFrameFromMap(map[string][]any{"v": {1}},
		WithDataFrameIndex(index.NewStringIndex([]string{"a"})))
	sb, _ := DataFrameFromMap(map[string][]any{"v": {2}},
		WithDataFrameIndex(index.NewStringIndex([]string{"b"})))
	out, err = Concat([]*DataFrame{sa, sb})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.Index().(*index.StringIndex); !ok {
		t.Errorf("string label concat index = %T", out.Index())
	}
	// ignore index -> RangeIndex
	out, err = Concat([]*DataFrame{a, b}, ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.Index().(*index.RangeIndex); !ok {
		t.Errorf("ignore-index concat index = %T", out.Index())
	}
}

func TestConcatImmutabilityAndAliasing(t *testing.T) {
	a, b := concatFixture(t)
	ab, bb := a.ToRows(), b.ToRows()
	out, err := Concat([]*DataFrame{a, b}, ConcatIgnoreIndex(true))
	if err != nil {
		t.Fatal(err)
	}
	// mutate the output; inputs must not change
	for _, name := range out.Columns() {
		_ = out.MustCol(name).Set(0, nil)
		_ = out.MustCol(name).Set(2, nil)
	}
	aa, ba := a.ToRows(), b.ToRows()
	for i := range ab {
		for j := range ab[i] {
			if ab[i][j] != aa[i][j] {
				t.Fatal("concat aliased first input")
			}
		}
	}
	for i := range bb {
		for j := range bb[i] {
			if bb[i][j] != ba[i][j] {
				t.Fatal("concat aliased second input")
			}
		}
	}
}

func TestConcatSeriesTyped(t *testing.T) {
	a := series.SeriesOf("v", []int{1, 2})
	b := series.SeriesOf("v", []int{3})
	out, err := series.Concat(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 3 || out.StorageDType() != dtype.Int || out.IsObjectBacked() {
		t.Fatalf("series concat: len=%d storage=%v", out.Len(), out.StorageDType())
	}
	promoted, err := series.Concat(a, series.FloatSeries("v", []float64{4.5}))
	if err != nil {
		t.Fatal(err)
	}
	if promoted.StorageDType() != dtype.Float64 {
		t.Errorf("series promotion = %v", promoted.StorageDType())
	}
	mixed, err := series.Concat(a, series.StringSeries("v", []string{"x"}))
	if err != nil {
		t.Fatal(err)
	}
	if !mixed.IsObjectBacked() {
		t.Error("incompatible series concat should be object-backed")
	}
	if _, err := series.Concat(); err == nil {
		t.Error("empty ConcatSeries should error")
	}
}
