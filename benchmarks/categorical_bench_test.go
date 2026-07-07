package benchmarks

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

// catBenchColumns builds a 500K-row low-cardinality key as plain strings
// and as categorical, plus a value column.
func catBenchColumns(b *testing.B) (str, cat, vals *pd.Series) {
	b.Helper()
	n := 500_000
	countries := []string{"AR", "BR", "CL", "UY", "PY", "PE", "BO", "EC"}
	keys := make([]string, n)
	nums := make([]float64, n)
	for i := 0; i < n; i++ {
		keys[i] = countries[i%len(countries)]
		nums[i] = float64(800 + i%2000)
	}
	str = pd.StringSeries("country", keys)
	var err error
	if cat, err = str.Astype(pd.Category); err != nil {
		b.Fatal(err)
	}
	return str, cat, pd.FloatSeries("salary", nums)
}

func benchFrame(b *testing.B, key, vals *pd.Series) *pd.DataFrame {
	b.Helper()
	df, err := pd.NewDataFrame(key, vals)
	if err != nil {
		b.Fatal(err)
	}
	return df
}

func BenchmarkGroupByMeanStringKey500K(b *testing.B) {
	str, _, vals := catBenchColumns(b)
	df := benchFrame(b, str, vals)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country").Mean("salary"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGroupByMeanCategoricalKey500K(b *testing.B) {
	_, cat, vals := catBenchColumns(b)
	df := benchFrame(b, cat, vals)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country").Mean("salary"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSortValuesStringKey500K(b *testing.B) {
	str, _, _ := catBenchColumns(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		str.SortValues(true)
	}
}

func BenchmarkSortValuesCategoricalKey500K(b *testing.B) {
	_, cat, _ := catBenchColumns(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cat.SortValues(true)
	}
}

func BenchmarkValueCountsStringKey500K(b *testing.B) {
	str, _, _ := catBenchColumns(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		str.ValueCounts()
	}
}

func BenchmarkValueCountsCategoricalKey500K(b *testing.B) {
	_, cat, _ := catBenchColumns(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cat.ValueCounts()
	}
}

// mergeBenchSides builds a 200K-row left and 8-row right dimension table.
func mergeBenchSides(b *testing.B, categorical bool) (*pd.DataFrame, *pd.DataFrame) {
	b.Helper()
	n := 200_000
	countries := []string{"AR", "BR", "CL", "UY", "PY", "PE", "BO", "EC"}
	keys := make([]string, n)
	vals := make([]int, n)
	for i := 0; i < n; i++ {
		keys[i] = countries[i%len(countries)]
		vals[i] = i
	}
	names := []string{"Argentina", "Brasil", "Chile", "Uruguay", "Paraguay", "Peru", "Bolivia", "Ecuador"}
	lk := pd.StringSeries("country", keys)
	rk := pd.StringSeries("country", countries)
	if categorical {
		var err error
		if lk, err = lk.Astype(pd.Category); err != nil {
			b.Fatal(err)
		}
		if rk, err = rk.Astype(pd.Category); err != nil {
			b.Fatal(err)
		}
	}
	left, err := pd.NewDataFrame(lk, pd.IntSeries("v", vals))
	if err != nil {
		b.Fatal(err)
	}
	right, err := pd.NewDataFrame(rk, pd.StringSeries("name", names))
	if err != nil {
		b.Fatal(err)
	}
	return left, right
}

func BenchmarkMergeInnerStringKey200K(b *testing.B) {
	left, right := mergeBenchSides(b, false)
	opts := pd.MergeOptions{On: []string{"country"}, How: "inner"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := left.Merge(right, opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMergeInnerCategoricalKey200K(b *testing.B) {
	left, right := mergeBenchSides(b, true)
	opts := pd.MergeOptions{On: []string{"country"}, How: "inner"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := left.Merge(right, opts); err != nil {
			b.Fatal(err)
		}
	}
}

// The memory pair measures the storage footprint of each representation:
// Copy materializes the full backing buffers, so bytes/op is the resident
// size of a 500K-row low-cardinality column (string headers vs int32
// codes sharing one small category list).
func BenchmarkMemoryStringStorage500K(b *testing.B) {
	str, _, _ := catBenchColumns(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if str.Copy().Len() != 500_000 {
			b.Fatal("bad length")
		}
	}
}

func BenchmarkMemoryCategoricalStorage500K(b *testing.B) {
	_, cat, _ := catBenchColumns(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if cat.Copy().Len() != 500_000 {
			b.Fatal("bad length")
		}
	}
}
