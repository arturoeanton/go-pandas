package benchmarks

import (
	"fmt"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/series"
)

// mergeBenchFrames: 100K-row left, 10K-row right; keys overlap.
func mergeBenchFrames(b *testing.B, stringKeys, object bool) (*pd.DataFrame, *pd.DataFrame) {
	b.Helper()
	n, m := 100_000, 10_000
	lid := make([]any, n)
	rid := make([]any, m)
	lv := make([]any, n)
	rv := make([]any, m)
	for i := 0; i < n; i++ {
		if stringKeys {
			lid[i] = "key-" + fmt.Sprint(i%20_000)
		} else {
			lid[i] = i % 20_000
		}
		lv[i] = float64(i)
	}
	for i := 0; i < m; i++ {
		if stringKeys {
			rid[i] = "key-" + fmt.Sprint(i)
		} else {
			rid[i] = i
		}
		rv[i] = float64(i * 10)
	}
	left, err := pd.DataFrameFromMap(map[string][]any{"id": lid, "lv": lv},
		pd.WithColumnOrder("id", "lv"))
	if err != nil {
		b.Fatal(err)
	}
	right, err := pd.DataFrameFromMap(map[string][]any{"id": rid, "rv": rv},
		pd.WithColumnOrder("id", "rv"))
	if err != nil {
		b.Fatal(err)
	}
	if object {
		objL := pd.NewSeries("id", left.MustCol("id").Values(), pd.WithDType(pd.Object))
		objR := pd.NewSeries("id", right.MustCol("id").Values(), pd.WithDType(pd.Object))
		left, _ = left.Assign("id", objL)
		right, _ = right.Assign("id", objR)
	}
	return left, right
}

func benchMerge(b *testing.B, how string, stringKeys, object bool) {
	left, right := mergeBenchFrames(b, stringKeys, object)
	opts := pd.MergeOptions{On: []string{"id"}, How: how}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := left.Merge(right, opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMergeInnerIntKey100K(b *testing.B)    { benchMerge(b, "inner", false, false) }
func BenchmarkMergeLeftIntKey100K(b *testing.B)     { benchMerge(b, "left", false, false) }
func BenchmarkMergeOuterIntKey100K(b *testing.B)    { benchMerge(b, "outer", false, false) }
func BenchmarkMergeInnerStringKey100K(b *testing.B) { benchMerge(b, "inner", true, false) }
func BenchmarkMergeLeftStringKey100K(b *testing.B)  { benchMerge(b, "left", true, false) }
func BenchmarkMergeObjectFallback100K(b *testing.B) { benchMerge(b, "left", false, true) }

func BenchmarkMergeMultiKey100K(b *testing.B) {
	n, m := 100_000, 10_000
	build := func(count, mod int, valName string) *pd.DataFrame {
		a := make([]any, count)
		k := make([]any, count)
		v := make([]any, count)
		for i := 0; i < count; i++ {
			a[i] = []string{"AR", "BR", "CL", "UY"}[i%4]
			k[i] = i % mod
			v[i] = float64(i)
		}
		df, err := pd.DataFrameFromMap(map[string][]any{"c": a, "k": k, valName: v},
			pd.WithColumnOrder("c", "k", valName))
		if err != nil {
			b.Fatal(err)
		}
		return df
	}
	left := build(n, 5000, "lv")
	right := build(m, 5000, "rv")
	opts := pd.MergeOptions{On: []string{"c", "k"}, How: "inner"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := left.Merge(right, opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMergeDuplicateKeys100K(b *testing.B) {
	// 100K left rows over 100 keys x 10 right rows over the same keys
	n := 100_000
	lid := make([]any, n)
	for i := 0; i < n; i++ {
		lid[i] = i % 100
	}
	rid := make([]any, 1000)
	for i := range rid {
		rid[i] = i % 100
	}
	left, _ := pd.DataFrameFromMap(map[string][]any{"id": lid}, pd.WithColumnOrder("id"))
	right, _ := pd.DataFrameFromMap(map[string][]any{"id": rid}, pd.WithColumnOrder("id"))
	opts := pd.MergeOptions{On: []string{"id"}, How: "inner"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := left.Merge(right, opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMergeIndicator100K(b *testing.B) {
	left, right := mergeBenchFrames(b, false, false)
	opts := pd.MergeOptions{On: []string{"id"}, How: "outer", Indicator: true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := left.Merge(right, opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJoinByRangeIndex100K(b *testing.B) {
	n := 100_000
	lv := make([]any, n)
	rv := make([]any, n)
	for i := 0; i < n; i++ {
		lv[i] = float64(i)
		rv[i] = float64(i * 2)
	}
	left, _ := pd.DataFrameFromMap(map[string][]any{"lv": lv})
	right, _ := pd.DataFrameFromMap(map[string][]any{"rv": rv})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := left.Join(right, pd.JoinOptions{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJoinByStringIndex100K(b *testing.B) {
	n := 100_000
	labels := make([]string, n)
	lv := make([]any, n)
	for i := 0; i < n; i++ {
		labels[i] = "row-" + fmt.Sprint(i)
		lv[i] = float64(i)
	}
	idx := index.NewStringIndex(labels)
	left, err := pd.NewDataFrame(series.NewSeries("lv", lv, series.WithIndex(idx)))
	if err != nil {
		b.Fatal(err)
	}
	right, err := pd.NewDataFrame(series.NewSeries("rv", lv, series.WithIndex(idx)))
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := left.Join(right, pd.JoinOptions{}); err != nil {
			b.Fatal(err)
		}
	}
}
