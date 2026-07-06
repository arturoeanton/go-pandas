package pandas_test

import (
	"strings"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

// TestAliases exercises every documented alias at least once.
func TestAliases(t *testing.T) {
	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"a": 1, "b": 2.0},
		{"a": nil, "b": 3.0},
	}, pd.WithColumnOrder("a", "b"))
	if err != nil {
		t.Fatal(err)
	}
	// Column is an alias of Col.
	if _, err := df.Column("a"); err != nil {
		t.Errorf("Column alias: %v", err)
	}
	// IsNull/NotNull are aliases of IsNA/NotNA.
	if !pd.IsNull(nil) || pd.NotNull(nil) || !pd.NotNull(1) {
		t.Error("IsNull/NotNull aliases")
	}
	s := df.MustCol("a")
	if s.IsNull().AsMask()[1] != s.IsNA().AsMask()[1] {
		t.Error("Series IsNull alias")
	}
	// IAt is an alias of At; ILoc too.
	if v, _ := s.IAt(0); v != 1 {
		t.Errorf("IAt = %v", v)
	}
	if v, _ := s.ILoc(0); v != 1 {
		t.Errorf("ILoc = %v", v)
	}
	// ReplaceNA is an alias of FillNA.
	if s.ReplaceNA(0).HasNA() {
		t.Error("ReplaceNA alias")
	}
	// MinPeriods is an alias of RollingMinPeriods.
	if _, err := s.Rolling(2, pd.MinPeriods(1)).Mean(); err != nil {
		t.Errorf("MinPeriods alias: %v", err)
	}
	// IgnoreIndex / Join are concat option aliases.
	if _, err := pd.Concat([]*pd.DataFrame{df, df}, pd.IgnoreIndex(true), pd.Join("outer")); err != nil {
		t.Errorf("concat aliases: %v", err)
	}
	// WithNRows is an alias of WithLimit.
	limited, err := pd.ReadCSVReader(strings.NewReader("a\n1\n2\n"), pd.WithNRows(1))
	if err != nil || limited.Len() != 1 {
		t.Errorf("WithNRows alias: %v, %v", limited, err)
	}
	// AxisRows/AxisColumns constants and Axis helper.
	if pd.AxisRows != 0 || pd.AxisColumns != 1 || pd.Axis(1) != 1 {
		t.Error("axis constants")
	}
}

func TestPhase2RootAPI(t *testing.T) {
	// dtype parsing and SelectDTypes.
	dt, err := pd.ParseDType("datetime64[ns]")
	if err != nil || dt != pd.Time {
		t.Errorf("ParseDType = %v, %v", dt, err)
	}
	// ToDatetime on a string series.
	dates, err := pd.ToDatetime(pd.StringSeries("d", []string{"2024-01-02"}))
	if err != nil {
		t.Fatal(err)
	}
	if y, _ := dates.Dt().Year().At(0); y != 2024 {
		t.Errorf("ToDatetime year = %v", y)
	}
	// Array root ufuncs and binaries.
	arr := pd.Sqrt(pd.Array([]float64{4, 9}))
	if arr.MustAt(1) != 3 {
		t.Errorf("pd.Sqrt = %v", arr)
	}
	sum, err := pd.Add(pd.Array([]float64{1}), pd.Array([]float64{2}))
	if err != nil || sum.MustAt(0) != 3 {
		t.Errorf("pd.Add = %v, %v", sum, err)
	}
	mx, err := pd.Maximum(pd.Array([]float64{1, 9}), pd.Array([]float64{5, 2}))
	if err != nil || mx.MustAt(0) != 5 || mx.MustAt(1) != 9 {
		t.Errorf("pd.Maximum = %v, %v", mx, err)
	}
	// Expression math keeps working under the *Expr names.
	df, _ := pd.DataFrameFromRecords([]map[string]any{{"x": -3.0}})
	out, err := df.AssignExpr("ax", pd.AbsExpr(pd.Col("x")))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := out.MustCol("ax").At(0); v != 3.0 {
		t.Errorf("AbsExpr = %v", v)
	}
	// Typed constructors record dtype.
	if pd.ArrayInt([]int{1}).DType() != pd.Int {
		t.Error("ArrayInt dtype")
	}
	// LabelSlice round-trips through Loc.
	labeled, _ := pd.DataFrameFromMap(map[string][]any{"v": {1, 2, 3}},
		pd.WithDataFrameIndex(pd.NewStringIndex([]string{"a", "b", "c"})))
	sliced, err := labeled.Loc().Rows(pd.LabelSlice("b", "c")).Get()
	if err != nil || sliced.Len() != 2 {
		t.Errorf("LabelSlice = %v, %v", sliced, err)
	}
}
