package benchmarks

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

func concatBenchFrame(b *testing.B, n int, withExtra, object bool) *pd.DataFrame {
	b.Helper()
	v := make([]any, n)
	s := make([]any, n)
	for i := 0; i < n; i++ {
		v[i] = float64(i)
		s[i] = "row"
	}
	cols := map[string][]any{"v": v, "s": s}
	order := []string{"v", "s"}
	if withExtra {
		cols["extra"] = v
		order = append(order, "extra")
	}
	df, err := pd.DataFrameFromMap(cols, pd.WithColumnOrder(order...))
	if err != nil {
		b.Fatal(err)
	}
	if object {
		for _, name := range []string{"v", "s"} {
			obj := pd.NewSeries(name, df.MustCol(name).Values(), pd.WithDType(pd.Object))
			df, _ = df.Assign(name, obj)
		}
	}
	return df
}

func BenchmarkConcatAxis0SameSchema100K(b *testing.B) {
	a := concatBenchFrame(b, 100_000, false, false)
	c := concatBenchFrame(b, 100_000, false, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pd.Concat([]*pd.DataFrame{a, c}, pd.IgnoreIndex(true)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcatAxis0OuterMissingColumns100K(b *testing.B) {
	a := concatBenchFrame(b, 100_000, true, false)
	c := concatBenchFrame(b, 100_000, false, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pd.Concat([]*pd.DataFrame{a, c}, pd.IgnoreIndex(true)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcatAxis0NumericPromotion100K(b *testing.B) {
	n := 100_000
	iv := make([]any, n)
	for i := range iv {
		iv[i] = i
	}
	ints, _ := pd.DataFrameFromMap(map[string][]any{"v": iv})
	floats := concatBenchFrame(b, n, false, false)
	sel, _ := floats.Select("v")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pd.Concat([]*pd.DataFrame{ints, sel}, pd.IgnoreIndex(true)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcatAxis1Aligned100K(b *testing.B) {
	a := concatBenchFrame(b, 100_000, false, false)
	c, _ := concatBenchFrame(b, 100_000, false, false).Rename(map[string]string{"v": "v2", "s": "s2"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pd.Concat([]*pd.DataFrame{a, c}, pd.ConcatAxis(1)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcatObjectFallback100K(b *testing.B) {
	a := concatBenchFrame(b, 100_000, false, true)
	c := concatBenchFrame(b, 100_000, false, true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pd.Concat([]*pd.DataFrame{a, c}, pd.IgnoreIndex(true)); err != nil {
			b.Fatal(err)
		}
	}
}
