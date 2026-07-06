// Package compat_test compares go-pandas behavior against golden outputs
// generated from real pandas/NumPy (see python/ for the generators). The
// golden JSON files are committed, so these tests do not require Python.
package compat_test

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

// FloatTolerance is the maximum absolute difference accepted between
// go-pandas and pandas/NumPy floating point results.
const FloatTolerance = 1e-9

type goldenFile struct {
	Cases []goldenCase `json:"cases"`
}

type goldenCase struct {
	Name     string          `json:"name"`
	Expected json.RawMessage `json:"expected"`
}

type arrayExpected struct {
	Shape []int     `json:"shape"`
	Data  []float64 `json:"data"`
	Error bool      `json:"error"`
}

type frameExpected struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

func loadGoldens(t *testing.T, name string) []goldenCase {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("goldens", name))
	if err != nil {
		t.Fatalf("reading golden %s: %v", name, err)
	}
	var f goldenFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parsing golden %s: %v", name, err)
	}
	return f.Cases
}

// --- array helpers -------------------------------------------------------

func checkArray(t *testing.T, got *pd.NDArray, gotErr error, raw json.RawMessage) {
	t.Helper()
	var want arrayExpected
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if want.Error {
		if gotErr == nil {
			t.Fatal("expected an error, got none")
		}
		return
	}
	if gotErr != nil {
		t.Fatalf("unexpected error: %v", gotErr)
	}
	shape := got.Shape()
	if len(shape) != len(want.Shape) {
		t.Fatalf("shape %v, want %v", shape, want.Shape)
	}
	for i := range shape {
		if shape[i] != want.Shape[i] {
			t.Fatalf("shape %v, want %v", shape, want.Shape)
		}
	}
	data := got.Data()
	if len(data) != len(want.Data) {
		t.Fatalf("data length %d, want %d", len(data), len(want.Data))
	}
	for i := range data {
		if math.Abs(data[i]-want.Data[i]) > FloatTolerance {
			t.Fatalf("data[%d] = %v, want %v", i, data[i], want.Data[i])
		}
	}
}

// --- frame helpers -------------------------------------------------------

func checkFrame(t *testing.T, got *pd.DataFrame, gotErr error, raw json.RawMessage) {
	t.Helper()
	if gotErr != nil {
		t.Fatalf("unexpected error: %v", gotErr)
	}
	var want frameExpected
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	cols := got.Columns()
	if len(cols) != len(want.Columns) {
		t.Fatalf("columns %v, want %v", cols, want.Columns)
	}
	for i := range cols {
		if cols[i] != want.Columns[i] {
			t.Fatalf("columns %v, want %v", cols, want.Columns)
		}
	}
	rows := got.ToRows()
	if len(rows) != len(want.Rows) {
		t.Fatalf("row count %d, want %d", len(rows), len(want.Rows))
	}
	for i := range rows {
		for j := range want.Rows[i] {
			if !cellEqual(rows[i][j], want.Rows[i][j]) {
				t.Fatalf("cell [%d][%d] = %v (%T), want %v", i, j, rows[i][j], rows[i][j], want.Rows[i][j])
			}
		}
	}
}

// cellEqual compares a go-pandas cell with a JSON-decoded expected value.
func cellEqual(got, want any) bool {
	if want == nil || got == nil {
		return want == nil && got == nil
	}
	if wf, ok := want.(float64); ok {
		switch g := got.(type) {
		case float64:
			return math.Abs(g-wf) <= FloatTolerance
		case int:
			return math.Abs(float64(g)-wf) <= FloatTolerance
		case int64:
			return math.Abs(float64(g)-wf) <= FloatTolerance
		}
		return false
	}
	if wb, ok := want.(bool); ok {
		gb, ok := got.(bool)
		return ok && gb == wb
	}
	if ws, ok := want.(string); ok {
		gs, ok := got.(string)
		return ok && gs == ws
	}
	return got == want
}

// --- shared fixtures ------------------------------------------------------

func peopleFrame(t *testing.T) *pd.DataFrame {
	t.Helper()
	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "name": "Ana", "age": 30, "salary": 1000.0},
		{"country": "AR", "name": "Luis", "age": 40, "salary": 2000.0},
		{"country": "BR", "name": "Joao", "age": 35, "salary": 1500.0},
	}, pd.WithColumnOrder("country", "name", "age", "salary"))
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

// --- NumPy goldens ---------------------------------------------------------

func TestNumpyGoldens(t *testing.T) {
	arrayCases := map[string]func() (*pd.NDArray, error){
		"array_creation": func() (*pd.NDArray, error) { return pd.Array([]float64{1, 2, 3}), nil },
		"zeros_2x3":      func() (*pd.NDArray, error) { return pd.Zeros(2, 3), nil },
		"ones_3":         func() (*pd.NDArray, error) { return pd.Ones(3), nil },
		"arange_5":       func() (*pd.NDArray, error) { return pd.Arange(5), nil },
		"arange_2_10_2":  func() (*pd.NDArray, error) { return pd.Arange(2, 10, 2), nil },
		"linspace_0_1_5": func() (*pd.NDArray, error) { return pd.Linspace(0, 1, 5), nil },
		"reshape_2x3":    func() (*pd.NDArray, error) { return pd.Arange(6).Reshape(2, 3) },
		"transpose_2x3": func() (*pd.NDArray, error) {
			a, err := pd.Arange(6).Reshape(2, 3)
			if err != nil {
				return nil, err
			}
			return a.T()
		},
		"sum_all":  func() (*pd.NDArray, error) { return pd.Arange(6).Sum() },
		"mean_all": func() (*pd.NDArray, error) { return pd.Arange(6).Mean() },
		"std_all":  func() (*pd.NDArray, error) { return pd.Arange(6).Std() },
		"sqrt":     func() (*pd.NDArray, error) { return pd.Array([]float64{1, 4, 9}).Sqrt(), nil },
		"exp":      func() (*pd.NDArray, error) { return pd.Array([]float64{0, 1}).Exp(), nil },
		"log":      func() (*pd.NDArray, error) { return pd.Array([]float64{1, math.E}).Log(), nil },
		"broadcast_scalar": func() (*pd.NDArray, error) {
			return pd.Array([]float64{1, 2, 3}).AddScalar(10), nil
		},
		"broadcast_vector_to_matrix": func() (*pd.NDArray, error) {
			a, err := pd.FromSlice([]float64{1, 2, 3, 4, 5, 6}, 2, 3)
			if err != nil {
				return nil, err
			}
			return a.Add(pd.Array([]float64{10, 20, 30}))
		},
		"broadcast_col_row": func() (*pd.NDArray, error) {
			col, err := pd.FromSlice([]float64{1, 2}, 2, 1)
			if err != nil {
				return nil, err
			}
			row, err := pd.FromSlice([]float64{10, 20, 30}, 1, 3)
			if err != nil {
				return nil, err
			}
			return col.Add(row)
		},
		"broadcast_incompatible": func() (*pd.NDArray, error) {
			return pd.Array([]float64{1, 2, 3}).Add(pd.Zeros(4))
		},
		"dot_vectors": func() (*pd.NDArray, error) {
			return pd.Dot(pd.Array([]float64{1, 2, 3}), pd.Array([]float64{4, 5, 6}))
		},
		"matvec": func() (*pd.NDArray, error) {
			m, err := pd.FromSlice([]float64{1, 2, 3, 4}, 2, 2)
			if err != nil {
				return nil, err
			}
			return m.Dot(pd.Array([]float64{5, 6}))
		},
		"matmul_2x2": func() (*pd.NDArray, error) {
			a, err := pd.FromSlice([]float64{1, 2, 3, 4}, 2, 2)
			if err != nil {
				return nil, err
			}
			b, err := pd.FromSlice([]float64{5, 6, 7, 8}, 2, 2)
			if err != nil {
				return nil, err
			}
			return pd.MatMul(a, b)
		},
	}
	for _, file := range []string{"numpy_basic.json", "numpy_broadcast.json", "numpy_linalg.json"} {
		for _, c := range loadGoldens(t, file) {
			c := c
			t.Run(c.Name, func(t *testing.T) {
				fn, ok := arrayCases[c.Name]
				if !ok {
					t.Fatalf("no implementation for golden case %q", c.Name)
				}
				got, err := fn()
				checkArray(t, got, err, c.Expected)
			})
		}
	}
}

// --- pandas goldens ----------------------------------------------------------

func TestPandasGoldens(t *testing.T) {
	frameCases := map[string]func(t *testing.T) (*pd.DataFrame, error){
		"head_2": func(t *testing.T) (*pd.DataFrame, error) { return peopleFrame(t).Head(2), nil },
		"tail_1": func(t *testing.T) (*pd.DataFrame, error) { return peopleFrame(t).Tail(1), nil },
		"select_name_salary": func(t *testing.T) (*pd.DataFrame, error) {
			return peopleFrame(t).Select("name", "salary")
		},
		"filter_age_gt_30": func(t *testing.T) (*pd.DataFrame, error) {
			return peopleFrame(t).Where(pd.Col("age").Gt(30))
		},
		"sort_salary_desc": func(t *testing.T) (*pd.DataFrame, error) {
			return peopleFrame(t).SortValues("salary", false)
		},
		"assign_bonus": func(t *testing.T) (*pd.DataFrame, error) {
			df, err := peopleFrame(t).AssignExpr("bonus", pd.Col("salary").Mul(0.1))
			if err != nil {
				return nil, err
			}
			return df.Select("name", "bonus")
		},
		"melt": func(t *testing.T) (*pd.DataFrame, error) {
			df, err := pd.DataFrameFromRecords([]map[string]any{
				{"name": "Ana", "math": 9.0, "bio": 8.0},
				{"name": "Luis", "math": 7.0, "bio": 6.0},
			}, pd.WithColumnOrder("name", "math", "bio"))
			if err != nil {
				return nil, err
			}
			return df.Melt(pd.MeltOptions{IDVars: []string{"name"}})
		},
		"rolling_mean_3": func(t *testing.T) (*pd.DataFrame, error) {
			df, err := pd.DataFrameFromMap(map[string][]any{"v": {1.0, 2.0, 3.0, 4.0, 5.0}})
			if err != nil {
				return nil, err
			}
			return df.Rolling(3).Mean()
		},
		"isna": func(t *testing.T) (*pd.DataFrame, error) {
			df, err := missingFrame()
			if err != nil {
				return nil, err
			}
			return df.IsNA(), nil
		},
		"dropna": func(t *testing.T) (*pd.DataFrame, error) {
			df, err := missingFrame()
			if err != nil {
				return nil, err
			}
			return df.DropNA(), nil
		},
		"fillna": func(t *testing.T) (*pd.DataFrame, error) {
			df, err := missingFrame()
			if err != nil {
				return nil, err
			}
			return df.FillNA(map[string]any{"a": 0, "b": "?"})
		},
		"groupby_sum": func(t *testing.T) (*pd.DataFrame, error) {
			return peopleFrame(t).GroupBy("country").Sum("salary")
		},
		"groupby_mean": func(t *testing.T) (*pd.DataFrame, error) {
			return peopleFrame(t).GroupBy("country").Mean("salary")
		},
		"groupby_count": func(t *testing.T) (*pd.DataFrame, error) {
			return peopleFrame(t).GroupBy("country").Count("name")
		},
		"groupby_agg": func(t *testing.T) (*pd.DataFrame, error) {
			return peopleFrame(t).GroupBy("country").Agg(map[string]string{
				"salary": "mean",
				"age":    "max",
			})
		},
		"merge_inner": func(t *testing.T) (*pd.DataFrame, error) {
			l, r := mergeFrames(t)
			return l.Merge(r, pd.MergeOptions{On: []string{"id"}, How: "inner"})
		},
		"merge_left": func(t *testing.T) (*pd.DataFrame, error) {
			l, r := mergeFrames(t)
			return l.Merge(r, pd.MergeOptions{On: []string{"id"}, How: "left"})
		},
		"merge_outer": func(t *testing.T) (*pd.DataFrame, error) {
			l, r := mergeFrames(t)
			return l.Merge(r, pd.MergeOptions{On: []string{"id"}, How: "outer"})
		},
		"concat": func(t *testing.T) (*pd.DataFrame, error) {
			a, err := pd.DataFrameFromRecords([]map[string]any{
				{"x": 1, "y": "a"}, {"x": 2, "y": "b"},
			}, pd.WithColumnOrder("x", "y"))
			if err != nil {
				return nil, err
			}
			b, err := pd.DataFrameFromRecords([]map[string]any{
				{"x": 3, "y": "c"},
			}, pd.WithColumnOrder("x", "y"))
			if err != nil {
				return nil, err
			}
			return pd.Concat([]*pd.DataFrame{a, b}, pd.ConcatIgnoreIndex(true))
		},
	}
	for _, file := range []string{"pandas_basic.json", "pandas_missing.json", "pandas_groupby.json", "pandas_merge.json"} {
		for _, c := range loadGoldens(t, file) {
			c := c
			t.Run(c.Name, func(t *testing.T) {
				fn, ok := frameCases[c.Name]
				if !ok {
					t.Fatalf("no implementation for golden case %q", c.Name)
				}
				got, err := fn(t)
				checkFrame(t, got, err, c.Expected)
			})
		}
	}
}

func missingFrame() (*pd.DataFrame, error) {
	return pd.DataFrameFromRecords([]map[string]any{
		{"a": 1, "b": "x"},
		{"a": nil, "b": "y"},
		{"a": 3, "b": nil},
	}, pd.WithColumnOrder("a", "b"))
}
