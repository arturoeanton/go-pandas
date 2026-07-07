package compat_test

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

// Fixtures mirroring generate_pandas_goldens.py multiindex_suite().

func miFrame(t *testing.T) *pd.DataFrame {
	t.Helper()
	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "city": "Buenos Aires", "salary": 1000.0},
		{"country": "AR", "city": "Cordoba", "salary": 800.0},
		{"country": "BR", "city": "Sao Paulo", "salary": 1500.0},
		{"country": "AR", "city": "Buenos Aires", "salary": 1200.0},
	}, pd.WithColumnOrder("country", "city", "salary"))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

func miIndexed(t *testing.T) *pd.DataFrame {
	t.Helper()
	indexed, err := miFrame(t).SetIndex("country", "city")
	if err != nil {
		t.Fatal(err)
	}
	return indexed
}

var multiindexCases = map[string]caseFn{
	"mi_set_reset_roundtrip": func(t *testing.T) (any, error) {
		return miIndexed(t).ResetIndex(), nil
	},
	"mi_levels_sorted": func(t *testing.T) (any, error) {
		mi, err := pd.NewMultiIndexFromArrays(
			[][]any{{"b", "a", "b"}, {"y", "x", "x"}}, []string{"l1", "l2"})
		if err != nil {
			return nil, err
		}
		var flat []any
		for _, level := range mi.Levels() {
			flat = append(flat, level...)
		}
		return pd.NewSeries("levels", flat), nil
	},
	"mi_codes": func(t *testing.T) (any, error) {
		mi, err := pd.NewMultiIndexFromArrays(
			[][]any{{"b", "a", "b"}, {"y", "x", "x"}}, []string{"l1", "l2"})
		if err != nil {
			return nil, err
		}
		var flat []int
		for _, codes := range mi.Codes() {
			for _, c := range codes {
				flat = append(flat, int(c))
			}
		}
		return pd.IntSeries("codes", flat), nil
	},
	"mi_loc_full_tuple": func(t *testing.T) (any, error) {
		rows, err := miIndexed(t).Loc().Tuple("AR", "Buenos Aires").Get()
		if err != nil {
			return nil, err
		}
		return rows.ResetIndex(), nil
	},
	"mi_loc_prefix": func(t *testing.T) (any, error) {
		rows, err := miIndexed(t).Loc().TuplePrefix("AR").Get()
		if err != nil {
			return nil, err
		}
		return rows.ResetIndex(), nil
	},
	"mi_groupby_default": func(t *testing.T) (any, error) {
		return miFrame(t).GroupBy("country", "city").Mean("salary")
	},
	"mi_groupby_as_index_roundtrip": func(t *testing.T) (any, error) {
		g, err := miFrame(t).GroupBy("country", "city").AsIndex(true).Mean("salary")
		if err != nil {
			return nil, err
		}
		return g.ResetIndex(), nil
	},
	"mi_na_component_roundtrip": func(t *testing.T) (any, error) {
		df, err := pd.DataFrameFromRecords([]map[string]any{
			{"country": "AR", "city": "BA", "v": 1.0},
			{"country": nil, "city": "X", "v": 2.0},
			{"country": "BR", "city": nil, "v": 3.0},
		}, pd.WithColumnOrder("country", "city", "v"))
		if err != nil {
			return nil, err
		}
		indexed, err := df.SetIndex("country", "city")
		if err != nil {
			return nil, err
		}
		return indexed.ResetIndex(), nil
	},
}

func init() {
	for name, fn := range multiindexCases {
		pandasCases[name] = fn
	}
}
