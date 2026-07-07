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
real pandas and NumPy** (301 test cases, pandas 2.3 / NumPy 2.0).

Since v0.10, the **reshape surface is closed for common workflows**:
`Stack`/`Unstack` over MultiIndex, `PivotTable` with multiple values
and aggfuncs, `GroupBy` `Transform`/`Filter`, and a Query grammar with
arithmetic, `in`/`not in` and datetime comparisons.
Since v0.9, the common **time-series workflow works end to end**:
format-aware `pd.ToDatetime` (strftime directives, raise/coerce), a
real `DatetimeIndex` from `SetIndex`, and `df.Resample("D").Sum()` over
observed buckets (H/D/W/MS/ME; a 100K-row daily sum takes ~2.6 ms).
Since v0.8, the **MultiIndex is real**: levels + int32 codes like
pandas, with multi-column `SetIndex`, `ResetIndex`, tuple-based `Loc`
(full-tuple lookups in ~100 ns at 100K rows), optional
`GroupBy(...).AsIndex(true)` output, and index preservation through
every engine. Since v0.7, `pd.Category` is a **typed categorical dtype**: int32 codes
into a shared category list, with code fast paths everywhere — on an
Apple M4 with 500K rows over 8 distinct labels, sorting a categorical is
~91x faster than strings, value_counts ~134x, group-means ~3.4x, and the
storage is ~3.4x smaller (gains depend on label repetition; see
docs/categorical.md). Since v0.6, `Merge`/`Join` run on a **typed hash-join engine** (a 100K
x 10K int-key merge takes ~2 ms with ~180 allocations, ~8x faster than
v0.5). Since v0.5, `GroupBy` runs on a **typed engine** (typed key maps +
segment reducers — a 100K-row group-mean takes ~0.9 ms with 70
allocations, ~10x faster than v0.4). Since v0.4, `Where`/`AssignExpr`/`Query` run on a **columnar expression
engine**: typed kernels over column buffers instead of a map per row
(a 100K-row numeric filter runs in ~0.9 ms with 24 allocations —
~20x faster than the row-map path; `pd.DebugPlan` shows the chosen path). Since v0.3 storage is **typed**: `pd.ArrayInt` really stores `[]int`,
`pd.SeriesOf("x", []string{...})` really stores `[]string`, DataFrame
columns and CSV parsing infer typed columns, and arithmetic promotes
dtypes NumPy-style (`int + float64 → float64`). Mixed data falls back to
object storage — `StorageDType()` / `IsObjectBacked()` tell you which.

## Release candidate status

go-pandas is preparing for **v1.0**. The API freeze audit is documented
in [docs/api_freeze.md](docs/api_freeze.md), the v1.0 plan and policies
in [docs/v1_plan.md](docs/v1_plan.md), and known differences from
pandas/NumPy are treated as **part of the compatibility contract**.
The API is still experimental but frozen in all but name — the stability of every public group is classified in
[docs/api_freeze.md](docs/api_freeze.md). The core DataFrame, Series,
NDArray, GroupBy, Merge, Concat, Categorical, MultiIndex and
time-series paths are golden-tested against real pandas/NumPy outputs,
and known differences are documented, never hidden. Compatibility is
conceptual and behavioral where tested, not Python syntax
compatibility. Current coverage, computed from the matrices with
`go run ./cmd/compat-report`: pandas 94% of 136 tracked rows, NumPy
91% of 54 tracked rows — including partial rows
([full report](compat/coverage_report.md), [what's intentionally
different](compat/known_differences.md), [prerelease
status](docs/prerelease.md)).

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
| `s.astype("category")` / `s.cat.codes` | `s.Astype(pd.Category)` / `cat, _ := s.Cat(); cat.Codes()` |
| `df.set_index(["c1","c2"])` | `df.SetIndex("c1", "c2")` |
| `df.loc[("AR", "BA")]` | `df.Loc().Tuple("AR", "BA").Get()` |
| `df.groupby(["a","b"], as_index=True)` | `df.GroupBy("a", "b").AsIndex(true)` |
| `pd.to_datetime(s, format="%Y-%m-%d")` | `pd.ToDatetime(s, pd.WithDatetimeFormat("%Y-%m-%d"))` |
| `df.resample("D").sum()` | `df.Resample("D").Sum()` |
| `df.stack()` / `s.unstack()` | `df.Stack()` / `pd.UnstackSeries(s)` |
| `df.groupby("k")["v"].transform("mean")` | `df.GroupBy("k").Transform("v", "mean")` |
| `df.groupby("k").filter(lambda g: len(g) > 2)` | `df.GroupBy("k").Filter(pd.GroupSize().Gt(2))` |

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

fmt.Println(pd.DebugPlan(df, pd.Col("age").Gt(30))) // "columnar: ..."
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

// Categorical (v0.7): int32 codes, ordered rank comparisons
size, _ := pd.CategoricalSeries("size", []string{"m", "s", "l"},
    pd.WithCategories("s", "m", "l"), pd.WithOrdered(true))
big := size.Gt("s")            // by category rank: [true false true]
cat, _ := size.Cat()           // .Categories() .Codes() .Rename...
_, _ = big, cat
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

// typed storage + promotion (v0.3)
ints := pd.ArrayInt([]int{1, 2, 3})     // RawData() is []int
sum, _ := ints.Add(pd.Array([]float64{0.5, 0.5, 0.5}))
_ = sum.DType()                         // pd.Float64
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
one, _ := pd.ConcatSeries(s1, s2) // typed append + promotion (v0.6.1)
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

[docs/roadmap.md](docs/roadmap.md) — pandas depth next (stack/unstack,
timezones, resample options), then performance backends (Arrow, gonum,
SIMD), v1.0 stable API.

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
