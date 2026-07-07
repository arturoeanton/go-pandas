package expr

// The row-map evaluator — Expr.Eval(map[string]any) and
// Predicate.EvalBool — is the compatibility fallback of the columnar
// engine. It stays fully supported: the DataFrame methods try
// TryEvalMask/TryEvalColumnar first and, on ErrNotColumnar, build row
// maps exactly as before v0.4.
//
// Expressions that currently fall back:
//
//   - any operand over an object-backed column (mixed values)
//   - comparisons between operands of different kinds (e.g. a numeric
//     column against a string literal)
//   - custom Expr/Predicate implementations outside this package
//   - Between on non-columnar operands, IsIn with non-string/numeric
//     candidates
//
// EvalRow and EvalRowBool are thin named wrappers so callers (and
// benchmarks) can invoke the fallback explicitly.

// EvalRow evaluates an expression against one row map (the fallback
// path).
func EvalRow(e Expr, row map[string]any) (any, error) { return e.Eval(row) }

// EvalRowBool evaluates a predicate against one row map (the fallback
// path).
func EvalRowBool(p Predicate, row map[string]any) (bool, error) { return p.EvalBool(row) }
