package benchmarks

import (
	"testing"
	"time"

	pd "github.com/arturoeanton/go-pandas"
)

// tsBenchStrings builds 100K datetime strings over ~70 days.
func tsBenchStrings(b *testing.B) *pd.Series {
	b.Helper()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	strs := make([]string, 100_000)
	for i := range strs {
		strs[i] = base.Add(time.Duration(i) * time.Minute).Format("2006-01-02 15:04:05")
	}
	return pd.StringSeries("date", strs)
}

// tsBenchIndexed builds a 100K-row frame on a DatetimeIndex; shuffled
// controls whether timestamps arrive unsorted.
func tsBenchIndexed(b *testing.B, shuffled bool) *pd.DataFrame {
	b.Helper()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	n := 100_000
	times := make([]time.Time, n)
	vals := make([]float64, n)
	for i := 0; i < n; i++ {
		k := i
		if shuffled {
			k = (i * 999983) % n // deterministic permutation
		}
		times[i] = base.Add(time.Duration(k) * time.Minute)
		vals[i] = float64(k % 1000)
	}
	df, err := pd.NewDataFrame(pd.TimeSeries("date", times), pd.FloatSeries("v", vals))
	if err != nil {
		b.Fatal(err)
	}
	indexed, err := df.SetIndex("date")
	if err != nil {
		b.Fatal(err)
	}
	return indexed
}

func BenchmarkToDatetimeFormat100K(b *testing.B) {
	s := tsBenchStrings(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pd.ToDatetime(s, pd.WithDatetimeFormat("%Y-%m-%d %H:%M:%S")); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkToDatetimeInfer100K(b *testing.B) {
	s := tsBenchStrings(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pd.ToDatetime(s); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDatetimeIndexLookup100K(b *testing.B) {
	df := tsBenchIndexed(b, false)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	targets := make([]time.Time, 64)
	for i := range targets {
		targets[i] = base.Add(time.Duration((i*997)%100_000) * time.Minute)
	}
	idx := df.Index()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := idx.Positions(targets[i%len(targets)]); len(got) == 0 {
			b.Fatal("label not found")
		}
	}
}

func BenchmarkResampleDailySum100K(b *testing.B) {
	df := tsBenchIndexed(b, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Resample("D").Sum(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResampleHourlyMean100K(b *testing.B) {
	df := tsBenchIndexed(b, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Resample("H").Mean(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResampleMonthlyCount100K(b *testing.B) {
	df := tsBenchIndexed(b, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Resample("MS").Count(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResampleUnsorted100K(b *testing.B) {
	df := tsBenchIndexed(b, true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Resample("D").Sum(); err != nil {
			b.Fatal(err)
		}
	}
}
