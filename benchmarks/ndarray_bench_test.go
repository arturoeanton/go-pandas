package benchmarks

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

func BenchmarkNDArrayAdd1M(b *testing.B) {
	x := pd.Arange(1_000_000)
	y := pd.Ones(1_000_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := x.Add(y); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNDArrayBroadcast1M(b *testing.B) {
	x, err := pd.Arange(1_000_000).Reshape(1000, 1000)
	if err != nil {
		b.Fatal(err)
	}
	row := pd.Ones(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := x.Add(row); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNDArrayMatMul100x100(b *testing.B) {
	x, err := pd.Arange(10_000).Reshape(100, 100)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pd.MatMul(x, x); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNDArraySum1M(b *testing.B) {
	x := pd.Arange(1_000_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = x.SumAll()
	}
}
