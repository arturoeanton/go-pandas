// Package compat_test compares go-pandas behavior against golden outputs
// generated from real pandas/NumPy (compat/python/*.py). The golden JSON
// files are committed, so these tests never require Python.
package compat_test

import (
	"math"
	"path/filepath"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
	testutil "github.com/arturoeanton/go-pandas/internal/testing"
)

// caseFn produces the go-pandas result for one golden case. Supported
// return types: *pd.DataFrame, *pd.Series, *pd.NDArray, *pd.BoolArray and
// float64 (scalars).
type caseFn func(t *testing.T) (any, error)

// runSuites executes every golden case in a directory against a registry
// of case implementations.
func runSuites(t *testing.T, dir string, registry map[string]caseFn) {
	files, err := filepath.Glob(filepath.Join("goldens", dir, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no golden files under goldens/%s (err=%v)", dir, err)
	}
	for _, file := range files {
		golden := testutil.LoadGolden(t, file)
		for _, c := range golden.Cases {
			c := c
			t.Run(golden.Suite+"/"+c.Name, func(t *testing.T) {
				fn, ok := registry[c.Name]
				if !ok {
					t.Fatalf("no Go implementation registered for golden case %q", c.Name)
				}
				expected := testutil.ParseExpected(t, c)
				got, err := fn(t)
				if expected.Error {
					if err == nil {
						t.Fatalf("expected an error for %s, got none", c.Operation)
					}
					return
				}
				if err != nil {
					t.Fatalf("%s: %v", c.Operation, err)
				}
				dispatch(t, got, expected)
			})
		}
	}
}

// dispatch compares a case result against its expected payload based on
// which expectation fields are populated.
func dispatch(t *testing.T, got any, expected testutil.GoldenExpected) {
	t.Helper()
	switch {
	case expected.BoolData != nil:
		AssertIsBoolArray(t, got, expected)
	case expected.Scalar != nil:
		f, ok := got.(float64)
		if !ok {
			t.Fatalf("expected a scalar result, got %T", got)
		}
		if !testutil.AllClose(f, *expected.Scalar) {
			t.Fatalf("scalar = %v, want %v", f, *expected.Scalar)
		}
	case expected.Columns != nil:
		frame, ok := got.(*pd.DataFrame)
		if !ok {
			t.Fatalf("expected a DataFrame result, got %T", got)
		}
		testutil.AssertFrameEqual(t, frame, expected)
	case expected.Values != nil:
		s, ok := got.(*pd.Series)
		if !ok {
			t.Fatalf("expected a Series result, got %T", got)
		}
		testutil.AssertSeriesEqual(t, s, expected)
	case expected.Data != nil:
		arr, ok := got.(*pd.NDArray)
		if !ok {
			t.Fatalf("expected an NDArray result, got %T", got)
		}
		testutil.AssertArrayAllClose(t, arr, expected)
	case expected.Shape != nil:
		// Property-only check (random arrays): shape plus optional range.
		arr, ok := got.(*pd.NDArray)
		if !ok {
			t.Fatalf("expected an NDArray result, got %T", got)
		}
		shape := arr.Shape()
		if len(shape) != len(expected.Shape) {
			t.Fatalf("shape = %v, want %v", shape, expected.Shape)
		}
		for i := range shape {
			if shape[i] != expected.Shape[i] {
				t.Fatalf("shape = %v, want %v", shape, expected.Shape)
			}
		}
		for _, v := range arr.Data() {
			if expected.Min != nil && v < *expected.Min {
				t.Fatalf("value %v below %v", v, *expected.Min)
			}
			if expected.Max != nil && v >= *expected.Max {
				t.Fatalf("value %v not below %v", v, *expected.Max)
			}
		}
	default:
		t.Fatal("golden case has no recognizable expectation")
	}
}

// AssertIsBoolArray narrows and compares a boolean mask result.
func AssertIsBoolArray(t *testing.T, got any, expected testutil.GoldenExpected) {
	t.Helper()
	mask, ok := got.(*pd.BoolArray)
	if !ok {
		t.Fatalf("expected a BoolArray result, got %T", got)
	}
	testutil.AssertBoolArrayEqual(t, mask, expected)
}

// Shared pandas fixtures — keep in sync with generate_pandas_goldens.py ----

func peopleFrame(t *testing.T) *pd.DataFrame {
	t.Helper()
	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "name": "Ana", "age": 30, "salary": 1000.0, "dept": "eng"},
		{"country": "AR", "name": "Luis", "age": 40, "salary": 2000.0, "dept": "sales"},
		{"country": "BR", "name": "Joao", "age": 35, "salary": 1500.0, "dept": "eng"},
		{"country": "BR", "name": "Bia", "age": 28, "salary": 1200.0, "dept": "eng"},
		{"country": "AR", "name": "Mia", "age": 22, "salary": 800.0, "dept": "sales"},
	}, pd.WithColumnOrder("country", "name", "age", "salary", "dept"))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

func missingFrame(t *testing.T) *pd.DataFrame {
	t.Helper()
	df, err := pd.DataFrameFromMap(map[string][]any{
		"a": {1, nil, 3, nil},
		"b": {"x", "y", nil, nil},
		"c": {1.5, 2.5, 3.5, nil},
	}, pd.WithColumnOrder("a", "b", "c"))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

func mergeFrames(t *testing.T) (*pd.DataFrame, *pd.DataFrame) {
	t.Helper()
	left, err := pd.DataFrameFromRecords([]map[string]any{
		{"id": 1, "name": "Ana"},
		{"id": 2, "name": "Luis"},
		{"id": 3, "name": "Marta"},
	}, pd.WithColumnOrder("id", "name"))
	if err != nil {
		t.Fatal(err)
	}
	right, err := pd.DataFrameFromRecords([]map[string]any{
		{"id": 1, "salary": 1000.0},
		{"id": 2, "salary": 2000.0},
		{"id": 4, "salary": 4000.0},
	}, pd.WithColumnOrder("id", "salary"))
	if err != nil {
		t.Fatal(err)
	}
	return left, right
}

func gradesFrame(t *testing.T) *pd.DataFrame {
	t.Helper()
	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"name": "Ana", "math": 9.0, "bio": 8.0},
		{"name": "Luis", "math": 7.0, "bio": 6.0},
	}, pd.WithColumnOrder("name", "math", "bio"))
	if err != nil {
		t.Fatal(err)
	}
	return df
}

func numSeries() *pd.Series {
	return pd.NewSeries("v", []any{3.0, 1.0, 4.0, nil, 5.0})
}

func intSeries() *pd.Series {
	return pd.SeriesOf("v", []int{3, 1, 4, 1, 5})
}

func strSeries() *pd.Series {
	return pd.NewSeries("s", []any{"Hello", "world", " Go ", "Anaconda", nil})
}

func rollingSeries() *pd.Series {
	return pd.FloatSeries("v", []float64{1, 2, 3, 4, 5})
}

// Shared NumPy fixtures — keep in sync with generate_numpy_goldens.py -------

func mArr(t *testing.T) *pd.NDArray {
	t.Helper()
	m, err := pd.Arange(6).Reshape(2, 3)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func vArr() *pd.NDArray { return pd.Array([]float64{1, 2, 3}) }
func wArr() *pd.NDArray { return pd.Array([]float64{4, 5, 6}) }

func sqArr(t *testing.T) *pd.NDArray {
	t.Helper()
	a, err := pd.Array2D([][]float64{{1, 2}, {3, 4}})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func negArr() *pd.NDArray { return pd.Array([]float64{-1.5, 2.5, -3.5}) }

func withNaNArr() *pd.NDArray {
	return pd.Array([]float64{1, math.NaN(), 3, math.Inf(1)})
}
