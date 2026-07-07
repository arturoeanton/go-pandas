package fuzz_test

import (
	"strings"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

// FuzzSeriesAstypeTyped: conversions must preserve length, keep masks
// consistent and land in a valid storage dtype.
func FuzzSeriesAstypeTyped(f *testing.F) {
	f.Add(int64(1), 2.5, "3", true)
	f.Add(int64(-9e15), 1e300, "abc", false)
	f.Add(int64(0), 0.0, "", true)
	f.Fuzz(func(t *testing.T, i int64, fl float64, s string, b bool) {
		src := pd.NewSeries("v", []any{i, fl, s, b, nil})
		for _, dt := range []pd.DType{pd.Int64, pd.Float64, pd.String, pd.Bool} {
			out, err := src.Astype(dt)
			if err != nil {
				continue
			}
			if out.Len() != src.Len() {
				t.Fatalf("astype changed length: %d -> %d", src.Len(), out.Len())
			}
			if v, _ := out.At(4); v != nil {
				t.Fatalf("mask lost through astype: %v", v)
			}
			if out.StorageDType() == pd.Invalid {
				t.Fatal("invalid storage dtype")
			}
		}
	})
}

// FuzzNDArrayTypedArithmetic: promoted results keep shape, valid dtype
// and never mutate inputs.
func FuzzNDArrayTypedArithmetic(f *testing.F) {
	f.Add(1, 2, 3, 1.5, 2.5, 3.5)
	f.Add(0, 0, 0, 0.0, 0.0, 0.0)
	f.Add(-1000, 1000, 7, -1e10, 1e10, 0.1)
	f.Fuzz(func(t *testing.T, a1, a2, a3 int, b1, b2, b3 float64) {
		ints := pd.ArrayInt([]int{a1, a2, a3})
		floats := pd.ArrayFloat64([]float64{b1, b2, b3})
		for _, op := range []func(*pd.NDArray, *pd.NDArray) (*pd.NDArray, error){
			(*pd.NDArray).Add, (*pd.NDArray).Sub, (*pd.NDArray).Mul, (*pd.NDArray).Div,
		} {
			out, err := op(ints, floats)
			if err != nil {
				t.Fatal(err)
			}
			if out.Size() != 3 {
				t.Fatalf("size = %d", out.Size())
			}
			if out.DType() != pd.Float64 {
				t.Fatalf("int op float dtype = %v", out.DType())
			}
		}
		// inputs unchanged
		if got := ints.Values(); got[0] != a1 || got[2] != a3 {
			t.Fatalf("int input mutated: %v", got)
		}
		ii, err := ints.Mul(ints)
		if err != nil {
			t.Fatal(err)
		}
		if ii.DType() != pd.Int {
			t.Fatalf("int*int dtype = %v", ii.DType())
		}
	})
}

// FuzzCSVTypedInference: parsed frames must expose valid typed storage.
func FuzzCSVTypedInference(f *testing.F) {
	f.Add("a,b\n1,x\n2,y\n")
	f.Add("a\n1\n2.5\n")
	f.Add("a\ntrue\nfalse\nNA\n")
	f.Add("a,b,c\n,,\n1,2,3\n")
	f.Fuzz(func(t *testing.T, input string) {
		df, err := pd.ReadCSVReader(strings.NewReader(input))
		if err != nil {
			return
		}
		for name, dt := range df.StorageDTypes() {
			if dt == pd.Invalid {
				t.Fatalf("column %q has invalid storage dtype", name)
			}
		}
		for _, name := range df.Columns() {
			c, err := df.Col(name)
			if err != nil {
				t.Fatal(err)
			}
			if c.Len() != df.Len() {
				t.Fatalf("column %q length %d for %d rows", name, c.Len(), df.Len())
			}
		}
	})
}

// FuzzDataFrameRecordsTypedInference: heterogeneous records must always
// produce consistent columns (typed or object) without panics.
func FuzzDataFrameRecordsTypedInference(f *testing.F) {
	f.Add(int64(1), "x", 2.5, true)
	f.Add(int64(0), "", 0.0, false)
	f.Fuzz(func(t *testing.T, i int64, s string, fl float64, b bool) {
		df, err := pd.DataFrameFromRecords([]map[string]any{
			{"a": i, "b": s},
			{"a": fl, "b": b},
			{"a": nil, "b": nil},
		})
		if err != nil {
			t.Fatal(err)
		}
		if df.Len() != 3 {
			t.Fatalf("rows = %d", df.Len())
		}
		_ = df.String()
		_ = df.StorageDTypes()
		if !df.HasNA() {
			t.Fatal("nil cells should be missing")
		}
	})
}
