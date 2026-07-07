package compat_test

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/ndarray"
)

var hardeningV1001Cases = map[string]caseFn{
	"h_stack_with_na": func(t *testing.T) (any, error) {
		df, err := pd.DataFrameFromMap(map[string][]any{
			"a": {1.0, nil}, "b": {nil, 4.0},
		}, pd.WithColumnOrder("a", "b"),
			pd.WithDataFrameIndex(index.NewStringIndex([]string{"x", "y"}, "")))
		if err != nil {
			return nil, err
		}
		s, err := df.Stack()
		if err != nil {
			return nil, err
		}
		return pd.NewSeries("", s.Values()), nil
	},
	"h_pivot_fill_value": func(t *testing.T) (any, error) {
		sales, err := pd.DataFrameFromRecords([]map[string]any{
			{"c": "AR", "m": "jan", "v": 10.0},
			{"c": "BR", "m": "feb", "v": 20.0},
		}, pd.WithColumnOrder("c", "m", "v"))
		if err != nil {
			return nil, err
		}
		return sales.PivotTable(pd.PivotTableOptions{
			Values: []string{"v"}, Index: []string{"c"}, Columns: []string{"m"},
			AggFunc: "sum", FillValue: 0.0,
		})
	},
	"h_transform_with_na": func(t *testing.T) (any, error) {
		df, err := pd.DataFrameFromRecords([]map[string]any{
			{"k": "a", "v": 1.0}, {"k": "a", "v": nil}, {"k": "b", "v": 3.0},
		}, pd.WithColumnOrder("k", "v"))
		if err != nil {
			return nil, err
		}
		return df.GroupBy("k").Transform("v", "count")
	},
	"h_query_precedence": func(t *testing.T) (any, error) {
		prec, err := pd.DataFrameFromRecords([]map[string]any{
			{"a": 2.0, "b": 3.0, "c": 4.0},
			{"a": 5.0, "b": 1.0, "c": 0.0},
		}, pd.WithColumnOrder("a", "b", "c"))
		if err != nil {
			return nil, err
		}
		return prec.Query("a + b * c > 10")
	},
	"h_resample_allna_bucket": func(t *testing.T) (any, error) {
		dates, err := pd.ToDatetime(pd.StringSeries("date", []string{
			"2026-01-01 01:00:00", "2026-01-01 02:00:00", "2026-01-02 05:00:00",
		}))
		if err != nil {
			return nil, err
		}
		df, err := pd.NewDataFrame(dates, pd.NewSeries("v", []any{nil, nil, 3.0}))
		if err != nil {
			return nil, err
		}
		indexed, err := df.SetIndex("date")
		if err != nil {
			return nil, err
		}
		out, err := indexed.Resample("D").Mean()
		if err != nil {
			return nil, err
		}
		return out.ResetIndex(), nil
	},
}

var hardeningNumpyCases = map[string]caseFn{
	"take_repeated": func(t *testing.T) (any, error) {
		return ndarray.Array([]float64{10, 20, 30}).Take([]int{2, 2, 0, 2}, 0)
	},
}

func init() {
	for name, fn := range hardeningV1001Cases {
		pandasCases[name] = fn
	}
	for name, fn := range hardeningNumpyCases {
		numpyCases[name] = fn
	}
}
