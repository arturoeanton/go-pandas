// Command tour walks the v0.7–v0.10 feature surface end to end:
// categorical, MultiIndex, to_datetime/resample, stack/unstack,
// pivot_table, query grammar, groupby transform/filter and the NumPy
// set operations. Every snippet here is the documented public API.
package main

import (
	"fmt"

	pd "github.com/arturoeanton/go-pandas"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	// Categorical (v0.7): int32 codes, ordered rank comparisons.
	size, err := pd.CategoricalSeries("size", []string{"m", "s", "l", "m"},
		pd.WithCategories("s", "m", "l"), pd.WithOrdered(true))
	check(err)
	cat, err := size.Cat()
	check(err)
	fmt.Println("categorical:", cat.Categories(), cat.Codes(), "big:", size.Gt("m").AsMask())

	// MultiIndex (v0.8): multi-column SetIndex, tuple Loc.
	sales, err := pd.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "month": "jan", "sales": 10.0, "qty": 1.0},
		{"country": "AR", "month": "feb", "sales": 20.0, "qty": 2.0},
		{"country": "BR", "month": "jan", "sales": 30.0, "qty": 3.0},
		{"country": "AR", "month": "jan", "sales": 40.0, "qty": 4.0},
	}, pd.WithColumnOrder("country", "month", "sales", "qty"))
	check(err)
	indexed, err := sales.SetIndex("country", "month")
	check(err)
	arJan, err := indexed.Loc().Tuple("AR", "jan").Get()
	check(err)
	fmt.Println("multiindex loc rows:", arJan.Len())

	// Stack / Unstack (v0.10).
	stacked, err := indexed.Select("sales")
	check(err)
	s, err := stacked.Stack()
	check(err)
	fmt.Println("stacked len:", s.Len(), "levels:", s.Index().(*pd.MultiIndex).NLevels())

	// PivotTable with multiple values and aggfuncs (v0.10).
	pt, err := sales.PivotTable(pd.PivotTableOptions{
		Values: []string{"sales", "qty"}, Index: []string{"country"},
		Columns: []string{"month"}, AggFuncs: []string{"sum", "mean"},
	})
	check(err)
	fmt.Println("pivot columns:", pt.Columns())

	// Query grammar (v0.10): arithmetic, not in, parentheses.
	q, err := sales.Query(`(sales + qty > 12) and country not in ["CL"]`)
	check(err)
	fmt.Println("query rows:", q.Len())

	// GroupBy Transform / Filter (v0.10).
	tr, err := sales.GroupBy("country").Transform("sales", "mean")
	check(err)
	fmt.Println("transform:", tr.Values())
	fl, err := sales.GroupBy("country").Filter(pd.GroupSize().Gt(1))
	check(err)
	fmt.Println("filter rows:", fl.Len())

	// to_datetime + resample (v0.9).
	dates, err := pd.ToDatetime(pd.StringSeries("date", []string{
		"2026-01-02 10:00:00", "2026-01-01 09:00:00", "2026-01-01 15:00:00",
	}), pd.WithDatetimeFormat("%Y-%m-%d %H:%M:%S"))
	check(err)
	tsf, err := pd.NewDataFrame(dates, pd.FloatSeries("v", []float64{2, 1, 3}))
	check(err)
	byDate, err := tsf.SetIndex("date")
	check(err)
	daily, err := byDate.Resample("D").Sum()
	check(err)
	fmt.Println("daily buckets:", daily.Len())

	// NumPy set ops (v0.10) + typed Take (v0.10.1).
	a := pd.Array([]float64{1, 2, 2, 4, 7})
	fmt.Println("isin:", a.IsIn([]any{2.0, 7.0}).Data())
	pos, err := a.SearchSorted([]float64{3}, "left")
	check(err)
	fmt.Println("searchsorted:", pos)
	taken, err := a.Take([]int{4, 0, 2}, 0)
	check(err)
	fmt.Println("take:", taken.Values())
}
