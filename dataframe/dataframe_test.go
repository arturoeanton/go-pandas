package dataframe

import (
	"errors"
	"strings"
	"testing"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/ndarray"
	"github.com/arturoeanton/go-pandas/series"
)

func sampleFrame(t *testing.T) *DataFrame {
	t.Helper()
	df, err := DataFrameFromRecords([]map[string]any{
		{"country": "AR", "name": "Ana", "age": 30, "salary": 1000.0},
		{"country": "AR", "name": "Luis", "age": 40, "salary": 2000.0},
		{"country": "BR", "name": "Joao", "age": 35, "salary": 1500.0},
	}, WithColumnOrder("country", "name", "age", "salary"))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

func colValues(t *testing.T, df *DataFrame, name string) []any {
	t.Helper()
	c, err := df.Col(name)
	if err != nil {
		t.Fatal(err)
	}
	return c.Values()
}

func TestConstructors(t *testing.T) {
	df := sampleFrame(t)
	rows, cols := df.Shape()
	if rows != 3 || cols != 4 {
		t.Fatalf("shape = %d, %d", rows, cols)
	}
	if got := df.Columns(); got[0] != "country" || got[3] != "salary" {
		t.Errorf("columns = %v", got)
	}
	if df.Empty() {
		t.Error("Empty on non-empty frame")
	}
	dt := df.DTypes()
	if dt["age"] != dtype.Int || dt["salary"] != dtype.Float64 || dt["name"] != dtype.String {
		t.Errorf("dtypes = %v", dt)
	}

	fromRows, err := DataFrameFromRows([]string{"a", "b"}, [][]any{{1, "x"}, {2, "y"}})
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, fromRows, "a"); v[1] != 2 {
		t.Errorf("from rows: %v", v)
	}

	fromMap, err := DataFrameFromMap(map[string][]any{"x": {1, 2}, "y": {"a", "b"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := fromMap.Columns(); got[0] != "x" || got[1] != "y" {
		t.Errorf("map columns sorted = %v", got)
	}

	type person struct {
		Name string `pd:"name"`
		Age  int
	}
	fromStructs, err := DataFrameFromStructs([]person{{"Ana", 30}, {"Luis", 40}})
	if err != nil {
		t.Fatal(err)
	}
	if got := fromStructs.Columns(); got[0] != "name" || got[1] != "Age" {
		t.Errorf("struct columns = %v", got)
	}

	// mismatched lengths
	if _, err := NewDataFrame(
		series.IntSeries("a", []int{1, 2}),
		series.IntSeries("b", []int{1}),
	); !errors.Is(err, errs.ErrLengthMismatch) {
		t.Errorf("length mismatch error = %v", err)
	}
}

func TestRoundTrips(t *testing.T) {
	df := sampleFrame(t)
	recs := df.ToRecords()
	if len(recs) != 3 || recs[0]["name"] != "Ana" {
		t.Errorf("ToRecords = %v", recs[0])
	}
	rows := df.ToRows()
	if rows[2][0] != "BR" {
		t.Errorf("ToRows = %v", rows[2])
	}
	arr, err := df.ToNDArray("age", "salary")
	if err != nil {
		t.Fatal(err)
	}
	if got := arr.Shape(); got[0] != 3 || got[1] != 2 {
		t.Errorf("ToNDArray shape = %v", got)
	}
	if arr.MustAt(1, 1) != 2000 {
		t.Errorf("ToNDArray value = %v", arr.MustAt(1, 1))
	}
	back, err := DataFrameFromNDArray(arr, []string{"age", "salary"})
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, back, "age"); v[2] != 35.0 {
		t.Errorf("from ndarray = %v", v)
	}
}

func TestSelectDropRename(t *testing.T) {
	df := sampleFrame(t)
	sel, err := df.Select("name", "age")
	if err != nil {
		t.Fatal(err)
	}
	if got := sel.Columns(); len(got) != 2 || got[0] != "name" {
		t.Errorf("select = %v", got)
	}
	if _, err := df.Select("nope"); !errors.Is(err, errs.ErrColumnNotFound) {
		t.Errorf("select missing error = %v", err)
	}
	dropped, err := df.Drop("country")
	if err != nil {
		t.Fatal(err)
	}
	if len(dropped.Columns()) != 3 {
		t.Errorf("drop = %v", dropped.Columns())
	}
	renamed, err := df.Rename(map[string]string{"name": "person"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := renamed.Col("person"); err != nil {
		t.Errorf("rename: %v", err)
	}
}

func TestRowAccess(t *testing.T) {
	df := sampleFrame(t)
	row, err := df.Row(1)
	if err != nil {
		t.Fatal(err)
	}
	if row["name"] != "Luis" || row["age"] != 40 {
		t.Errorf("Row(1) = %v", row)
	}
	if df.Head(2).Len() != 2 || df.Tail(1).Len() != 1 {
		t.Error("Head/Tail length")
	}
	taken, err := df.Take([]int{2, 0})
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, taken, "name"); v[0] != "Joao" {
		t.Errorf("Take = %v", v)
	}
	sampled, err := df.Sample(2, WithSampleSeed(1))
	if err != nil || sampled.Len() != 2 {
		t.Errorf("Sample = %v, %v", sampled, err)
	}
}

func TestFilterWhereQuery(t *testing.T) {
	df := sampleFrame(t)
	mask := df.MustCol("age").Gt(30)
	filtered, err := df.Filter(mask)
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Len() != 2 {
		t.Fatalf("Filter len = %d", filtered.Len())
	}
	where, err := df.Where(expr.And(
		expr.Col("country").Eq("AR"),
		expr.Col("salary").Ge(2000),
	))
	if err != nil {
		t.Fatal(err)
	}
	if where.Len() != 1 {
		t.Fatalf("Where len = %d", where.Len())
	}
	if v := colValues(t, where, "name"); v[0] != "Luis" {
		t.Errorf("Where = %v", v)
	}
	q, err := df.Query(`age >= 35 and country in ["AR", "BR"]`)
	if err != nil {
		t.Fatal(err)
	}
	if q.Len() != 2 {
		t.Errorf("Query len = %d", q.Len())
	}
}

func TestAssign(t *testing.T) {
	df := sampleFrame(t)
	withBonus, err := df.AssignExpr("bonus", expr.Col("salary").Mul(0.1))
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, withBonus, "bonus"); v[1] != 200.0 {
		t.Errorf("AssignExpr = %v", v)
	}
	withFlag, err := df.AssignValue("active", true)
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, withFlag, "active"); v[2] != true {
		t.Errorf("AssignValue = %v", v)
	}
	withFn, err := df.AssignFunc("initial", func(row map[string]any) any {
		return string(row["name"].(string)[0])
	})
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, withFn, "initial"); v[0] != "A" {
		t.Errorf("AssignFunc = %v", v)
	}
	// replace existing column
	replaced, err := df.AssignValue("age", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(replaced.Columns()) != 4 {
		t.Errorf("replace should keep column count, got %v", replaced.Columns())
	}
}

func TestMissingData(t *testing.T) {
	df, err := DataFrameFromRecords([]map[string]any{
		{"a": 1, "b": "x"},
		{"a": nil, "b": "y"},
		{"a": 3, "b": nil},
	}, WithColumnOrder("a", "b"))
	if err != nil {
		t.Fatal(err)
	}
	if !df.HasNA() {
		t.Error("HasNA = false")
	}
	isna := df.IsNA()
	if v := colValues(t, isna, "a"); v[1] != true || v[0] != false {
		t.Errorf("IsNA = %v", v)
	}
	dropped := df.DropNA()
	if dropped.Len() != 1 {
		t.Errorf("DropNA len = %d", dropped.Len())
	}
	subset := df.DropNA(DropNASubset("a"))
	if subset.Len() != 2 {
		t.Errorf("DropNA subset len = %d", subset.Len())
	}
	filled, err := df.FillNA(map[string]any{"a": 0, "b": "?"})
	if err != nil {
		t.Fatal(err)
	}
	if filled.HasNA() {
		t.Error("FillNA left missing values")
	}
	if v := colValues(t, filled, "b"); v[2] != "?" {
		t.Errorf("FillNA = %v", v)
	}
}

func TestSorting(t *testing.T) {
	df := sampleFrame(t)
	bySalary, err := df.SortValues("salary", false)
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, bySalary, "name"); v[0] != "Luis" || v[2] != "Ana" {
		t.Errorf("sort desc = %v", v)
	}
	multi, err := df.SortValuesBy([]string{"country", "age"}, []bool{true, false})
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, multi, "name"); v[0] != "Luis" || v[1] != "Ana" || v[2] != "Joao" {
		t.Errorf("multi sort = %v", v)
	}
}

func TestStatsAndDescribe(t *testing.T) {
	df := sampleFrame(t)
	if got := df.Sum()["salary"]; got != 4500 {
		t.Errorf("Sum = %v", got)
	}
	if got := df.Mean()["age"]; got != 35 {
		t.Errorf("Mean = %v", got)
	}
	if got := df.Count()["name"]; got != 3 {
		t.Errorf("Count = %v", got)
	}
	if got := df.Min()["name"]; got != "Ana" {
		t.Errorf("Min string = %v", got)
	}
	if got := df.Max()["salary"]; got != 2000.0 {
		t.Errorf("Max = %v", got)
	}
	d := df.Describe()
	if got := d.Columns(); len(got) != 2 {
		t.Fatalf("describe columns = %v", got)
	}
	meanRow, err := d.Loc().Row("mean")
	if err != nil {
		t.Fatal(err)
	}
	if meanRow["age"] != 35.0 {
		t.Errorf("describe mean age = %v", meanRow["age"])
	}
	info := df.Info()
	if !strings.Contains(info, "salary") {
		t.Errorf("Info = %q", info)
	}
}

func TestLocILoc(t *testing.T) {
	df := sampleFrame(t)
	sub, err := df.ILoc().Rows(sliceSpec(0, 2)).Cols(1, 2).Get()
	if err != nil {
		t.Fatal(err)
	}
	r, c := sub.Shape()
	if r != 2 || c != 2 {
		t.Fatalf("iloc shape = %d, %d", r, c)
	}
	if got := sub.Columns(); got[0] != "name" {
		t.Errorf("iloc cols = %v", got)
	}
	row, err := df.ILoc().Row(-1)
	if err != nil || row["name"] != "Joao" {
		t.Errorf("iloc row -1 = %v, %v", row, err)
	}
	locSub, err := df.Loc().Rows(0, 2).Cols("name").Get()
	if err != nil {
		t.Fatal(err)
	}
	if v := colValues(t, locSub, "name"); len(v) != 2 || v[1] != "Joao" {
		t.Errorf("loc = %v", v)
	}
}

func TestApplyMapPipe(t *testing.T) {
	df := sampleFrame(t)
	rowSums, err := df.Select("age", "salary")
	if err != nil {
		t.Fatal(err)
	}
	s, err := rowSums.Apply(1, func(values []any) any {
		total := 0.0
		for _, v := range values {
			if f, ok := dtype.AsFloat(v); ok {
				total += f
			}
		}
		return total
	})
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := s.At(0); v != 1030.0 {
		t.Errorf("Apply axis=1 = %v", v)
	}
	piped, err := df.Pipe(func(d *DataFrame) (*DataFrame, error) { return d.Select("name") })
	if err != nil || len(piped.Columns()) != 1 {
		t.Errorf("Pipe = %v, %v", piped.Columns(), err)
	}
}

func sliceSpec(start, stop int) ndarray.SliceSpec {
	return ndarray.Slice(start, stop)
}
