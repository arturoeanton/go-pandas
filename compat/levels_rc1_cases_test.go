package compat_test

import (
	"fmt"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

func rc1Frame(t *testing.T) (*pd.DataFrame, *pd.MultiIndex) {
	t.Helper()
	mi, err := pd.MultiIndexFromTuples([]string{"c", "t", "y"}, []pd.Tuple{
		{"AR", "BA", 2023}, {"AR", "CO", 2024}, {"BR", "SP", 2023}, {"AR", "BA", 2024},
	})
	if err != nil {
		t.Fatal(err)
	}
	df, err := pd.DataFrameFromMap(map[string][]any{"v": {1.0, 2.0, 3.0, 4.0}},
		pd.WithDataFrameIndex(mi))
	if err != nil {
		t.Fatal(err)
	}
	return df, mi
}

var levelsRC1Cases = map[string]caseFn{
	"rc1_droplevel": func(t *testing.T) (any, error) {
		_, mi := rc1Frame(t)
		dropped, err := mi.DropLevel("t")
		if err != nil {
			return nil, err
		}
		dm := dropped.(*pd.MultiIndex)
		labels := make([]any, dm.Len())
		for i := range labels {
			tup := dm.Tuple(i)
			labels[i] = fmt.Sprintf("%v|%v", tup[0], tup[1])
		}
		return pd.NewSeries("", labels), nil
	},
	"rc1_swaplevel": func(t *testing.T) (any, error) {
		_, mi := rc1Frame(t)
		sw, err := mi.SwapLevel(0, 2)
		if err != nil {
			return nil, err
		}
		var flat []any
		for _, n := range sw.Names() {
			flat = append(flat, n)
		}
		for _, v := range sw.Tuple(0) {
			flat = append(flat, fmt.Sprint(v))
		}
		return pd.NewSeries("", flat), nil
	},
	"rc1_xs_level0": func(t *testing.T) (any, error) {
		df, _ := rc1Frame(t)
		out, err := df.XS("AR", "c")
		if err != nil {
			return nil, err
		}
		return out.ResetIndex(), nil
	},
	"rc1_xs_level1": func(t *testing.T) (any, error) {
		df, _ := rc1Frame(t)
		out, err := df.XS("BA", 1)
		if err != nil {
			return nil, err
		}
		return out.ResetIndex(), nil
	},
}

func init() {
	for name, fn := range levelsRC1Cases {
		pandasCases[name] = fn
	}
}
