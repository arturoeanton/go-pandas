<p align="center">
  <img src="logo.png" alt="go-pandas" width="240"/>
</p>

# go-pandas

**Pandas and NumPy style data analysis for Go.**

> If you know pandas and NumPy, the concepts should transfer immediately.

go-pandas is **experimental**. It aims for pandas/NumPy *conceptual*
compatibility, not Python syntax compatibility: every important pandas or
NumPy concept has one obvious Go equivalent, and unsupported APIs return
explicit `ErrNotImplemented` errors instead of fake implementations.

```bash
go get github.com/arturoeanton/go-pandas
```

```go
import pd "github.com/arturoeanton/go-pandas"
```

## Pandas → Go translation

Go cannot express `df["age"]` or `df[df.age > 30]`, so go-pandas gives each
idiom a direct equivalent:

| pandas | go-pandas |
|---|---|
| `pd.DataFrame(records)` | `pd.DataFrameFromRecords(records)` |
| `df["age"]` | `df.Col("age")` |
| `df[["name", "age"]]` | `df.Select("name", "age")` |
| `df[df["age"] > 30]` | `df.Where(pd.Col("age").Gt(30))` |
| `df.query("age > 30 and c == 'AR'")` | `df.Query("age > 30 and c == \"AR\"")` |
| `df.loc[:, ["name"]]` | `df.Loc().Cols("name").Get()` |
| `df.iloc[0:10, 1:3]` | `df.ILoc().Rows(pd.Slice(0, 10)).ColsRange(pd.Slice(1, 3)).Get()` |
| `df["total"] = df.price * df.qty` | `df.AssignExpr("total", pd.Col("price").Mul(pd.Col("qty")))` |
| `df.groupby("c")["v"].mean()` | `df.GroupBy("c").Mean("v")` |
| `df.merge(right, on="id")` | `df.Merge(right, pd.MergeOptions{On: []string{"id"}})` |
| `pd.concat([a, b])` | `pd.Concat([]*pd.DataFrame{a, b})` |
| `df.sort_values("v", ascending=False)` | `df.SortValues("v", false)` |
| `df.dropna()` / `df.fillna(...)` | `df.DropNA()` / `df.FillNA(map[string]any{...})` |
| `pd.read_csv(path)` | `pd.ReadCSV(path)` |
| `s.str.upper()` / `s.dt.year` | `s.Str().Upper()` / `s.Dt().Year()` |

Full matrix: [compat/pandas_matrix.md](compat/pandas_matrix.md)

## NumPy → Go translation

| NumPy | go-pandas |
|---|---|
| `np.array([1, 2, 3])` | `pd.Array([]float64{1, 2, 3})` |
| `np.arange(6).reshape(2, 3)` | `pd.Arange(6).Reshape(2, 3)` |
| `a + 10` | `a.AddScalar(10)` |
| `a + b` (broadcasting) | `a.Add(b)` |
| `a[0:2, 1:3]` | `a.Slice(pd.Slice(0, 2), pd.Slice(1, 3))` |
| `a.T` | `a.T()` |
| `a.sum(axis=0)` | `a.Sum(0)` |
| `np.matmul(a, b)` | `pd.MatMul(a, b)` |
| `np.where(cond, x, y)` | `ndarray.Where(cond, x, y)` |
| `np.random.randn(2, 3)` | `pd.Randn(2, 3)` |

Full matrix: [compat/numpy_matrix.md](compat/numpy_matrix.md)

## DataFrame

```go
df, _ := pd.DataFrameFromRecords([]map[string]any{
    {"country": "AR", "name": "Ana", "age": 30, "salary": 1000.0},
    {"country": "AR", "name": "Luis", "age": 40, "salary": 2000.0},
    {"country": "BR", "name": "Joao", "age": 35, "salary": 1500.0},
}, pd.WithColumnOrder("country", "name", "age", "salary"))

result, _ := df.Where(pd.Col("age").Gt(30))
result, _ = result.Select("country", "name", "salary")
result, _ = result.SortValues("salary", false)
fmt.Println(result)
//    country  name  salary
// 1  AR       Luis  2000
// 2  BR       Joao  1500
//
// [2 rows x 3 columns]
```

## Series

```go
s := pd.SeriesOf("age", []int{10, 20, 30})
mean, _ := s.Mean()            // 20
mask := s.Gt(15)               // Bool series
clean := s.FillNA(0).DropNA()  // missing-data helpers
```

Missing values follow the pandas model: `nil`, `pd.NA()`, `pd.NaT()` and
`NaN` are all missing; reductions skip them by default (`pd.SkipNA(false)`
to opt out); comparisons with missing values are `false`; `<NA>` prints in
tables.

## NDArray

```go
a, _ := pd.FromSlice([]float64{1, 2, 3, 4, 5, 6}, 2, 3)
b := pd.Array([]float64{10, 20, 30})
c, _ := a.Add(b) // NumPy broadcasting: (2,3) + (3,) -> (2,3)
fmt.Println(c)
// array([[11, 22, 33],
//        [14, 25, 36]])
```

Slicing, `Reshape` and `Transpose` return **views** backed by strides —
mutate a view and the base array changes; call `Copy()` for independence.
Broadcasting uses stride-0 views, never materialized copies.

## CSV / JSON

```go
df, _ := pd.ReadCSV("people.csv",
    pd.WithParseDates("joined"),
    pd.WithNAValues("", "NA", "null"))
_ = df.ToCSV("out.csv")
_ = df.ToJSON("out.json")   // records orientation
df2, _ := pd.ReadNDJSON("events.ndjson")
```

## GroupBy

```go
grouped, _ := df.GroupBy("country").Agg(map[string]string{
    "salary": "mean",
    "age":    "max",
})
//    country  age_max  salary_mean
// 0  AR       40       1500
// 1  BR       35       1500
```

## Merge

```go
merged, _ := left.Merge(right, pd.MergeOptions{
    On:  []string{"id"},
    How: "outer", // inner, left, right, outer, cross
})
```

## Rolling windows

```go
ma, _ := prices.Rolling(3, pd.RollingMinPeriods(1)).Mean()
sums, _ := df.Rolling(2).Sum() // per numeric column
```

## Examples

```bash
go run ./examples/basic
go run ./examples/pandas_compat
go run ./examples/numpy_compat
go run ./examples/groupby
go run ./examples/merge
go run ./examples/io_csv
go run ./examples/ndarray
go run ./examples/rolling
```

## Compatibility testing

Core behavior is verified against golden outputs generated from real
pandas/NumPy (`compat/goldens/*.json`, regenerate with
`make regen-goldens`; Python is not needed for `go test ./...`). Matrices
documenting per-API status:

- [compat/pandas_matrix.md](compat/pandas_matrix.md)
- [compat/numpy_matrix.md](compat/numpy_matrix.md)

## Roadmap

- **v0.1** (this release): Series, DataFrame, float64 NDArray, indexes,
  missing values, expressions, groupby, merge/join/concat, reshape basics,
  rolling, CSV/JSON/NDJSON, golden tests.
- **v0.2**: typed NDArrays (int/bool), more axis reductions and ufuncs,
  sort/search, Gonum linalg adapter.
- **v0.3**: better MultiIndex, categorical dtype, timezone-aware
  datetimes, resampling, pivot_table, stronger query parser.
- **v0.4**: Arrow/DuckDB/Parquet adapters, typed column storage, SIMD.
- **v0.5**: Excel/SQL IO, time-based rolling, ewm, nullable typed dtypes.
- **v1.0**: stable API and production-ready performance.

## Performance notes

v0.1 is correctness-first, but the architecture avoids the classic traps:
columnar storage with stable column order, hash-based groupby and merge,
stride-based broadcasting and views without copies, no reflection in hot
loops (reflection is used only in `DataFrameFromStructs`). Series values
are stored as `[]any` in v0.1; typed column storage is planned for v0.4.

## Limitations

- NDArray stores float64 only (comparisons yield `*BoolArray`).
- Series/DataFrame arithmetic aligns by position, not by index labels.
- `Stack`, `Unstack` and `Resample` return `ErrNotImplemented`.
- `MultiIndex` supports construction and display only.
- No timezone handling beyond what `time.Time` carries.
- Map-based constructors order columns alphabetically unless
  `pd.WithColumnOrder` is given (Go maps are unordered).
- `Series.ValueCounts`/`Series.Describe` return a `Series` (pandas
  returns a Series from `value_counts` too; the DataFrame form would
  create an import cycle between the series and dataframe packages).

License: Apache 2.0.
