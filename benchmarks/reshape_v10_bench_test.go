package benchmarks

import (
	"fmt"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/ndarray"
)

// v10BenchFrame: 100K rows, 8 keys x 12 sub-keys, two numeric columns.
func v10BenchFrame(b *testing.B) *pd.DataFrame {
	b.Helper()
	n := 100_000
	keys := make([]any, n)
	sub := make([]any, n)
	sales := make([]any, n)
	qty := make([]any, n)
	countries := []string{"AR", "BR", "CL", "UY", "PY", "PE", "BO", "EC"}
	for i := 0; i < n; i++ {
		keys[i] = countries[i%len(countries)]
		sub[i] = fmt.Sprintf("m%02d", i%12)
		sales[i] = float64(i % 1000)
		qty[i] = float64(i % 50)
	}
	df, err := pd.DataFrameFromMap(
		map[string][]any{"country": keys, "month": sub, "sales": sales, "qty": qty},
		pd.WithColumnOrder("country", "month", "sales", "qty"))
	if err != nil {
		b.Fatal(err)
	}
	return df
}

func BenchmarkStack100K(b *testing.B) {
	df := v10BenchFrame(b)
	numeric, err := df.Select("sales", "qty")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := numeric.Stack(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnstack100K(b *testing.B) {
	df := v10BenchFrame(b)
	numeric, _ := df.Select("sales", "qty")
	stacked, err := numeric.Stack()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pd.UnstackSeries(stacked); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPivotTable100K(b *testing.B) {
	df := v10BenchFrame(b)
	opts := pd.PivotTableOptions{
		Values: []string{"sales", "qty"}, Index: []string{"country"},
		Columns: []string{"month"}, AggFuncs: []string{"sum", "mean"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.PivotTable(opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGroupByTransformMean100K(b *testing.B) {
	df := v10BenchFrame(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country").Transform("sales", "mean"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGroupByFilter100K(b *testing.B) {
	df := v10BenchFrame(b)
	cond := pd.GroupSize().Gt(10_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country").Filter(cond); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryArithmetic100K(b *testing.B) {
	df := v10BenchFrame(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Query("sales + qty > 900"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryInList100K(b *testing.B) {
	df := v10BenchFrame(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Query(`country in ["AR", "BR"]`); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNDArrayIsIn100K(b *testing.B) {
	data := make([]float64, 100_000)
	for i := range data {
		data[i] = float64(i % 977)
	}
	a := ndarray.Array(data)
	candidates := []any{1.0, 500.0, 976.0}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.IsIn(candidates)
	}
}

func BenchmarkNDArraySearchSorted100K(b *testing.B) {
	data := make([]float64, 100_000)
	for i := range data {
		data[i] = float64(i)
	}
	a := ndarray.Array(data)
	queries := make([]float64, 100)
	for i := range queries {
		queries[i] = float64(i * 997)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := a.SearchSorted(queries, "left"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNDArrayTake100K(b *testing.B) {
	data := make([]float64, 100_000)
	for i := range data {
		data[i] = float64(i)
	}
	a := ndarray.Array(data)
	indices := make([]int, 100_000)
	for i := range indices {
		indices[i] = (i * 7) % len(data)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := a.Take(indices, 0); err != nil {
			b.Fatal(err)
		}
	}
}
