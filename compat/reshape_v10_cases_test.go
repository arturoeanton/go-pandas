package compat_test

import (
	"fmt"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/index"
)

func v10People(t *testing.T) *pd.DataFrame {
	t.Helper()
	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "salary": 1000.0, "bonus": 300.0},
		{"country": "AR", "salary": 2000.0, "bonus": 50.0},
		{"country": "BR", "salary": 1500.0, "bonus": 100.0},
		{"country": "CL", "salary": 800.0, "bonus": 500.0},
		{"country": "AR", "salary": 800.0, "bonus": 100.0},
	}, pd.WithColumnOrder("country", "salary", "bonus"))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

func v10Stacked(t *testing.T) *pd.Series {
	t.Helper()
	df, err := pd.DataFrameFromMap(map[string][]any{
		"a": {1.0, 2.0}, "b": {3.0, 4.0},
	}, pd.WithColumnOrder("a", "b"),
		pd.WithDataFrameIndex(index.NewStringIndex([]string{"x", "y"}, "")))
	if err != nil {
		t.Fatal(err)
	}
	s, err := df.Stack()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

var reshapeV10Cases = map[string]caseFn{
	"v10_stack_values": func(t *testing.T) (any, error) {
		return pd.NewSeries("", v10Stacked(t).Values()), nil
	},
	"v10_stack_labels": func(t *testing.T) (any, error) {
		mi := v10Stacked(t).Index().(*pd.MultiIndex)
		labels := make([]any, mi.Len())
		for i := range labels {
			tup := mi.Tuple(i)
			labels[i] = fmt.Sprintf("%v|%v", tup[0], tup[1])
		}
		return pd.NewSeries("", labels), nil
	},
	"v10_unstack": func(t *testing.T) (any, error) {
		df, err := pd.DataFrameFromRecords([]map[string]any{
			{"country": "AR", "city": "BA", "v": 1.0},
			{"country": "AR", "city": "CO", "v": 2.0},
			{"country": "BR", "city": "SP", "v": 3.0},
		}, pd.WithColumnOrder("country", "city", "v"))
		if err != nil {
			return nil, err
		}
		indexed, err := df.SetIndex("country", "city")
		if err != nil {
			return nil, err
		}
		out, err := indexed.Unstack()
		if err != nil {
			return nil, err
		}
		return out.ResetIndex(), nil
	},
	"v10_stack_multiindex": func(t *testing.T) (any, error) {
		df, err := pd.DataFrameFromRecords([]map[string]any{
			{"country": "AR", "city": "BA", "v": 1.0},
			{"country": "AR", "city": "CO", "v": 2.0},
			{"country": "BR", "city": "SP", "v": 3.0},
		}, pd.WithColumnOrder("country", "city", "v"))
		if err != nil {
			return nil, err
		}
		indexed, err := df.SetIndex("country", "city")
		if err != nil {
			return nil, err
		}
		s, err := indexed.Stack()
		if err != nil {
			return nil, err
		}
		if s.Index().(*pd.MultiIndex).NLevels() != 3 {
			t.Fatal("MultiIndex stack must append a level")
		}
		return pd.NewSeries("", s.Values()), nil
	},
	"v10_transform_mean": func(t *testing.T) (any, error) {
		return v10People(t).GroupBy("country").Transform("salary", "mean")
	},
	"v10_transform_sum": func(t *testing.T) (any, error) {
		return v10People(t).GroupBy("country").Transform("salary", "sum")
	},
	"v10_filter_size": func(t *testing.T) (any, error) {
		return v10People(t).GroupBy("country").Filter(pd.GroupSize().Gt(1))
	},
	"v10_query_arith": func(t *testing.T) (any, error) {
		return v10People(t).Query("salary + bonus > 1300")
	},
	"v10_query_in": func(t *testing.T) (any, error) {
		return v10People(t).Query(`country in ["AR", "BR"]`)
	},
	"v10_query_not_in": func(t *testing.T) (any, error) {
		return v10People(t).Query(`country not in ["CL"]`)
	},
	"v10_query_parens": func(t *testing.T) (any, error) {
		return v10People(t).Query(`(salary > 900 and bonus < 200) or country == "CL"`)
	},
	"v10_pivot_multi": func(t *testing.T) (any, error) {
		sales, err := pd.DataFrameFromRecords([]map[string]any{
			{"country": "AR", "month": "jan", "sales": 10.0, "qty": 1.0},
			{"country": "AR", "month": "feb", "sales": 20.0, "qty": 2.0},
			{"country": "BR", "month": "jan", "sales": 30.0, "qty": 3.0},
			{"country": "AR", "month": "jan", "sales": 40.0, "qty": 4.0},
		}, pd.WithColumnOrder("country", "month", "sales", "qty"))
		if err != nil {
			return nil, err
		}
		return sales.PivotTable(pd.PivotTableOptions{
			Values:   []string{"sales", "qty"},
			Index:    []string{"country"},
			Columns:  []string{"month"},
			AggFuncs: []string{"sum", "mean"},
		})
	},
}

func init() {
	for name, fn := range reshapeV10Cases {
		pandasCases[name] = fn
	}
}
