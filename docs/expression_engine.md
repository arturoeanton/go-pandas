# The columnar expression engine (v0.4)

## Why it exists

Before v0.4, `df.Where`, `df.AssignExpr` and `df.Query` evaluated
expressions one row at a time over a freshly built `map[string]any` per
row — boxing every cell and allocating a map even when the columns were
fully typed. The columnar engine evaluates the whole expression over the
typed column buffers from v0.3, then gathers rows once.

The row-map evaluator is still there: it is the documented fallback, and
custom `Expr`/`Predicate` implementations keep working through it.

## Which expressions run columnar

An expression takes the fast path when every operand resolves to typed
storage:

- Comparisons `Eq/Ne/Gt/Ge/Lt/Le` — numeric (int/int64/float32/float64,
  bool as 0/1), string (lexicographic) and `time.Time` columns, against
  scalars or other columns of the same kind.
- `IsNA` / `NotNA` (mask reads), `IsIn` (numeric or string sets),
  `Contains` / `StartsWith` / `EndsWith` (string columns).
- `And` / `Or` / `Not` over columnar predicates.
- Arithmetic `Add/Sub/Mul/Div/Mod/Pow` — column↔column and
  column↔scalar; string `Add` concatenates.
- `AbsExpr/SqrtExpr/LogExpr/ExpExpr`, `Lower/Upper/Len`.
- `Where(cond, x, y)` selection.

Check the path of any expression:

```go
fmt.Println(pd.DebugPlan(df, pd.Col("age").Gt(30)))
// columnar: (col(age) > 30)
```

`df.Plan(e)` returns the structured `*expr.Plan` for tests.

## Which expressions fall back to row-map

- Any operand over an **object-backed** column (mixed values).
- Comparisons between operands of **different kinds** (numeric column vs
  string literal, ...) — the row evaluator's per-value semantics apply.
- Custom `Expr`/`Predicate` implementations from outside the package.
- Scalar-only expressions.

Fallback is behavior-identical by construction: the engine's tests run
the same predicates through both paths and require equal frames.

## NA behavior in predicates

The columnar mask is three-valued: each row is true, false or NA
(missing operand). Combination follows Kleene logic — `false AND NA` is
false, `true OR NA` is true, everything else involving NA stays NA;
`NOT NA` is NA.

- **Filtering** (`Where`, `Query`, boolean masks): NA rows are **not
  selected**, matching pandas boolean indexing and the documented
  "comparisons with NA are false" rule.
- **Assigning a predicate** (`df.AssignExpr("flag", ...)`): NA results
  become `false` in the Bool column, matching the row evaluator and
  pandas' classic (non-nullable) bool arrays.
- Arithmetic propagates NA into the result column's mask.

## DType preservation

Results stay typed:

- int (op) int → Int64 column (Div → Float64, like everywhere else).
- anything float → Float64 column.
- string concat → String column; `Len` → Int column.
- predicates → Bool column.
- `AssignExpr("x", pd.Col("y"))` copies the column with its dtype.

## Performance

100K rows, Apple M4 (see docs/performance.md for the full table):

```text
Where numeric        4.3 ms columnar vs 17.3 ms row-map   (~4x)
AssignExpr numeric   1.3 ms columnar vs 16.4 ms row-map   (~13x, 63 vs 386K allocs)
String contains      0.62 ms
Query (and)          3.2 ms
```

## Known limitations

- The engine plans by attempting evaluation; there is no separate static
  cost model.
- `Where`'s remaining allocations come from row gathering (index label
  boxing in `Take`) — a typed index gather is future work.
- Mixed-kind comparisons always fall back rather than erroring early.
- `EvalContext` resolves columns through a closure instead of holding a
  `*DataFrame` (avoids an import cycle); the columnar interfaces live in
  `expr` and `internal/column` and are not part of the stable public API.
