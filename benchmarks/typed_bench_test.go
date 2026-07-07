package benchmarks

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

const benchN = 100_000

func intSeriesTyped(b *testing.B) *pd.Series {
	b.Helper()
	data := make([]int, benchN)
	for i := range data {
		data[i] = i % 1000
	}
	s := pd.SeriesOf("v", data)
	if s.IsObjectBacked() {
		b.Fatal("expected typed backing")
	}
	return s
}

func intSeriesObject(b *testing.B) *pd.Series {
	b.Helper()
	values := make([]any, benchN)
	for i := range values {
		values[i] = i % 1000
	}
	s := pd.NewSeries("v", values, pd.WithDType(pd.Object))
	if !s.IsObjectBacked() {
		b.Fatal("expected object backing")
	}
	return s
}

func BenchmarkSeriesIntSumTyped(b *testing.B) {
	s := intSeriesTyped(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Sum(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSeriesIntSumObject(b *testing.B) {
	s := intSeriesObject(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Sum(); err != nil {
			b.Fatal(err)
		}
	}
}

func floatSeries(b *testing.B, object bool) *pd.Series {
	b.Helper()
	if !object {
		data := make([]float64, benchN)
		for i := range data {
			data[i] = float64(i) / 3
		}
		return pd.FloatSeries("v", data)
	}
	values := make([]any, benchN)
	for i := range values {
		values[i] = float64(i) / 3
	}
	return pd.NewSeries("v", values, pd.WithDType(pd.Object))
}

func BenchmarkSeriesFloatMeanTyped(b *testing.B) {
	s := floatSeries(b, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Mean(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSeriesFloatMeanObject(b *testing.B) {
	s := floatSeries(b, true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Mean(); err != nil {
			b.Fatal(err)
		}
	}
}

func groupByFrame(b *testing.B, object bool) *pd.DataFrame {
	b.Helper()
	countries := []string{"AR", "BR", "CL", "UY", "PY"}
	country := make([]any, benchN)
	salary := make([]any, benchN)
	for i := 0; i < benchN; i++ {
		country[i] = countries[i%len(countries)]
		salary[i] = float64(800 + i%2000)
	}
	df, err := pd.DataFrameFromMap(map[string][]any{
		"country": country,
		"salary":  salary,
	}, pd.WithColumnOrder("country", "salary"))
	if err != nil {
		b.Fatal(err)
	}
	if object {
		obj := pd.NewSeries("salary", salary, pd.WithDType(pd.Object))
		df, err = df.Assign("salary", obj)
		if err != nil {
			b.Fatal(err)
		}
	}
	return df
}

func BenchmarkDataFrameGroupByTyped(b *testing.B) {
	df := groupByFrame(b, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country").Mean("salary"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDataFrameGroupByObject(b *testing.B) {
	df := groupByFrame(b, true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country").Mean("salary"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNDArrayIntAdd(b *testing.B) {
	data := make([]int, 1_000_000)
	for i := range data {
		data[i] = i
	}
	x := pd.ArrayInt(data)
	y := pd.ArrayInt(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := x.Add(y); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNDArrayFloat64Add(b *testing.B) {
	x := pd.Arange(1_000_000)
	y := pd.Arange(1_000_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := x.Add(y); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNDArrayAstypeIntToFloat(b *testing.B) {
	data := make([]int, 1_000_000)
	for i := range data {
		data[i] = i
	}
	x := pd.ArrayInt(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := x.Astype(pd.Float64); err != nil {
			b.Fatal(err)
		}
	}
}
