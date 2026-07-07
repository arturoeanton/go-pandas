package compat_test

import (
	"strings"
	"testing"
	"time"

	pd "github.com/arturoeanton/go-pandas"
)

func TestPandasGoldens(t *testing.T) {
	runSuites(t, "pandas", pandasCases)
}

// frameOf wraps a Series construction into a one-column frame.
func frameOf(s *pd.Series, err error) (any, error) {
	if err != nil {
		return nil, err
	}
	return pd.NewDataFrame(s)
}

func dateSeries() *pd.Series {
	mk := func(s string) time.Time {
		t, _ := pd.ParseDatetime(s)
		return t
	}
	return pd.TimeSeries("d", []time.Time{
		mk("2024-01-01"),
		mk("2024-03-15 10:30:45"),
		mk("2023-12-31"),
		mk("2024-06-01"),
	})
}

func naKeyFrame() (*pd.DataFrame, error) {
	return pd.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "v": 1.0},
		{"country": nil, "v": 2.0},
		{"country": "BR", "v": 3.0},
		{"country": nil, "v": 4.0},
	}, pd.WithColumnOrder("country", "v"))
}

func shopFrame(t *testing.T) *pd.DataFrame {
	t.Helper()
	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"item": "pen", "price": 1.5, "qty": 10},
		{"item": "book", "price": 12.0, "qty": 2},
		{"item": "mug", "price": 7.25, "qty": 4},
	}, pd.WithColumnOrder("item", "price", "qty"))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

func dupFrame(t *testing.T) *pd.DataFrame {
	t.Helper()
	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"a": 1, "b": "x"},
		{"a": 1, "b": "x"},
		{"a": 2, "b": "y"},
	}, pd.WithColumnOrder("a", "b"))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

var pandasCases = map[string]caseFn{
	// dataframe_core ------------------------------------------------------
	"head_2": func(t *testing.T) (any, error) { return peopleFrame(t).Head(2), nil },
	"iloc_take": func(t *testing.T) (any, error) {
		return peopleFrame(t).ILoc().Rows(0, 2, 4).Get()
	},
	"tail_2": func(t *testing.T) (any, error) { return peopleFrame(t).Tail(2), nil },
	"select_name_age": func(t *testing.T) (any, error) {
		return peopleFrame(t).Select("name", "age")
	},
	"drop_columns": func(t *testing.T) (any, error) {
		return peopleFrame(t).Drop("dept", "name")
	},
	"rename_age": func(t *testing.T) (any, error) {
		return peopleFrame(t).Rename(map[string]string{"age": "years"})
	},
	"sort_salary_desc": func(t *testing.T) (any, error) {
		return peopleFrame(t).SortValues("salary", false)
	},
	"sort_multi": func(t *testing.T) (any, error) {
		return peopleFrame(t).SortValuesBy([]string{"country", "age"}, []bool{true, false})
	},
	"assign_bonus": func(t *testing.T) (any, error) {
		df, err := peopleFrame(t).AssignExpr("bonus", pd.Col("salary").Mul(0.1))
		if err != nil {
			return nil, err
		}
		return df.Select("name", "bonus")
	},
	"query_age_salary": func(t *testing.T) (any, error) {
		return peopleFrame(t).Query("age > 25 and salary < 1600")
	},
	"query_in": func(t *testing.T) (any, error) {
		return peopleFrame(t).Query(`country in ["BR"]`)
	},
	"describe": func(t *testing.T) (any, error) {
		sub, err := peopleFrame(t).Select("age", "salary")
		if err != nil {
			return nil, err
		}
		return sub.Describe().ResetIndex(), nil
	},
	"round_1": func(t *testing.T) (any, error) {
		salary, err := peopleFrame(t).MustCol("salary").DivScalar(3.0)
		if err != nil {
			return nil, err
		}
		df, err := pd.NewDataFrame(salary)
		if err != nil {
			return nil, err
		}
		return df.Round(1)
	},
	"clip_age": func(t *testing.T) (any, error) {
		sub, err := peopleFrame(t).Select("age")
		if err != nil {
			return nil, err
		}
		return sub.Clip(25, 35)
	},
	"duplicated": func(t *testing.T) (any, error) {
		return dupFrame(t).Duplicated()
	},
	"drop_duplicates": func(t *testing.T) (any, error) {
		return dupFrame(t).DropDuplicates()
	},
	"nunique": func(t *testing.T) (any, error) {
		df := peopleFrame(t)
		counts := df.NUnique()
		names := df.Columns()
		values := make([]any, len(names))
		for i, name := range names {
			values[i] = counts[name]
		}
		return pd.NewSeries("nunique", values, pd.WithIndex(pd.NewStringIndex(names))), nil
	},
	"corr": func(t *testing.T) (any, error) {
		sub, err := peopleFrame(t).Select("age", "salary")
		if err != nil {
			return nil, err
		}
		m, err := sub.Corr()
		if err != nil {
			return nil, err
		}
		return m.ResetIndex(), nil
	},
	"select_dtypes_number": func(t *testing.T) (any, error) {
		return peopleFrame(t).SelectDTypes(pd.Include(pd.Number))
	},

	// series_core ----------------------------------------------------------
	"head_3":       func(t *testing.T) (any, error) { return numSeries().Head(3), nil },
	"astype_float": func(t *testing.T) (any, error) { return intSeries().Astype(pd.Float64) },
	"isna":         func(t *testing.T) (any, error) { return numSeries().IsNA(), nil },
	"notna":        func(t *testing.T) (any, error) { return numSeries().NotNA(), nil },
	"dropna":       func(t *testing.T) (any, error) { return numSeries().DropNA(), nil },
	"fillna_0":     func(t *testing.T) (any, error) { return numSeries().FillNA(0.0), nil },
	"unique":       func(t *testing.T) (any, error) { return intSeries().Unique(), nil },
	"nunique_series": func(t *testing.T) (any, error) {
		return float64(intSeries().NUnique(true)), nil
	},
	"value_counts": func(t *testing.T) (any, error) { return intSeries().ValueCounts(), nil },
	"sort_values":  func(t *testing.T) (any, error) { return numSeries().SortValues(true), nil },
	"mean":         scalarCase(func() (float64, error) { return numSeries().Mean() }),
	"median":       scalarCase(func() (float64, error) { return numSeries().Median() }),
	"std":          scalarCase(func() (float64, error) { return numSeries().Std() }),
	"var":          scalarCase(func() (float64, error) { return numSeries().Var() }),
	"quantile_25":  scalarCase(func() (float64, error) { return numSeries().Quantile(0.25) }),
	"sum":          scalarCase(func() (float64, error) { return numSeries().Sum() }),
	"between_2_4": func(t *testing.T) (any, error) {
		return intSeries().Between(2, 4, "both"), nil
	},
	"isin":         func(t *testing.T) (any, error) { return intSeries().IsIn(1, 5), nil },
	"cumsum":       func(t *testing.T) (any, error) { return numSeries().Cumsum() },
	"cummax":       func(t *testing.T) (any, error) { return numSeries().Cummax() },
	"cumprod":      func(t *testing.T) (any, error) { return intSeries().Cumprod() },
	"diff_1":       func(t *testing.T) (any, error) { return numSeries().Diff(1) },
	"pct_change_1": func(t *testing.T) (any, error) { return numSeries().PctChange(1) },
	"rank_average": func(t *testing.T) (any, error) { return intSeries().Rank() },
	"rank_dense": func(t *testing.T) (any, error) {
		return intSeries().Rank(pd.RankMethod("dense"))
	},
	"clip_2_4": func(t *testing.T) (any, error) { return intSeries().Clip(2, 4) },
	"round_0":  func(t *testing.T) (any, error) { return numSeries().Round(0) },
	"abs": func(t *testing.T) (any, error) {
		return pd.FloatSeries("n", []float64{-1.5, 2.0, -3.0}).Abs()
	},
	"shift_1": func(t *testing.T) (any, error) { return intSeries().Shift(1), nil },

	// groupby ---------------------------------------------------------------
	"size": func(t *testing.T) (any, error) { return peopleFrame(t).GroupBy("country").Size() },
	"count_name": func(t *testing.T) (any, error) {
		return peopleFrame(t).GroupBy("country").Count("name")
	},
	"sum_salary": func(t *testing.T) (any, error) {
		return peopleFrame(t).GroupBy("country").Sum("salary")
	},
	"mean_salary": func(t *testing.T) (any, error) {
		return peopleFrame(t).GroupBy("country").Mean("salary")
	},
	"median_salary": func(t *testing.T) (any, error) {
		return peopleFrame(t).GroupBy("country").Median("salary")
	},
	"min_salary": func(t *testing.T) (any, error) {
		return peopleFrame(t).GroupBy("country").Min("salary")
	},
	"max_salary": func(t *testing.T) (any, error) {
		return peopleFrame(t).GroupBy("country").Max("salary")
	},
	"std_salary": func(t *testing.T) (any, error) {
		return peopleFrame(t).GroupBy("country").Std("salary")
	},
	"mean_two_keys": func(t *testing.T) (any, error) {
		return peopleFrame(t).GroupBy("country", "dept").Mean("salary")
	},
	"agg_named": func(t *testing.T) (any, error) {
		return peopleFrame(t).GroupBy("country").AggList(map[string][]string{
			"salary": {"max", "mean"},
			"age":    {"min"},
		})
	},
	"nunique_dept": func(t *testing.T) (any, error) {
		return peopleFrame(t).GroupBy("country").NUnique("dept")
	},
	"var_salary": func(t *testing.T) (any, error) {
		return peopleFrame(t).GroupBy("country").Var("salary")
	},
	"size_dropna_false": func(t *testing.T) (any, error) {
		df, err := naKeyFrame()
		if err != nil {
			return nil, err
		}
		return df.GroupByOpts([]pd.GroupByOption{pd.GroupDropNA(false)}, "country").Size()
	},
	"size_dropna_true": func(t *testing.T) (any, error) {
		df, err := naKeyFrame()
		if err != nil {
			return nil, err
		}
		return df.GroupBy("country").Size()
	},
	"first": func(t *testing.T) (any, error) {
		return peopleFrame(t).GroupBy("country").First("name")
	},
	"last": func(t *testing.T) (any, error) {
		return peopleFrame(t).GroupBy("country").Last("name")
	},

	// merge / join / concat ---------------------------------------------------
	"merge_inner": mergeCase("inner"),
	"merge_left":  mergeCase("left"),
	"merge_right": mergeCase("right"),
	"merge_outer": mergeCase("outer"),
	"merge_left_on_right_on": func(t *testing.T) (any, error) {
		left, err := pd.DataFrameFromRecords([]map[string]any{
			{"user_id": 1, "v": 10},
			{"user_id": 2, "v": 20},
		}, pd.WithColumnOrder("user_id", "v"))
		if err != nil {
			return nil, err
		}
		_, right := mergeFrames(t)
		return left.Merge(right, pd.MergeOptions{
			LeftOn:  []string{"user_id"},
			RightOn: []string{"id"},
		})
	},
	"merge_dup_keys": func(t *testing.T) (any, error) {
		l, _ := pd.DataFrameFromRecords([]map[string]any{
			{"id": 1, "l": "a"}, {"id": 1, "l": "b"},
		}, pd.WithColumnOrder("id", "l"))
		r, _ := pd.DataFrameFromRecords([]map[string]any{
			{"id": 1, "r": "x"}, {"id": 1, "r": "y"}, {"id": 1, "r": "z"},
		}, pd.WithColumnOrder("id", "r"))
		return l.Merge(r, pd.MergeOptions{On: []string{"id"}, How: "inner"})
	},
	"merge_string_key_left": func(t *testing.T) (any, error) {
		l, _ := pd.DataFrameFromRecords([]map[string]any{
			{"k": "a", "l": 1}, {"k": "b", "l": 2},
		}, pd.WithColumnOrder("k", "l"))
		r, _ := pd.DataFrameFromRecords([]map[string]any{
			{"k": "b", "r": 10}, {"k": "c", "r": 20},
		}, pd.WithColumnOrder("k", "r"))
		return l.Merge(r, pd.MergeOptions{On: []string{"k"}, How: "left"})
	},
	"merge_time_key_inner": func(t *testing.T) (any, error) {
		d1, _ := pd.ParseDatetime("2024-01-01")
		d2, _ := pd.ParseDatetime("2024-02-01")
		d3, _ := pd.ParseDatetime("2024-03-01")
		l, _ := pd.DataFrameFromRecords([]map[string]any{
			{"d": d1, "l": 1}, {"d": d2, "l": 2},
		}, pd.WithColumnOrder("d", "l"))
		r, _ := pd.DataFrameFromRecords([]map[string]any{
			{"d": d2, "r": 10}, {"d": d3, "r": 20},
		}, pd.WithColumnOrder("d", "r"))
		return l.Merge(r, pd.MergeOptions{On: []string{"d"}, How: "inner"})
	},
	"merge_multi_key_inner": func(t *testing.T) (any, error) {
		l, _ := pd.DataFrameFromRecords([]map[string]any{
			{"country": "AR", "department": "eng", "l": 1},
			{"country": "AR", "department": "sales", "l": 2},
			{"country": "BR", "department": "eng", "l": 3},
		}, pd.WithColumnOrder("country", "department", "l"))
		r, _ := pd.DataFrameFromRecords([]map[string]any{
			{"country": "AR", "department": "eng", "r": 10},
			{"country": "BR", "department": "eng", "r": 30},
		}, pd.WithColumnOrder("country", "department", "r"))
		return l.Merge(r, pd.MergeOptions{On: []string{"country", "department"}, How: "inner"})
	},
	"merge_outer_indicator": func(t *testing.T) (any, error) {
		l, r := mergeFrames(t)
		return l.Merge(r, pd.MergeOptions{On: []string{"id"}, How: "outer", Indicator: true})
	},
	"merge_validate_one_to_one": func(t *testing.T) (any, error) {
		l, r := mergeFrames(t)
		return l.Merge(r, pd.MergeOptions{On: []string{"id"}, How: "inner", Validate: "one_to_one"})
	},
	"merge_cross": func(t *testing.T) (any, error) {
		l, err := pd.DataFrameFromMap(map[string][]any{"x": {1, 2}})
		if err != nil {
			return nil, err
		}
		r, err := pd.DataFrameFromMap(map[string][]any{"y": {"a", "b"}})
		if err != nil {
			return nil, err
		}
		return l.Merge(r, pd.MergeOptions{How: "cross"})
	},
	"concat_rows": func(t *testing.T) (any, error) {
		a, b := concatFixtures(t)
		return pd.Concat([]*pd.DataFrame{a, b}, pd.IgnoreIndex(true))
	},
	"concat_union": func(t *testing.T) (any, error) {
		a, _ := concatFixtures(t)
		c, err := pd.DataFrameFromRecords([]map[string]any{
			{"x": 4, "z": true},
		}, pd.WithColumnOrder("x", "z"))
		if err != nil {
			return nil, err
		}
		return pd.Concat([]*pd.DataFrame{a, c}, pd.IgnoreIndex(true))
	},
	"concat_join_inner": func(t *testing.T) (any, error) {
		a, _ := concatFixtures(t)
		c, err := pd.DataFrameFromRecords([]map[string]any{
			{"x": 4, "z": true},
		}, pd.WithColumnOrder("x", "z"))
		if err != nil {
			return nil, err
		}
		return pd.Concat([]*pd.DataFrame{a, c}, pd.Join("inner"), pd.IgnoreIndex(true))
	},
	"concat_axis1": func(t *testing.T) (any, error) {
		a, err := pd.DataFrameFromMap(map[string][]any{"a": {1, 2}})
		if err != nil {
			return nil, err
		}
		b, err := pd.DataFrameFromMap(map[string][]any{"b": {"x", "y"}})
		if err != nil {
			return nil, err
		}
		return pd.Concat([]*pd.DataFrame{a, b}, pd.ConcatAxis(1))
	},
	"concat_promotion": func(t *testing.T) (any, error) {
		a, err := pd.DataFrameFromMap(map[string][]any{"v": {1, 2}})
		if err != nil {
			return nil, err
		}
		b, err := pd.DataFrameFromMap(map[string][]any{"v": {2.5}})
		if err != nil {
			return nil, err
		}
		return pd.Concat([]*pd.DataFrame{a, b}, pd.IgnoreIndex(true))
	},
	"join_index": func(t *testing.T) (any, error) {
		l, err := pd.DataFrameFromMap(map[string][]any{"v": {1, 2}})
		if err != nil {
			return nil, err
		}
		r, err := pd.DataFrameFromMap(map[string][]any{"w": {10, 20}})
		if err != nil {
			return nil, err
		}
		return l.Join(r, pd.JoinOptions{})
	},

	// reshape ------------------------------------------------------------------
	"melt": func(t *testing.T) (any, error) {
		return gradesFrame(t).Melt(pd.MeltOptions{IDVars: []string{"name"}})
	},
	"melt_value_vars": func(t *testing.T) (any, error) {
		return gradesFrame(t).Melt(pd.MeltOptions{IDVars: []string{"name"}, ValueVars: []string{"math"}})
	},
	"pivot": func(t *testing.T) (any, error) {
		long, err := gradesFrame(t).Melt(pd.MeltOptions{IDVars: []string{"name"}})
		if err != nil {
			return nil, err
		}
		return long.Pivot(pd.PivotOptions{Index: "name", Columns: "variable", Values: "value"})
	},
	"pivot_table_mean": func(t *testing.T) (any, error) {
		dup, err := pd.DataFrameFromRecords([]map[string]any{
			{"country": "AR", "dept": "eng", "salary": 1000.0},
			{"country": "AR", "dept": "eng", "salary": 2000.0},
			{"country": "AR", "dept": "sales", "salary": 800.0},
			{"country": "BR", "dept": "eng", "salary": 1500.0},
		}, pd.WithColumnOrder("country", "dept", "salary"))
		if err != nil {
			return nil, err
		}
		return dup.PivotTable(pd.PivotTableOptions{
			Index:     []string{"country"},
			Columns:   []string{"dept"},
			Values:    []string{"salary"},
			AggFunc:   "mean",
			FillValue: 0.0,
		})
	},

	// missing values --------------------------------------------------------------
	"isna_frame": func(t *testing.T) (any, error) { return missingFrame(t).IsNA(), nil },
	"dropna_any": func(t *testing.T) (any, error) { return missingFrame(t).DropNA(), nil },
	"dropna_all": func(t *testing.T) (any, error) {
		return missingFrame(t).DropNA(pd.DropNAHow("all")), nil
	},
	"dropna_thresh_2": func(t *testing.T) (any, error) {
		return missingFrame(t).DropNA(pd.DropNAThresh(2)), nil
	},
	"dropna_subset_a": func(t *testing.T) (any, error) {
		return missingFrame(t).DropNA(pd.DropNASubset("a")), nil
	},
	"fillna_map": func(t *testing.T) (any, error) {
		return missingFrame(t).FillNA(map[string]any{"a": 0, "b": "?", "c": 0.0})
	},
	"notna_series": func(t *testing.T) (any, error) {
		return missingFrame(t).MustCol("a").NotNA(), nil
	},

	// datetime ------------------------------------------------------------------
	"year":           dtCase(func(s *pd.Series) *pd.Series { return s.Dt().Year() }),
	"month":          dtCase(func(s *pd.Series) *pd.Series { return s.Dt().Month() }),
	"day":            dtCase(func(s *pd.Series) *pd.Series { return s.Dt().Day() }),
	"hour":           dtCase(func(s *pd.Series) *pd.Series { return s.Dt().Hour() }),
	"minute":         dtCase(func(s *pd.Series) *pd.Series { return s.Dt().Minute() }),
	"second":         dtCase(func(s *pd.Series) *pd.Series { return s.Dt().Second() }),
	"weekday":        dtCase(func(s *pd.Series) *pd.Series { return s.Dt().Weekday() }),
	"dayofyear":      dtCase(func(s *pd.Series) *pd.Series { return s.Dt().DayOfYear() }),
	"quarter":        dtCase(func(s *pd.Series) *pd.Series { return s.Dt().Quarter() }),
	"is_month_start": dtCase(func(s *pd.Series) *pd.Series { return s.Dt().IsMonthStart() }),
	"is_month_end":   dtCase(func(s *pd.Series) *pd.Series { return s.Dt().IsMonthEnd() }),
	"is_year_start":  dtCase(func(s *pd.Series) *pd.Series { return s.Dt().IsYearStart() }),
	"is_year_end":    dtCase(func(s *pd.Series) *pd.Series { return s.Dt().IsYearEnd() }),

	// string accessor ----------------------------------------------------------
	"contains_o":   strCase(func(s *pd.Series) *pd.Series { return s.Str().Contains("o") }),
	"lower":        strCase(func(s *pd.Series) *pd.Series { return s.Str().Lower() }),
	"upper":        strCase(func(s *pd.Series) *pd.Series { return s.Str().Upper() }),
	"len":          strCase(func(s *pd.Series) *pd.Series { return s.Str().Len() }),
	"strip":        strCase(func(s *pd.Series) *pd.Series { return s.Str().Strip() }),
	"replace_l_L":  strCase(func(s *pd.Series) *pd.Series { return s.Str().Replace("l", "L") }),
	"startswith_A": strCase(func(s *pd.Series) *pd.Series { return s.Str().HasPrefix("A") }),
	"endswith_d":   strCase(func(s *pd.Series) *pd.Series { return s.Str().HasSuffix("d") }),
	"get_0":        strCase(func(s *pd.Series) *pd.Series { return s.Str().Get(0) }),
	"slice_1_3":    strCase(func(s *pd.Series) *pd.Series { return s.Str().Slice(1, 3) }),

	// rolling / expanding ---------------------------------------------------------
	"rolling_mean_3": func(t *testing.T) (any, error) { return rollingSeries().Rolling(3).Mean() },
	"rolling_sum_3":  func(t *testing.T) (any, error) { return rollingSeries().Rolling(3).Sum() },
	"rolling_min_periods_1": func(t *testing.T) (any, error) {
		return rollingSeries().Rolling(3, pd.MinPeriods(1)).Mean()
	},
	"rolling_std_3":    func(t *testing.T) (any, error) { return rollingSeries().Rolling(3).Std() },
	"rolling_median_3": func(t *testing.T) (any, error) { return rollingSeries().Rolling(3).Median() },
	"rolling_max_2":    func(t *testing.T) (any, error) { return rollingSeries().Rolling(2).Max() },
	"expanding_mean":   func(t *testing.T) (any, error) { return rollingSeries().Expanding().Mean() },
	"expanding_sum":    func(t *testing.T) (any, error) { return rollingSeries().Expanding().Sum() },
	"df_rolling_mean_2": func(t *testing.T) (any, error) {
		df, err := pd.DataFrameFromMap(map[string][]any{
			"open":  {1.0, 2.0, 3.0, 4.0},
			"close": {2.0, 3.0, 4.0, 5.0},
		}, pd.WithColumnOrder("open", "close"))
		if err != nil {
			return nil, err
		}
		return df.Rolling(2).Mean()
	},

	// dtypes (v0.3 typed storage) -----------------------------------------------
	"dtype_series_int": func(t *testing.T) (any, error) {
		return pd.SeriesOf("v", []int{1, 2, 3}), nil
	},
	"dtype_series_int_na": func(t *testing.T) (any, error) {
		return pd.NewSeries("v", []any{1, nil, 3}), nil
	},
	"dtype_series_mixed_na": func(t *testing.T) (any, error) {
		return pd.NewSeries("v", []any{1, 2.5, nil}), nil
	},
	"dtype_series_bool": func(t *testing.T) (any, error) {
		return pd.BoolSeries("v", []bool{true, false}), nil
	},
	"dtype_series_string": func(t *testing.T) (any, error) {
		return pd.StringSeries("v", []string{"a", "b"}), nil
	},
	"dtype_frame_int_na": func(t *testing.T) (any, error) {
		df, err := pd.DataFrameFromMap(map[string][]any{"age": {1, nil, 3}})
		if err != nil {
			return nil, err
		}
		return df.Col("age")
	},
	"dtype_astype_float": func(t *testing.T) (any, error) {
		df, err := pd.DataFrameFromMap(map[string][]any{"age": {1, nil, 3}})
		if err != nil {
			return nil, err
		}
		converted, err := df.Astype(map[string]pd.DType{"age": pd.Float64})
		if err != nil {
			return nil, err
		}
		return converted.Col("age")
	},
	"dtype_to_datetime": func(t *testing.T) (any, error) {
		return pd.ToDatetime(pd.StringSeries("d", []string{"2024-01-02"}))
	},

	// expressions (v0.4 columnar engine) ------------------------------------------
	"expr_filter_gt": func(t *testing.T) (any, error) {
		return peopleFrame(t).Where(pd.Col("age").Gt(30))
	},
	"expr_filter_and": func(t *testing.T) (any, error) {
		return peopleFrame(t).Where(pd.And(pd.Col("age").Gt(30), pd.Col("salary").Lt(2000)))
	},
	"expr_filter_or_not": func(t *testing.T) (any, error) {
		return peopleFrame(t).Where(pd.Or(pd.Col("age").Ge(40), pd.Not(pd.Col("dept").Eq("eng"))))
	},
	"expr_filter_contains": func(t *testing.T) (any, error) {
		return peopleFrame(t).Where(pd.Col("name").Contains("a"))
	},
	"expr_filter_isin": func(t *testing.T) (any, error) {
		return peopleFrame(t).Where(pd.Col("country").IsIn("BR"))
	},
	"expr_assign_total": func(t *testing.T) (any, error) {
		return shopFrame(t).AssignExpr("total", pd.Col("price").Mul(pd.Col("qty")))
	},
	"expr_assign_flag": func(t *testing.T) (any, error) {
		df, err := peopleFrame(t).AssignExpr("flag", pd.Col("age").Gt(30))
		if err != nil {
			return nil, err
		}
		return df.Select("name", "flag")
	},
	"expr_assign_ratio": func(t *testing.T) (any, error) {
		df, err := shopFrame(t).AssignExpr("r", pd.Col("price").Div(pd.Col("qty")))
		if err != nil {
			return nil, err
		}
		return df.Select("item", "r")
	},
	"expr_query_gt": func(t *testing.T) (any, error) {
		return peopleFrame(t).Query("age > 30")
	},
	"expr_query_and": func(t *testing.T) (any, error) {
		return peopleFrame(t).Query("age > 30 and salary < 2000")
	},

	// io ------------------------------------------------------------------------
	"read_csv_basic": csvCase("name,age,score\nAna,30,9.5\nLuis,40,8.0\n"),
	"read_csv_na":    csvCase("a,b\n1,x\nNA,y\n3,\n"),
	"read_csv_semicolon": csvCase("a;b\n1;x\n2;y\n",
		pd.WithComma(';')),
	"read_csv_usecols": csvCase("a,b,c\n1,2,3\n4,5,6\n",
		pd.WithUseCols("a", "c")),
	"read_csv_nrows": csvCase("a,b,c\n1,2,3\n4,5,6\n",
		pd.WithNRows(1)),
	"read_csv_parse_dates": csvCase("day,v\n2024-01-02,1\n2024-02-03,2\n",
		pd.WithParseDates("day")),
	"read_csv_no_header": csvCase("1,x\n2,y\n",
		pd.WithHeader(false)),
}

func scalarCase(f func() (float64, error)) caseFn {
	return func(t *testing.T) (any, error) {
		v, err := f()
		return v, err
	}
}

func mergeCase(how string) caseFn {
	return func(t *testing.T) (any, error) {
		left, right := mergeFrames(t)
		return left.Merge(right, pd.MergeOptions{On: []string{"id"}, How: how})
	}
}

func concatFixtures(t *testing.T) (*pd.DataFrame, *pd.DataFrame) {
	t.Helper()
	a, err := pd.DataFrameFromRecords([]map[string]any{
		{"x": 1, "y": "a"}, {"x": 2, "y": "b"},
	}, pd.WithColumnOrder("x", "y"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := pd.DataFrameFromRecords([]map[string]any{
		{"x": 3, "y": "c"},
	}, pd.WithColumnOrder("x", "y"))
	if err != nil {
		t.Fatal(err)
	}
	return a, b
}

func dtCase(f func(s *pd.Series) *pd.Series) caseFn {
	return func(t *testing.T) (any, error) { return f(dateSeries()), nil }
}

func strCase(f func(s *pd.Series) *pd.Series) caseFn {
	return func(t *testing.T) (any, error) { return f(strSeries()), nil }
}

func csvCase(csv string, opts ...pd.CSVOption) caseFn {
	return func(t *testing.T) (any, error) {
		return pd.ReadCSVReader(strings.NewReader(csv), opts...)
	}
}
