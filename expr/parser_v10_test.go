package expr

import (
	"testing"
	"time"
)

func evalRow(t *testing.T, q string, row map[string]any) bool {
	t.Helper()
	pred, err := ParseQuery(q)
	if err != nil {
		t.Fatalf("parse %q: %v", q, err)
	}
	got, err := pred.EvalBool(row)
	if err != nil {
		t.Fatalf("eval %q: %v", q, err)
	}
	return got
}

func TestQueryArithmetic(t *testing.T) {
	row := map[string]any{"salary": 900.0, "bonus": 200.0, "n": 7}
	cases := map[string]bool{
		"salary + bonus > 1000":       true,
		"salary + bonus > 1200":       false,
		"salary - bonus >= 700":       true,
		"salary * 2 > 1799":           true,
		"salary / 3 < 301":            true,
		"n % 2 == 1":                  true,
		"(salary + bonus) * 2 > 2199": true,
		"-salary < 0":                 true,
		"salary + -100 == 800":        true,
		"2 + 3 == 5":                  true,
	}
	for q, want := range cases {
		if got := evalRow(t, q, row); got != want {
			t.Errorf("%q = %v, want %v", q, got, want)
		}
	}
}

func TestQueryInNotIn(t *testing.T) {
	row := map[string]any{"country": "AR", "n": 2}
	cases := map[string]bool{
		`country in ["AR", "BR"]`:     true,
		`country in ["CL"]`:           false,
		`country not in ["CL", "UY"]`: true,
		`country not in ["AR"]`:       false,
		`n in [1, 2, 3]`:              true,
		`n not in [1, 3]`:             true,
	}
	for q, want := range cases {
		if got := evalRow(t, q, row); got != want {
			t.Errorf("%q = %v, want %v", q, got, want)
		}
	}
}

func TestQueryBoolAndParens(t *testing.T) {
	row := map[string]any{"active": true, "admin": false, "age": 40, "salary": 1500.0}
	cases := map[string]bool{
		"active":                                    true,
		"not admin":                                 true,
		"active == true":                            true,
		"admin == false":                            true,
		"(age > 30 and salary > 1000) or admin":     true,
		"not (age > 30 and salary > 1000)":          false,
		"(age > 50 or admin) and active":            false,
		"(salary > 1000) and (age > 30) and active": true,
	}
	for q, want := range cases {
		if got := evalRow(t, q, row); got != want {
			t.Errorf("%q = %v, want %v", q, got, want)
		}
	}
}

func TestQueryDatetimeComparison(t *testing.T) {
	row := map[string]any{"date": time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)}
	cases := map[string]bool{
		`date >= "2026-01-01"`:         true,
		`date < "2026-01-01"`:          false,
		`date == "2026-01-15"`:         true,
		`date > "2026-01-15 00:00:01"`: false,
		`date <= "2026-02-01"`:         true,
	}
	for q, want := range cases {
		if got := evalRow(t, q, row); got != want {
			t.Errorf("%q = %v, want %v", q, got, want)
		}
	}
}

func TestQuerySyntaxErrors(t *testing.T) {
	for _, q := range []string{
		"salary >>> 3",
		"salary +",
		"(salary > 1",
		"salary in [1, 2",
		"in [1]",
		"salary + bonus in [1]",
		"salary ~ 3",
		"",
	} {
		if _, err := ParseQuery(q); err == nil {
			t.Errorf("%q must be a syntax error", q)
		}
	}
}
