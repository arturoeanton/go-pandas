package benchmarks

import (
	"fmt"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/index"
)

// miBenchArrays builds 100K rows over 8 countries x 50 cities.
func miBenchArrays(b *testing.B) ([][]any, *pd.DataFrame) {
	b.Helper()
	n := 100_000
	countries := []string{"AR", "BR", "CL", "UY", "PY", "PE", "BO", "EC"}
	l0 := make([]any, n)
	l1 := make([]any, n)
	vals := make([]any, n)
	for i := 0; i < n; i++ {
		l0[i] = countries[i%len(countries)]
		l1[i] = fmt.Sprintf("city-%02d", i%50)
		vals[i] = float64(800 + i%2000)
	}
	df, err := pd.DataFrameFromMap(
		map[string][]any{"country": l0, "city": l1, "salary": vals},
		pd.WithColumnOrder("country", "city", "salary"))
	if err != nil {
		b.Fatal(err)
	}
	return [][]any{l0, l1}, df
}

func miBenchIndex(b *testing.B) *index.MultiIndex {
	b.Helper()
	arrays, _ := miBenchArrays(b)
	mi, err := index.NewMultiIndexFromArrays(arrays, []string{"country", "city"})
	if err != nil {
		b.Fatal(err)
	}
	return mi
}

func BenchmarkMultiIndexBuild100K(b *testing.B) {
	arrays, _ := miBenchArrays(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := index.NewMultiIndexFromArrays(arrays, []string{"country", "city"}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMultiIndexTake100K(b *testing.B) {
	mi := miBenchIndex(b)
	positions := make([]int, mi.Len())
	for i := range positions {
		positions[i] = (i * 7) % mi.Len()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if mi.Take(positions).Len() != len(positions) {
			b.Fatal("bad take")
		}
	}
}

func BenchmarkMultiIndexFullTupleLookup100K(b *testing.B) {
	mi := miBenchIndex(b)
	mi.PositionsTuple([]any{"AR", "city-00"}) // build the lookup once
	queries := make([][]any, 64)
	for i := range queries {
		queries[i] = mi.Tuple((i * 997) % mi.Len())
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if len(mi.PositionsTuple(queries[i%len(queries)])) == 0 {
			b.Fatal("tuple not found")
		}
	}
}

func BenchmarkMultiIndexPrefixLookup100K(b *testing.B) {
	mi := miBenchIndex(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if len(mi.PositionsPrefix([]any{"AR"})) == 0 {
			b.Fatal("prefix not found")
		}
	}
}

func BenchmarkSetIndexMultiColumn100K(b *testing.B) {
	_, df := miBenchArrays(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.SetIndex("country", "city"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResetIndexMultiIndex100K(b *testing.B) {
	_, df := miBenchArrays(b)
	indexed, err := df.SetIndex("country", "city")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if indexed.ResetIndex().Len() != df.Len() {
			b.Fatal("bad reset")
		}
	}
}

func BenchmarkWherePreserveMultiIndex100K(b *testing.B) {
	_, df := miBenchArrays(b)
	indexed, err := df.SetIndex("country", "city")
	if err != nil {
		b.Fatal(err)
	}
	pred := pd.Col("salary").Gt(1800.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := indexed.Where(pred); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGroupByAsIndexMultiIndex100K(b *testing.B) {
	_, df := miBenchArrays(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country", "city").AsIndex(true).Mean("salary"); err != nil {
			b.Fatal(err)
		}
	}
}
