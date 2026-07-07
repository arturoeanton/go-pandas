package benchmarks

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/index"
)

func takePositions(n, every int) []int {
	pos := make([]int, 0, n/every)
	for i := 0; i < n; i += every {
		pos = append(pos, i)
	}
	return pos
}

func BenchmarkDataFrameTake100K(b *testing.B) {
	df := exprFrame(b, false)
	pos := takePositions(100_000, 3)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Take(pos); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDataFrameWhereStringColumnar100K(b *testing.B) {
	df := exprFrame(b, false)
	pred := pd.Col("name").Contains("a")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Where(pred); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSeriesTake100K(b *testing.B) {
	df := exprFrame(b, false)
	s := df.MustCol("salary")
	pos := takePositions(100_000, 3)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Take(pos); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIndexTakeRange100K(b *testing.B) {
	idx := index.NewRangeIndex(100_000)
	pos := takePositions(100_000, 3)
	pos[1] = 5 // force the irregular (Int64Index) path
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = index.Take(idx, pos)
	}
}

func BenchmarkIndexTakeString100K(b *testing.B) {
	labels := make([]string, 100_000)
	for i := range labels {
		labels[i] = "row-" + string(rune('a'+i%26))
	}
	idx := index.NewStringIndex(labels)
	pos := takePositions(100_000, 3)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = index.Take(idx, pos)
	}
}

func BenchmarkPositionsFromMask100K(b *testing.B) {
	n := 100_000
	mask := &expr.Mask{Data: make([]bool, n), NA: make([]bool, n)}
	for i := 0; i < n; i++ {
		mask.Data[i] = i%3 == 0
		mask.NA[i] = i%17 == 0
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = expr.PositionsFromMask(mask)
	}
}

func BenchmarkDropNA100K(b *testing.B) {
	n := 100_000
	values := make([]any, n)
	for i := 0; i < n; i++ {
		if i%5 == 0 {
			continue // nil
		}
		values[i] = float64(i)
	}
	df, err := pd.DataFrameFromMap(map[string][]any{"v": values})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = df.DropNA()
	}
}
