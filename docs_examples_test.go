package pandas_test

// This file keeps the README and docs/ code examples honest: every
// executable snippet shown in the documentation is exercised here.

import (
	"strings"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

func TestReadmeDataFrameExample(t *testing.T) {
	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "name": "Ana", "age": 30, "salary": 1000.0},
		{"country": "AR", "name": "Luis", "age": 40, "salary": 2000.0},
		{"country": "BR", "name": "Joao", "age": 35, "salary": 1500.0},
	}, pd.WithColumnOrder("country", "name", "age", "salary"))
	if err != nil {
		t.Fatal(err)
	}
	adults, err := df.Query(`age > 30 and country in ["AR", "BR"]`)
	if err != nil {
		t.Fatal(err)
	}
	top, err := adults.SortValues("salary", false)
	if err != nil {
		t.Fatal(err)
	}
	if top.Head(5).Len() != 2 {
		t.Fatalf("example result rows = %d", top.Len())
	}
	if plan := pd.DebugPlan(df, pd.Col("age").Gt(30)); !strings.HasPrefix(plan, "columnar") {
		t.Fatalf("DebugPlan = %s", plan)
	}
	if _, err := df.Corr(); err != nil {
		t.Fatal(err)
	}
	_ = df.DropNA()
	if _, err := df.SelectDTypes(pd.Include(pd.Number)); err != nil {
		t.Fatal(err)
	}
}

func TestReadmeSeriesExample(t *testing.T) {
	s := pd.SeriesOf("v", []int{3, 1, 4, 1, 5})
	if _, err := s.Rank(pd.RankMethod("dense")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PctChange(1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Cumsum(); err != nil {
		t.Fatal(err)
	}
	if s.ValueCounts().Len() != 4 {
		t.Fatal("value counts")
	}
}

func TestReadmeNDArrayExample(t *testing.T) {
	m, err := pd.Arange(6).Reshape(2, 3)
	if err != nil {
		t.Fatal(err)
	}
	c, err := m.Add(pd.Array([]float64{10, 20, 30}))
	if err != nil {
		t.Fatal(err)
	}
	view, err := m.Slice(pd.All(), pd.Slice(1, 3))
	if err != nil {
		t.Fatal(err)
	}
	norm := m.SubScalar(m.MeanAll()).DivScalar(m.StdAll())
	tr, err := norm.T()
	if err != nil {
		t.Fatal(err)
	}
	prod, err := pd.MatMul(m, tr)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _ = c, view, prod
}

func TestReadmeGroupByMergeExamples(t *testing.T) {
	df, _ := pd.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "dept": "eng", "salary": 1000.0, "age": 30},
		{"country": "BR", "dept": "eng", "salary": 1500.0, "age": 35},
	}, pd.WithColumnOrder("country", "dept", "salary", "age"))
	out, err := df.GroupBy("country", "dept").AggList(map[string][]string{
		"salary": {"mean", "max"},
		"age":    {"min"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"country", "dept", "age_min", "salary_mean", "salary_max"}
	got := out.Columns()
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("agg columns = %v, want %v", got, want)
		}
	}

	left, _ := pd.DataFrameFromMap(map[string][]any{"id": {1, 2}, "l": {"a", "b"}}, pd.WithColumnOrder("id", "l"))
	right, _ := pd.DataFrameFromMap(map[string][]any{"id": {1, 3}, "r": {"x", "y"}}, pd.WithColumnOrder("id", "r"))
	merged, err := pd.Merge(left, right, pd.MergeOptions{
		On: []string{"id"}, How: "outer",
		Validate: "one_to_one", Indicator: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if merged.Len() != 3 {
		t.Fatalf("merge example rows = %d", merged.Len())
	}
	stacked, err := pd.Concat([]*pd.DataFrame{left, left}, pd.IgnoreIndex(true), pd.Join("inner"))
	if err != nil || stacked.Len() != 4 {
		t.Fatalf("concat example = %v, %v", stacked, err)
	}
}

func TestReadmeLocILocRollingExamples(t *testing.T) {
	df, _ := pd.DataFrameFromMap(map[string][]any{
		"a": {1, 2, 3, 4, 5, 6, 7, 8}, "b": {1, 1, 1, 1, 1, 1, 1, 1}, "c": {2, 2, 2, 2, 2, 2, 2, 2},
	}, pd.WithColumnOrder("a", "b", "c"))
	out, err := df.ILoc().Rows(0, 2, pd.Slice(4, 8)).Cols(pd.Slice(1, 3)).Get()
	if err != nil {
		t.Fatal(err)
	}
	if r, c := out.Shape(); r != 6 || c != 2 {
		t.Fatalf("iloc example shape = %d x %d", r, c)
	}
	labeled, _ := pd.DataFrameFromMap(map[string][]any{"name": {"x", "y", "z"}},
		pd.WithDataFrameIndex(pd.NewStringIndex([]string{"a", "b", "d"})))
	if _, err := labeled.Loc().Rows(pd.LabelSlice("a", "d")).Cols("name").Get(); err != nil {
		t.Fatal(err)
	}

	prices := pd.FloatSeries("p", []float64{10, 11, 12, 11, 13})
	if _, err := prices.Rolling(3, pd.MinPeriods(1)).Mean(); err != nil {
		t.Fatal(err)
	}
	if _, err := prices.Rolling(3).Std(); err != nil {
		t.Fatal(err)
	}
	if _, err := prices.Expanding().Max(); err != nil {
		t.Fatal(err)
	}
}

func TestDocsIOExample(t *testing.T) {
	dir := t.TempDir()
	df, _ := pd.DataFrameFromMap(map[string][]any{
		"name": {"Ana", "Luis"}, "age": {30, 40}, "joined": {"2024-01-02", "2023-05-06"},
	}, pd.WithColumnOrder("name", "age", "joined"))
	csvPath := dir + "/people.csv"
	if err := df.ToCSV(csvPath); err != nil {
		t.Fatal(err)
	}
	back, err := pd.ReadCSV(csvPath,
		pd.WithUseCols("name", "age"),
		pd.WithNRows(1000),
		pd.WithNAValues("-"), pd.WithKeepDefaultNA(true))
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Columns()) != 2 {
		t.Fatalf("usecols = %v", back.Columns())
	}
	jsonPath := dir + "/out.json"
	if err := df.ToJSON(jsonPath, pd.JSONOrient("split")); err != nil {
		t.Fatal(err)
	}
	if _, err := pd.ReadJSON(jsonPath, pd.JSONOrient("split")); err != nil {
		t.Fatal(err)
	}
}
