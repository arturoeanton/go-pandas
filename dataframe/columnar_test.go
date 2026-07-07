package dataframe

import (
	"strings"
	"testing"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/series"
)

func planKind(t *testing.T, df *DataFrame, e expr.Expr) expr.PlanKind {
	t.Helper()
	return df.Plan(e).Kind
}

func TestPlanDiagnostics(t *testing.T) {
	df := sampleFrame(t)
	// typed numeric predicate -> columnar
	if k := planKind(t, df, expr.Col("age").Gt(30)); k != expr.PlanColumnar {
		t.Errorf("numeric predicate plan = %v", k)
	}
	// string contains -> columnar
	if k := planKind(t, df, expr.Col("name").Contains("a")); k != expr.PlanColumnar {
		t.Errorf("contains plan = %v", k)
	}
	// arithmetic -> columnar
	if k := planKind(t, df, expr.Col("salary").Mul(2)); k != expr.PlanColumnar {
		t.Errorf("arithmetic plan = %v", k)
	}
	// object-backed column -> fallback
	obj := series.NewSeries("obj", []any{1, 2, 3}, series.WithDType(dtype.Object))
	withObj, err := df.Assign("obj", obj)
	if err != nil {
		t.Fatal(err)
	}
	if k := planKind(t, withObj, expr.Col("obj").Gt(1)); k != expr.PlanFallback {
		t.Errorf("object column plan = %v", k)
	}
	// mixed-kind comparison -> fallback
	if k := planKind(t, df, expr.Col("age").Eq("thirty")); k != expr.PlanFallback {
		t.Errorf("mixed kinds plan = %v", k)
	}
	// unknown column -> error
	if k := planKind(t, df, expr.Col("nope").Gt(1)); k != expr.PlanError {
		t.Errorf("unknown column plan = %v", k)
	}
	if !strings.Contains(df.Plan(expr.Col("age").Gt(30)).String(), "columnar") {
		t.Error("plan string should mention the path")
	}
}

// TestColumnarMatchesFallback runs the same predicates through both
// engines and requires identical results.
func TestColumnarMatchesFallback(t *testing.T) {
	df, err := DataFrameFromRecords([]map[string]any{
		{"age": 30, "salary": 1000.0, "name": "Ana", "active": true},
		{"age": nil, "salary": 2000.0, "name": "Luis", "active": false},
		{"age": 35, "salary": nil, "name": nil, "active": true},
		{"age": 28, "salary": 1200.0, "name": "Bia", "active": false},
	}, WithColumnOrder("age", "salary", "name", "active"))
	if err != nil {
		t.Fatal(err)
	}
	preds := []expr.Predicate{
		expr.Col("age").Gt(29),
		expr.Col("age").Le(30),
		expr.Col("age").Ne(35),
		expr.Col("salary").Gt(expr.Col("age")),
		expr.And(expr.Col("age").Gt(20), expr.Col("salary").Lt(1500)),
		expr.Or(expr.Col("age").Gt(34), expr.Col("active").Eq(true)),
		expr.Not(expr.Col("active").Eq(true)),
		expr.Col("name").Contains("a"),
		expr.Col("name").StartsWith("B"),
		expr.Col("name").EndsWith("s"),
		expr.Col("age").IsNA(),
		expr.Col("age").NotNA(),
		expr.Col("age").IsIn(30, 28),
		expr.Col("name").IsIn("Ana", "Bia"),
		expr.Col("age").Between(28, 32),
	}
	for _, p := range preds {
		if k := planKind(t, df, p); k != expr.PlanColumnar {
			t.Errorf("%s plan = %v, want columnar", p, k)
			continue
		}
		fast, err := df.Where(p)
		if err != nil {
			t.Fatalf("%s columnar: %v", p, err)
		}
		slow, err := df.whereRows(p)
		if err != nil {
			t.Fatalf("%s fallback: %v", p, err)
		}
		if fast.Len() != slow.Len() {
			t.Fatalf("%s: columnar %d rows, fallback %d rows", p, fast.Len(), slow.Len())
		}
		fr, sr := fast.ToRows(), slow.ToRows()
		for i := range fr {
			for j := range fr[i] {
				if fr[i][j] != sr[i][j] {
					t.Fatalf("%s: row %d differs: %v vs %v", p, i, fr[i], sr[i])
				}
			}
		}
	}
}

func TestColumnarAssignDTypePreservation(t *testing.T) {
	df, _ := DataFrameFromMap(map[string][]any{
		"a": {1, 2, nil},
		"b": {10, 20, 30},
		"f": {1.5, 2.5, 3.5},
		"s": {"x", "y", "z"},
	}, WithColumnOrder("a", "b", "f", "s"))

	// int * int -> Int64 typed column, NA propagates
	sum, err := df.AssignExpr("t", expr.Col("a").Mul(expr.Col("b")))
	if err != nil {
		t.Fatal(err)
	}
	tc := sum.MustCol("t")
	if tc.StorageDType() != dtype.Int64 || tc.IsObjectBacked() {
		t.Errorf("int*int storage = %v object=%v", tc.StorageDType(), tc.IsObjectBacked())
	}
	if v, _ := tc.At(2); v != nil {
		t.Errorf("NA propagation = %v", v)
	}
	if v, _ := tc.At(1); v != int64(40) {
		t.Errorf("int*int value = %v (%T)", v, v)
	}

	// int / int -> Float64
	div, err := df.AssignExpr("d", expr.Col("b").Div(2))
	if err != nil {
		t.Fatal(err)
	}
	if div.MustCol("d").StorageDType() != dtype.Float64 {
		t.Errorf("div storage = %v", div.MustCol("d").StorageDType())
	}

	// predicate -> Bool column
	flag, err := df.AssignExpr("flag", expr.Col("b").Gt(15))
	if err != nil {
		t.Fatal(err)
	}
	if flag.MustCol("flag").StorageDType() != dtype.Bool {
		t.Errorf("flag storage = %v", flag.MustCol("flag").StorageDType())
	}

	// string concat -> String column
	cat, err := df.AssignExpr("c", expr.Col("s").Add("!"))
	if err != nil {
		t.Fatal(err)
	}
	if cat.MustCol("c").StorageDType() != dtype.String {
		t.Errorf("concat storage = %v", cat.MustCol("c").StorageDType())
	}
	if v, _ := cat.MustCol("c").At(0); v != "x!" {
		t.Errorf("concat = %v", v)
	}

	// bare column copy -> same dtype
	copyCol, err := df.AssignExpr("f2", expr.Col("f"))
	if err != nil {
		t.Fatal(err)
	}
	if copyCol.MustCol("f2").StorageDType() != dtype.Float64 {
		t.Errorf("column copy storage = %v", copyCol.MustCol("f2").StorageDType())
	}

	// Where(cond, x, y) expression
	sel, err := df.AssignExpr("w", expr.Where(expr.Col("b").Gt(15), "hi", "lo"))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := sel.MustCol("w").At(0); v != "lo" {
		t.Errorf("where expr = %v", v)
	}
	if k := planKind(t, df, expr.Where(expr.Col("b").Gt(15), "hi", "lo")); k != expr.PlanColumnar {
		t.Errorf("where expr plan = %v", k)
	}
}

func TestColumnarNAPredicate(t *testing.T) {
	df, _ := DataFrameFromMap(map[string][]any{"v": {1, nil, 3}})
	// NA rows never match, matching pandas boolean indexing
	out, err := df.Where(expr.Col("v").Gt(0))
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 2 {
		t.Errorf("NA row selected: len = %d", out.Len())
	}
	// ... including through Not (Kleene: not NA = NA -> dropped)
	out, err = df.Where(expr.Not(expr.Col("v").Gt(0)))
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("not(NA) selected rows: %d", out.Len())
	}
	// And with a definitive false wins over NA
	out, err = df.Where(expr.And(expr.Col("v").Gt(0), expr.Col("v").Lt(0)))
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("false AND NA selected rows: %d", out.Len())
	}
	// assigning a predicate: NA becomes false (classic bool column)
	flag, err := df.AssignExpr("flag", expr.Col("v").Gt(0))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := flag.MustCol("flag").At(1); v != false {
		t.Errorf("NA predicate assign = %v, want false", v)
	}
}

func TestColumnarTimeCompare(t *testing.T) {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	df, _ := DataFrameFromMap(map[string][]any{
		"d": {t0, t0.AddDate(0, 1, 0), t0.AddDate(0, 2, 0)},
	})
	if k := planKind(t, df, expr.Col("d").Gt(t0)); k != expr.PlanColumnar {
		t.Errorf("time predicate plan = %v", k)
	}
	out, err := df.Where(expr.Col("d").Gt(t0))
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 2 {
		t.Errorf("time filter len = %d", out.Len())
	}
}

func TestColumnarImmutability(t *testing.T) {
	df := sampleFrame(t)
	before := df.ToRows()
	if _, err := df.Where(expr.Col("age").Gt(30)); err != nil {
		t.Fatal(err)
	}
	if _, err := df.AssignExpr("x", expr.Col("salary").Mul(2)); err != nil {
		t.Fatal(err)
	}
	if _, err := df.Query("age > 30"); err != nil {
		t.Fatal(err)
	}
	after := df.ToRows()
	for i := range before {
		for j := range before[i] {
			if before[i][j] != after[i][j] {
				t.Fatalf("source frame mutated at [%d][%d]", i, j)
			}
		}
	}
	if len(df.Columns()) != 4 {
		t.Fatalf("source columns changed: %v", df.Columns())
	}
	// The assigned column must not alias frame storage.
	assigned, _ := df.AssignExpr("copy", expr.Col("age"))
	_ = assigned.MustCol("copy").Set(0, 999)
	if v, _ := df.MustCol("age").At(0); v == 999 {
		t.Fatal("assigned column aliases the source column")
	}
}

func TestQueryColumnar(t *testing.T) {
	df := sampleFrame(t)
	out, err := df.Query("age > 30 and salary < 1600")
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 1 {
		t.Errorf("query len = %d", out.Len())
	}
	if v := colValues(t, out, "name"); v[0] != "Joao" {
		t.Errorf("query = %v", v)
	}
	str, err := df.Query(`name.str.contains("a")`)
	if err != nil {
		t.Fatal(err)
	}
	if str.Len() != 2 { // Ana, Joao (lowercase a)
		t.Errorf("query contains len = %d", str.Len())
	}
}
