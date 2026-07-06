package pandas_test

import (
	"errors"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

// TestRootAPI exercises the re-exported surface exactly as a user would.
func TestRootAPI(t *testing.T) {
	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "name": "Ana", "age": 30, "salary": 1000.0},
		{"country": "AR", "name": "Luis", "age": 40, "salary": 2000.0},
		{"country": "BR", "name": "Joao", "age": 35, "salary": 1500.0},
	}, pd.WithColumnOrder("country", "name", "age", "salary"))
	if err != nil {
		t.Fatal(err)
	}

	filtered, err := df.Where(pd.And(
		pd.Col("age").Gt(30),
		pd.Col("country").IsIn("AR", "BR"),
	))
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Len() != 2 {
		t.Fatalf("filtered len = %d", filtered.Len())
	}

	total, err := df.AssignExpr("total", pd.Col("salary").Mul(12))
	if err != nil {
		t.Fatal(err)
	}
	c, err := total.Col("total")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := c.At(0); v != 12000.0 {
		t.Errorf("total = %v", v)
	}

	s := pd.SeriesOf("age", []int{10, 20, 30})
	if mean, _ := s.Mean(); mean != 20 {
		t.Errorf("series mean = %v", mean)
	}

	a := pd.Array([]float64{1, 2, 3}).AddScalar(10)
	if a.MustAt(2) != 13 {
		t.Errorf("ndarray = %v", a)
	}

	if !pd.IsNA(pd.NA()) || !pd.IsNA(nil) || pd.IsNA(0) {
		t.Error("NA helpers broken")
	}

	if _, err := df.Stack(); !errors.Is(err, pd.ErrNotImplementedBase) {
		t.Errorf("Stack should be ErrNotImplemented, got %v", err)
	}
	if !errors.Is(pd.ErrNotImplemented("X"), pd.ErrNotImplementedBase) {
		t.Error("ErrNotImplemented does not wrap the base error")
	}

	pd.SetDisplayOptions(pd.DisplayOptions{MaxRows: 30})
	if pd.GetDisplayOptions().MaxRows != 30 {
		t.Error("display options round trip")
	}
	pd.SetDisplayOptions(pd.DisplayOptions{MaxRows: 20})
}
