<p align="center">
  <img src="logo.png" alt="go-pandas" width="240"/>
</p>

# go-pandas

**Pandas and NumPy style data analysis for Go.**

> If you know pandas and NumPy, the concepts should transfer immediately.

## What is it?

go-pandas is a compatibility-oriented data toolkit: pandas-style
`DataFrame`/`Series` and NumPy-style `NDArray` with the same concepts,
names and behavior — verified against **golden outputs generated from
real pandas and NumPy** (200+ test cases, pandas 2.3 / NumPy 2.0).

## Stability status

go-pandas v0.2.x is **experimental**. The API is not yet v1 stable.
Compatibility is conceptual and behavioral where tested, not Python
syntax compatibility. Current coverage, computed from the matrices with
`go run ./cmd/compat-report`: pandas 93% of 98 tracked rows, NumPy 88%
of 52 tracked rows — including partial rows
([full report](compat/coverage_report.md), [what's intentionally
different](compat/known_differences.md)).

## Installation

```bash
go get github.com/arturoeanton/go-pandas
```

```go
import pd "github.com/arturoeanton/go-pandas"
```

Zero dependencies outside the standard library.

## Quick pandas translation

| pandas | go-pandas |
|---|---|
| `pd.DataFrame(records)` | `pd.DataFrameFromRecords(records)` |
| `df["age"]` | `df.Col("age")` |
| `df[["name", "age"]]` | `df.Select("name", "age")` |
| `df[df["age"] > 30]` | `df.Where(pd.Col("age").Gt(30))` |
| `df.query("a > 1 and s.str.contains('x')")` | `df.Query(...)` — same string |
| `df["total"] = df.price * df.qty` | `df.AssignExpr("total", pd.Col("price").Mul(pd.Col("qty")))` |
| `df.groupby("c")["v"].mean()` | `df.GroupBy("c").Mean("v")` |
| `pd.merge(l, r, on="id", how="left")` | `pd.Merge(l, r, pd.MergeOptions{On: []string{"id"}, How: "left"})` |
| `df.dropna(thresh=2)` | `df.DropNA(pd.DropNAThresh(2))` |
| `s.rank(method="dense")` | `s.Rank(pd.RankMethod("dense"))` |
| `s.dt.quarter` / `s.str.match(p)` | `s.Dt().Quarter()` / `s.Str().Match(p)` |

Full guide: [docs/pandas_translation_guide.md](docs/pandas_translation_guide.md)

## Quick NumPy translation

| NumPy | go-pandas |
|---|---|
| `np.arange(6).reshape(2, 3)` | `pd.Arange(6).Reshape(2, 3)` |
| `a + b` (broadcasting) | `a.Add(b)` |
| `a[0:2, 1:3]` | `a.Slice(pd.Slice(0, 2), pd.Slice(1, 3))` |
| `a[a > 0]` | `a.Mask(a.GtScalar(0))` |
| `a.sum(axis=0)` | `a.Sum(pd.Axis(0))` |
| `a.std(ddof=1)` | `a.StdDDof(1)` |
| `np.sqrt(a)` / `np.clip(a, 0, 1)` | `pd.Sqrt(a)` / `pd.Clip(a, 0, 1)` |
| `np.concatenate([a, b], 0)` | `pd.Concatenate([]*pd.NDArray{a, b}, 0)` |
| `np.sort(a)` / `np.unique(a)` | `a.Sort()` / `pd.Unique(a)` |
| `np.where(m, a, b)` | `pd.WhereArray(m, a, b)` |

Full guide: [docs/numpy_translation_guide.md](docs/numpy_translation_guide.md)

## DataFrame examples

```go
df, _ := pd.DataFrameFromRecords([]map[string]any{
    {"country": "AR", "name": "Ana", "age": 30, "salary": 1000.0},
    {"country": "AR", "name": "Luis", "age": 40, "salary": 2000.0},
    {"country": "BR", "name": "Joao", "age": 35, "salary": 1500.0},
}, pd.WithColumnOrder("country", "name", "age", "salary"))

adults, _ := df.Query(`age > 30 and country in ["AR", "BR"]`)
top, _ := adults.SortValues("salary", false)
fmt.Println(top.Head(5))

stats, _ := df.Corr()          // correlation matrix
clean := df.DropNA()           // missing-data handling
byType, _ := df.SelectDTypes(pd.Include(pd.Number))
```

## Series examples

```go
s := pd.SeriesOf("v", []int{3, 1, 4, 1, 5})
ranks, _ := s.Rank(pd.RankMethod("dense"))
change, _ := s.PctChange(1)
running, _ := s.Cumsum()
counts := s.ValueCounts()
```

## NumPy-like NDArray examples

```go
m, _ := pd.Arange(6).Reshape(2, 3)
c, _ := m.Add(pd.Array([]float64{10, 20, 30})) // broadcasting
view, _ := m.Slice(pd.All(), pd.Slice(1, 3))   // views share data
norm := m.SubScalar(m.MeanAll()).DivScalar(m.StdAll())
tr, _ := norm.T()
prod, _ := pd.MatMul(m, tr)
_ = c; _ = view; _ = prod
```

## CSV / JSON

```go
df, _ := pd.ReadCSV("people.csv",
    pd.WithUseCols("name", "age"),
    pd.WithNRows(1000),
    pd.WithParseDates("joined"),
    pd.WithNAValues("-"), pd.WithKeepDefaultNA(true))
_ = df.ToJSON("out.json", pd.JSONOrient("split")) // records/split/columns/values
```

## GroupBy

```go
out, _ := df.GroupBy("country", "dept").AggList(map[string][]string{
    "salary": {"mean", "max"},
    "age":    {"min"},
}) // columns: country, dept, age_min, salary_mean, salary_max
```

## Merge / Join / Concat

```go
merged, _ := pd.Merge(left, right, pd.MergeOptions{
    On: []string{"id"}, How: "outer",
    Validate: "one_to_one", Indicator: true,
})
stacked, _ := pd.Concat(frames, pd.IgnoreIndex(true), pd.Join("inner"))
```

## Missing values

`nil`, `NaN`, `pd.NA()` and `pd.NaT()` are missing; reductions skip them;
comparisons with them are false; they sort last and print as `<NA>`.
Details: [docs/missing_values.md](docs/missing_values.md)

## DTypes

`pd.ParseDType("datetime64[ns]")`, `s.Astype(pd.Float64)`,
`df.Astype(map[string]pd.DType{...})`, `df.SelectDTypes(pd.Include(pd.Number))`.
Details: [docs/dtype_semantics.md](docs/dtype_semantics.md)

## loc / iloc

```go
df.ILoc().Rows(0, 2, pd.Slice(4, 8)).Cols(pd.Slice(1, 3)).Get() // positional, [start:stop)
df.Loc().Rows(pd.LabelSlice("a", "d")).Cols("name").Get()       // labels, inclusive
```

## Rolling windows

```go
ma, _ := prices.Rolling(20, pd.MinPeriods(1)).Mean()
vol, _ := prices.Rolling(20).Std()
cum, _ := prices.Expanding().Max()
```

## Compatibility matrix

- [compat/pandas_matrix.md](compat/pandas_matrix.md)
- [compat/numpy_matrix.md](compat/numpy_matrix.md)
- [compat/coverage_report.md](compat/coverage_report.md)

## Known differences

Operator syntax, NA comparison rules, positional-vs-label slicing,
alignment, dtype simplifications and more:
[compat/known_differences.md](compat/known_differences.md)

## Performance

Columnar storage, hash joins/grouping, stride-based zero-copy
broadcasting. Benchmarks and known bottlenecks:
[docs/performance.md](docs/performance.md)

```bash
go test ./benchmarks/ -bench=. -benchmem
```

## Roadmap

[docs/roadmap.md](docs/roadmap.md) — v0.3 pandas depth (MultiIndex,
categorical, resample), v0.4 performance backends (typed storage, Arrow,
gonum), v1.0 stable API.

## Development

```bash
go test ./...            # unit + golden tests (no Python needed)
go test ./... -race
go test ./fuzz/ -fuzz=FuzzReadCSV -fuzztime=30s
go run ./examples/basic  # and 7 more example programs
```

## Compatibility testing

The compatibility suite uses committed golden files generated from real
pandas and NumPy — currently **pandas 2.3.3 and NumPy 2.0.2** (the
versions are recorded inside each golden file). Normal Go tests do not
require Python:

```bash
go test ./...          # includes 200+ golden cases
```

Use `make regen-goldens` or `make compat` to regenerate expected behavior
(requires python3 with pandas and numpy):

```bash
make regen-goldens
# or regenerate + verify in one step:
python3 compat/python/run_compat_suite.py
```

## Reporting incompatibilities

Found go-pandas behaving differently from pandas/NumPy in a way not
listed in [compat/known_differences.md](compat/known_differences.md)?
Open an issue with the Python snippet, its output, and the Go
translation — golden cases are added from exactly that shape of report.

License: Apache 2.0.
