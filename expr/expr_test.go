package expr

import (
	"testing"
)

func row(m map[string]any) map[string]any { return m }

func TestComparisons(t *testing.T) {
	r := row(map[string]any{"age": 30, "name": "Ana", "salary": 1000.0})
	cases := []struct {
		pred Predicate
		want bool
	}{
		{Col("age").Gt(20), true},
		{Col("age").Gt(30), false},
		{Col("age").Ge(30), true},
		{Col("age").Lt(40), true},
		{Col("age").Le(29), false},
		{Col("age").Eq(30), true},
		{Col("age").Ne(30), false},
		{Col("name").Eq("Ana"), true},
		{Col("salary").Gt(999), true},
		{Col("age").Between(30, 40), true},
		{Col("age").IsIn(10, 30), true},
		{Col("age").IsIn(10, 20), false},
		{Col("name").Contains("na"), true},
		{Col("name").StartsWith("A"), true},
		{Col("name").EndsWith("z"), false},
		{Col("age").NotNA(), true},
		{Col("age").IsNA(), false},
	}
	for i, tc := range cases {
		got, err := tc.pred.EvalBool(r)
		if err != nil {
			t.Fatalf("case %d (%s): %v", i, tc.pred, err)
		}
		if got != tc.want {
			t.Errorf("case %d (%s) = %v, want %v", i, tc.pred, got, tc.want)
		}
	}
}

func TestLogical(t *testing.T) {
	r := row(map[string]any{"a": 1, "b": 2})
	if ok, _ := And(Col("a").Eq(1), Col("b").Eq(2)).EvalBool(r); !ok {
		t.Error("And = false")
	}
	if ok, _ := Or(Col("a").Eq(9), Col("b").Eq(2)).EvalBool(r); !ok {
		t.Error("Or = false")
	}
	if ok, _ := Not(Col("a").Eq(1)).EvalBool(r); ok {
		t.Error("Not = true")
	}
}

func TestArithmetic(t *testing.T) {
	r := row(map[string]any{"price": 10.0, "qty": 3, "name": "x"})
	v, err := Col("price").Mul(Col("qty")).Eval(r)
	if err != nil {
		t.Fatal(err)
	}
	if v != 30.0 {
		t.Errorf("price*qty = %v", v)
	}
	v, _ = Col("qty").Add(1).Eval(r)
	if v != int64(4) {
		t.Errorf("qty+1 = %v (%T)", v, v)
	}
	v, _ = Col("qty").Div(2).Eval(r)
	if v != 1.5 {
		t.Errorf("qty/2 = %v", v)
	}
	v, _ = Col("qty").Pow(2).Eval(r)
	if v != 9.0 {
		t.Errorf("qty**2 = %v", v)
	}
	// NA propagation
	rna := row(map[string]any{"price": nil, "qty": 3})
	v, err = Col("price").Mul(Col("qty")).Eval(rna)
	if err != nil || v != nil {
		t.Errorf("NA*3 = %v, %v", v, err)
	}
	// string concat
	v, _ = Col("name").Add("!").Eval(r)
	if v != "x!" {
		t.Errorf("string concat = %v", v)
	}
}

func TestFunctions(t *testing.T) {
	r := row(map[string]any{"x": -4.0, "s": "Go"})
	if v, _ := Abs(Col("x")).Eval(r); v != 4.0 {
		t.Errorf("Abs = %v", v)
	}
	if v, _ := Sqrt(Abs(Col("x"))).Eval(r); v != 2.0 {
		t.Errorf("Sqrt = %v", v)
	}
	if v, _ := Lower(Col("s")).Eval(r); v != "go" {
		t.Errorf("Lower = %v", v)
	}
	if v, _ := Upper(Col("s")).Eval(r); v != "GO" {
		t.Errorf("Upper = %v", v)
	}
	if v, _ := Len(Col("s")).Eval(r); v != 2 {
		t.Errorf("Len = %v", v)
	}
}

func TestWhereExpr(t *testing.T) {
	e := Where(Col("age").Ge(18), "adult", "minor")
	if v, _ := e.Eval(row(map[string]any{"age": 20})); v != "adult" {
		t.Errorf("Where = %v", v)
	}
	if v, _ := e.Eval(row(map[string]any{"age": 10})); v != "minor" {
		t.Errorf("Where = %v", v)
	}
}

func TestMissingColumn(t *testing.T) {
	if _, err := Col("nope").Eq(1).EvalBool(row(map[string]any{"a": 1})); err == nil {
		t.Error("missing column should error")
	}
}

func TestParseQuery(t *testing.T) {
	r := row(map[string]any{"age": 35, "salary": 1500.0, "name": "Ana", "country": "AR"})
	cases := []struct {
		q    string
		want bool
	}{
		{`age > 30`, true},
		{`age >= 36`, false},
		{`age > 30 and salary < 2000`, true},
		{`age > 30 and salary > 2000`, false},
		{`age > 40 or salary > 1000`, true},
		{`name == "Ana"`, true},
		{`name != "Ana"`, false},
		{`country in ["AR", "BR"]`, true},
		{`country in ["US"]`, false},
		{`not (age < 18)`, true},
		{`age > 30 and (country == "BR" or country == "AR")`, true},
	}
	for _, tc := range cases {
		pred, err := ParseQuery(tc.q)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.q, err)
		}
		got, err := pred.EvalBool(r)
		if err != nil {
			t.Fatalf("eval %q: %v", tc.q, err)
		}
		if got != tc.want {
			t.Errorf("%q = %v, want %v", tc.q, got, tc.want)
		}
	}
	if _, err := ParseQuery("age >"); err == nil {
		t.Error("incomplete query should fail")
	}
	if _, err := ParseQuery("age ~ 3"); err == nil {
		t.Error("bad operator should fail")
	}
}
