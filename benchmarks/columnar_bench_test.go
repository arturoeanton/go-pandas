package benchmarks

import (
	"strings"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

// exprFrame builds a 100K-row frame; object=true forces the salary and
// name columns onto object storage so the expression engine falls back
// to the row-map path.
func exprFrame(b *testing.B, object bool) *pd.DataFrame {
	b.Helper()
	n := 100_000
	salary := make([]any, n)
	qty := make([]any, n)
	name := make([]any, n)
	for i := 0; i < n; i++ {
		salary[i] = float64(800 + i%2000)
		qty[i] = i % 7
		name[i] = "user-" + strings.Repeat("x", i%3) + "abc"[i%3:i%3+1]
	}
	df, err := pd.DataFrameFromMap(map[string][]any{
		"salary": salary, "qty": qty, "name": name,
	}, pd.WithColumnOrder("salary", "qty", "name"))
	if err != nil {
		b.Fatal(err)
	}
	if object {
		for _, col := range []string{"salary", "qty", "name"} {
			obj := pd.NewSeries(col, df.MustCol(col).Values(), pd.WithDType(pd.Object))
			df, err = df.Assign(col, obj)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
	return df
}

func requirePlan(b *testing.B, df *pd.DataFrame, e pd.Expr, want string) {
	b.Helper()
	if got := pd.DebugPlan(df, e); !strings.HasPrefix(got, want) {
		b.Fatalf("plan = %s, want %s", got, want)
	}
}

func BenchmarkWhereNumericColumnar100K(b *testing.B) {
	df := exprFrame(b, false)
	pred := pd.Col("salary").Gt(1500)
	requirePlan(b, df, pred, "columnar")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Where(pred); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWhereNumericRowMap100K(b *testing.B) {
	df := exprFrame(b, true)
	pred := pd.Col("salary").Gt(1500)
	requirePlan(b, df, pred, "row-fallback")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Where(pred); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAssignExprNumericColumnar100K(b *testing.B) {
	df := exprFrame(b, false)
	e := pd.Col("salary").Mul(pd.Col("qty"))
	requirePlan(b, df, e, "columnar")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.AssignExpr("total", e); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAssignExprNumericRowMap100K(b *testing.B) {
	df := exprFrame(b, true)
	e := pd.Col("salary").Mul(pd.Col("qty"))
	requirePlan(b, df, e, "row-fallback")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.AssignExpr("total", e); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryNumericColumnar100K(b *testing.B) {
	df := exprFrame(b, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Query("salary > 1500 and qty < 5"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStringContainsColumnar100K(b *testing.B) {
	df := exprFrame(b, false)
	pred := pd.Col("name").Contains("xa")
	requirePlan(b, df, pred, "columnar")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Where(pred); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBooleanAndColumnar100K(b *testing.B) {
	df := exprFrame(b, false)
	pred := pd.And(pd.Col("salary").Gt(1200), pd.Col("qty").Lt(6))
	requirePlan(b, df, pred, "columnar")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Where(pred); err != nil {
			b.Fatal(err)
		}
	}
}
