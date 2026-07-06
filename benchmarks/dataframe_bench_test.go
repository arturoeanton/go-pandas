// Package benchmarks measures the hot paths. Run with:
//
//	go test ./benchmarks/ -bench=. -benchmem
package benchmarks

import (
	"fmt"
	"strings"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

func buildFrame(b *testing.B, n int) *pd.DataFrame {
	b.Helper()
	countries := []string{"AR", "BR", "CL", "UY", "PY"}
	ids := make([]any, n)
	country := make([]any, n)
	salary := make([]any, n)
	for i := 0; i < n; i++ {
		ids[i] = i
		country[i] = countries[i%len(countries)]
		salary[i] = float64(800 + i%2000)
	}
	df, err := pd.DataFrameFromMap(map[string][]any{
		"id":      ids,
		"country": country,
		"salary":  salary,
	}, pd.WithColumnOrder("id", "country", "salary"))
	if err != nil {
		b.Fatal(err)
	}
	return df
}

func benchmarkFilter(b *testing.B, n int) {
	df := buildFrame(b, n)
	pred := pd.Col("salary").Gt(1500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Where(pred); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDataFrameFilter1K(b *testing.B)   { benchmarkFilter(b, 1_000) }
func BenchmarkDataFrameFilter100K(b *testing.B) { benchmarkFilter(b, 100_000) }

func BenchmarkDataFrameGroupBy100K(b *testing.B) {
	df := buildFrame(b, 100_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country").Mean("salary"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDataFrameMerge100K(b *testing.B) {
	left := buildFrame(b, 100_000)
	rightIDs := make([]any, 1000)
	bonus := make([]any, 1000)
	for i := range rightIDs {
		rightIDs[i] = i * 100
		bonus[i] = float64(i)
	}
	right, err := pd.DataFrameFromMap(map[string][]any{
		"id":    rightIDs,
		"bonus": bonus,
	}, pd.WithColumnOrder("id", "bonus"))
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := left.Merge(right, pd.MergeOptions{On: []string{"id"}, How: "left"}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadCSV100K(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("id,country,salary\n")
	for i := 0; i < 100_000; i++ {
		fmt.Fprintf(&sb, "%d,AR,%d.5\n", i, 800+i%2000)
	}
	csv := sb.String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pd.ReadCSVReader(strings.NewReader(csv)); err != nil {
			b.Fatal(err)
		}
	}
}
