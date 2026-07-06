package fuzz_test

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

func FuzzDataFrameFromRecords(f *testing.F) {
	f.Add("a", "b", 1, 2.5, true)
	f.Add("", "", 0, 0.0, false)
	f.Add("col", "col", -1, -2.5, true)
	f.Fuzz(func(t *testing.T, k1, k2 string, i int, fl float64, b bool) {
		records := []map[string]any{
			{k1: i, k2: fl},
			{k1: nil, k2: b},
			{},
		}
		df, err := pd.DataFrameFromRecords(records)
		if err != nil {
			return
		}
		if df.Len() != 3 {
			t.Fatalf("frame length = %d, want 3", df.Len())
		}
		_ = df.String()
		_ = df.DTypes()
		_ = df.IsNA()
	})
}

func FuzzDataFrameQuery(f *testing.F) {
	f.Add("age > 30")
	f.Add("age > 30 and name == \"Ana\"")
	f.Add("((((")
	f.Add("in in in")
	f.Add("age.str.contains(\"x\")")
	f.Add("not not not age > 1")
	f.Fuzz(func(t *testing.T, query string) {
		df, err := pd.DataFrameFromRecords([]map[string]any{
			{"age": 30, "name": "Ana"},
			{"age": 40, "name": "Luis"},
		})
		if err != nil {
			t.Fatal(err)
		}
		out, err := df.Query(query)
		if err != nil {
			return
		}
		if out.Len() > df.Len() {
			t.Fatalf("query grew the frame: %d > %d", out.Len(), df.Len())
		}
	})
}
