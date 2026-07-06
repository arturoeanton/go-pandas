package fuzz_test

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

func FuzzSeriesAstype(f *testing.F) {
	f.Add("42", "int64")
	f.Add("abc", "float64")
	f.Add("2024-01-02", "datetime64[ns]")
	f.Add("true", "bool")
	f.Add("", "string")
	f.Add("1e309", "float64")
	f.Fuzz(func(t *testing.T, value, dtypeName string) {
		dt, err := pd.ParseDType(dtypeName)
		if err != nil {
			return
		}
		s := pd.NewSeries("v", []any{value, nil, value + value})
		out, err := s.Astype(dt)
		if err != nil {
			return
		}
		if out.Len() != s.Len() {
			t.Fatalf("astype changed length: %d -> %d", s.Len(), out.Len())
		}
	})
}

func FuzzSeriesOps(f *testing.F) {
	f.Add(1.0, 2.0, 3.0)
	f.Add(0.0, 0.0, 0.0)
	f.Add(-1e308, 1e308, 0.5)
	f.Fuzz(func(t *testing.T, a, b, c float64) {
		s := pd.FloatSeries("v", []float64{a, b, c})
		if _, err := s.Cumsum(); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Rank(); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Diff(1); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Rolling(2).Mean(); err != nil {
			t.Fatal(err)
		}
		sorted := s.SortValues(true)
		if sorted.Len() != 3 {
			t.Fatalf("sort changed length: %d", sorted.Len())
		}
	})
}
