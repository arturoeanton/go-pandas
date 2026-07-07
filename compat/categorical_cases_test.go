package compat_test

import (
	"bytes"
	"strings"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

// Fixtures mirroring compat/python/generate_pandas_goldens.py
// categorical_suite() — keep in sync.

func catDefault(t *testing.T) *pd.Series {
	t.Helper()
	s, err := pd.NewSeries("s", []any{"m", "s", "l", "m", nil, "s"}).Astype(pd.Category)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func catOrdered(t *testing.T) *pd.Series {
	t.Helper()
	s, err := pd.CategoricalSeries("size", []string{"m", "s", "l", "m", "s"},
		pd.WithCategories("s", "m", "l"), pd.WithOrdered(true))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func catFrame(t *testing.T, sizes *pd.Series) *pd.DataFrame {
	t.Helper()
	df, err := pd.NewDataFrame(sizes, pd.FloatSeries("price", []float64{5, 1, 10, 6, 2}))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

func catAccessor(t *testing.T, s *pd.Series) *pd.CategoricalAccessor {
	t.Helper()
	acc, err := s.Cat()
	if err != nil {
		t.Fatal(err)
	}
	return acc
}

var categoricalCases = map[string]caseFn{
	"cat_codes_default": func(t *testing.T) (any, error) {
		codes := catAccessor(t, catDefault(t)).Codes()
		values := make([]int, len(codes))
		for i, c := range codes {
			values[i] = int(c)
		}
		return pd.IntSeries("codes", values), nil
	},
	"cat_categories_default": func(t *testing.T) (any, error) {
		return pd.NewSeries("categories", catAccessor(t, catDefault(t)).Categories()), nil
	},
	"cat_ordered_gt": func(t *testing.T) (any, error) {
		return catOrdered(t).Gt("m"), nil
	},
	"cat_ordered_le": func(t *testing.T) (any, error) {
		return catOrdered(t).Le("m"), nil
	},
	"cat_value_counts": func(t *testing.T) (any, error) {
		return catOrdered(t).ValueCounts(), nil
	},
	"cat_value_counts_explicit": func(t *testing.T) (any, error) {
		s, err := pd.CategoricalSeries("s", []string{"b", "a", "b", "c"},
			pd.WithCategories("c", "b", "a"))
		if err != nil {
			return nil, err
		}
		return s.ValueCounts(), nil
	},
	"cat_sort_values": func(t *testing.T) (any, error) {
		return catOrdered(t).SortValues(true), nil
	},
	"cat_groupby_mean": func(t *testing.T) (any, error) {
		return catFrame(t, catOrdered(t)).GroupBy("size").Mean()
	},
	"cat_rename_categories": func(t *testing.T) (any, error) {
		return catAccessor(t, catOrdered(t)).RenameCategories(map[any]any{"s": "small"})
	},
	"cat_set_categories_na": func(t *testing.T) (any, error) {
		return catAccessor(t, catOrdered(t)).SetCategories([]any{"m", "l"}, true)
	},
	"cat_merge_inner": func(t *testing.T) (any, error) {
		size, err := pd.CategoricalSeries("size", []string{"m", "s", "l", "m", "s"})
		if err != nil {
			return nil, err
		}
		dim, err := pd.CategoricalSeries("size", []string{"s", "m", "l"})
		if err != nil {
			return nil, err
		}
		right, err := pd.NewDataFrame(dim,
			pd.StringSeries("label", []string{"small", "medium", "large"}))
		if err != nil {
			return nil, err
		}
		return catFrame(t, size).Merge(right, pd.MergeOptions{On: []string{"size"}, How: "inner"})
	},
	"cat_csv_roundtrip": func(t *testing.T) (any, error) {
		size, err := pd.CategoricalSeries("size", []string{"m", "s", "l", "m", "s"})
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := catFrame(t, size).WriteCSV(&buf); err != nil {
			return nil, err
		}
		back, err := pd.ReadCSVReader(strings.NewReader(buf.String()), pd.WithCategorical("size"))
		if err != nil {
			return nil, err
		}
		if dt := back.MustCol("size").DType(); dt != pd.Category {
			t.Fatalf("round-trip dtype = %v, want category", dt)
		}
		return back, nil
	},
}

func init() {
	for name, fn := range categoricalCases {
		pandasCases[name] = fn
	}
}
