package benchmarks

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

// groupBenchFrame builds a 100K-row frame with string/int/multi keys.
func groupBenchFrame(b *testing.B, object bool) *pd.DataFrame {
	b.Helper()
	n := 100_000
	countries := []string{"AR", "BR", "CL", "UY", "PY", "PE", "BO", "EC"}
	skey := make([]any, n)
	ikey := make([]any, n)
	user := make([]any, n)
	salary := make([]any, n)
	for i := 0; i < n; i++ {
		skey[i] = countries[i%len(countries)]
		ikey[i] = i % 50
		user[i] = i % 1000
		salary[i] = float64(800 + i%2000)
	}
	df, err := pd.DataFrameFromMap(map[string][]any{
		"country": skey, "dept": ikey, "user_id": user, "salary": salary,
	}, pd.WithColumnOrder("country", "dept", "user_id", "salary"))
	if err != nil {
		b.Fatal(err)
	}
	if object {
		for _, col := range []string{"country", "salary"} {
			obj := pd.NewSeries(col, df.MustCol(col).Values(), pd.WithDType(pd.Object))
			df, err = df.Assign(col, obj)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
	return df
}

func BenchmarkGroupByStringKeySize100K(b *testing.B) {
	df := groupBenchFrame(b, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country").Size(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGroupByStringKeyMean100K(b *testing.B) {
	df := groupBenchFrame(b, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country").Mean("salary"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGroupByIntKeyMean100K(b *testing.B) {
	df := groupBenchFrame(b, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("dept").Mean("salary"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGroupByMultiKeyMean100K(b *testing.B) {
	df := groupBenchFrame(b, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country", "dept").Mean("salary"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGroupByAggList100K(b *testing.B) {
	df := groupBenchFrame(b, false)
	spec := map[string][]string{
		"salary": {"mean", "max", "min"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country").AggList(spec); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGroupByNUnique100K(b *testing.B) {
	df := groupBenchFrame(b, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country").NUnique("user_id"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGroupByObjectFallback100K(b *testing.B) {
	df := groupBenchFrame(b, true)
	if !df.MustCol("country").IsObjectBacked() {
		b.Fatal("expected object keys")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("country").Mean("salary"); err != nil {
			b.Fatal(err)
		}
	}
}
