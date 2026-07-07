package dataframe

import (
	"errors"
	"math"
	"testing"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/ndarray"
)

func TestReindexAndColumns(t *testing.T) {
	df, err := DataFrameFromMap(map[string][]any{"v": {1, 2}},
		WithDataFrameIndex(index.NewStringIndex([]string{"a", "b"})))
	if err != nil {
		t.Fatal(err)
	}
	out, err := df.Reindex(index.NewStringIndex([]string{"b", "z"}))
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, out, "v"); v[0] != 2 || v[1] != nil {
		t.Errorf("reindex = %v", v)
	}
	wide, err := df.ReindexColumns("v", "extra")
	if err != nil {
		t.Fatal(err)
	}
	if got := wide.Columns(); len(got) != 2 || got[1] != "extra" {
		t.Fatalf("reindex columns = %v", got)
	}
	if v := colValues(t, wide, "extra"); v[0] != nil {
		t.Errorf("new column should be NA, got %v", v)
	}
}

func TestValueCountsAndCov(t *testing.T) {
	df, _ := DataFrameFromRecords([]map[string]any{
		{"k": "a"}, {"k": "a"}, {"k": "b"},
	}, WithColumnOrder("k"))
	vc, err := df.ValueCounts("k")
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, vc, "count"); v[0] != 2 {
		t.Errorf("value counts = %v", v)
	}
	num, _ := DataFrameFromMap(map[string][]any{
		"x": {1.0, 2.0, 3.0},
		"y": {2.0, 4.0, 6.0},
	}, WithColumnOrder("x", "y"))
	cov, err := num.Cov()
	if err != nil {
		t.Fatal(err)
	}
	// cov(x, y) = 2 * var(x) = 2
	if v := colValues(t, cov, "y"); v[0] != 2.0 {
		t.Errorf("cov = %v", v)
	}
	corr, err := num.Corr()
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, corr, "y"); math.Abs(v[0].(float64)-1) > 1e-9 {
		t.Errorf("corr = %v", v)
	}
}

func TestAstypeMapAndSelectDTypes(t *testing.T) {
	df, _ := DataFrameFromRecords([]map[string]any{
		{"a": "1", "b": 2.5, "c": "x"},
	}, WithColumnOrder("a", "b", "c"))
	converted, err := df.Astype(map[string]dtype.DType{"a": dtype.Int64})
	if err != nil {
		t.Fatal(err)
	}
	if converted.DTypes()["a"] != dtype.Int64 {
		t.Errorf("astype dtypes = %v", converted.DTypes())
	}
	if _, err := df.Astype(map[string]dtype.DType{"c": dtype.Int64}); err == nil {
		t.Error("astype invalid should fail")
	}
	excluded, err := df.SelectDTypes(Exclude(dtype.String))
	if err != nil {
		t.Fatal(err)
	}
	if got := excluded.Columns(); len(got) != 1 || got[0] != "b" {
		t.Errorf("exclude string = %v", got)
	}
	if _, err := df.SelectDTypes(); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Errorf("empty SelectDTypes error = %v", err)
	}
}

func TestDropNAColumnsAndReplaceNA(t *testing.T) {
	df, _ := DataFrameFromMap(map[string][]any{
		"full":  {1, 2},
		"holey": {1, nil},
	}, WithColumnOrder("full", "holey"))
	dropped := df.DropNA(DropNAAxis(1))
	if got := dropped.Columns(); len(got) != 1 || got[0] != "full" {
		t.Errorf("dropna axis=1 = %v", got)
	}
	replaced := df.ReplaceNA(0)
	if replaced.HasNA() {
		t.Error("ReplaceNA left missing values")
	}
}

func TestILocMixedAndLabelSlice(t *testing.T) {
	df := sampleFrame(t)
	mixed, err := df.ILoc().Rows(0, ndarray.Slice(1, 3)).Cols(1).Get()
	if err != nil {
		t.Fatal(err)
	}
	if mixed.Len() != 3 {
		t.Fatalf("mixed rows = %d", mixed.Len())
	}
	if v := colValues(t, mixed, "name"); v[0] != "Ana" || v[2] != "Joao" {
		t.Errorf("mixed = %v", v)
	}
	labeled, err := DataFrameFromMap(map[string][]any{"v": {1, 2, 3}},
		WithDataFrameIndex(index.NewStringIndex([]string{"a", "b", "c"})))
	if err != nil {
		t.Fatal(err)
	}
	// inclusive label slice, like pandas .loc["a":"b"]
	sliced, err := labeled.Loc().Rows(LabelSlice("a", "b")).Get()
	if err != nil {
		t.Fatal(err)
	}
	if sliced.Len() != 2 {
		t.Errorf("label slice len = %d", sliced.Len())
	}
}

func TestSetIndexMultiAndResetIndex(t *testing.T) {
	df := sampleFrame(t)
	// v0.8: multi-column SetIndex builds a real MultiIndex.
	multi, err := df.SetIndex("country", "name")
	if err != nil {
		t.Fatalf("multi SetIndex error = %v", err)
	}
	if _, ok := multi.Index().(*index.MultiIndex); !ok {
		t.Fatalf("multi SetIndex index = %T, want *index.MultiIndex", multi.Index())
	}
	indexed, err := df.SetIndex("name")
	if err != nil {
		t.Fatal(err)
	}
	back := indexed.ResetIndex()
	if got := back.Columns(); got[0] != "name" {
		t.Errorf("reset index should re-insert the label column, got %v", got)
	}
	if v := colValues(t, back, "name"); v[0] != "Ana" {
		t.Errorf("reset index labels = %v", v)
	}
}

func TestExpandingDataFrame(t *testing.T) {
	df, _ := DataFrameFromMap(map[string][]any{"v": {1.0, 2.0, 3.0}})
	out, err := df.Expanding().Sum()
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, out, "v"); v[2] != 6.0 {
		t.Errorf("expanding sum = %v", v)
	}
	mean, err := df.Expanding(2).Mean()
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, mean, "v"); v[0] != nil || v[1] != 1.5 {
		t.Errorf("expanding mean = %v", v)
	}
}

func TestDuplicatedSubset(t *testing.T) {
	df, _ := DataFrameFromRecords([]map[string]any{
		{"k": "a", "v": 1},
		{"k": "a", "v": 2},
	}, WithColumnOrder("k", "v"))
	dup, err := df.Duplicated("k")
	if err != nil {
		t.Fatal(err)
	}
	if got := dup.AsMask(); got[0] || !got[1] {
		t.Errorf("duplicated subset = %v", got)
	}
	unique, err := df.DropDuplicates("k")
	if err != nil {
		t.Fatal(err)
	}
	if unique.Len() != 1 {
		t.Errorf("drop duplicates subset len = %d", unique.Len())
	}
}
