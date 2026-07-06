# Known differences from pandas/NumPy

## Syntax

Python supports operator overloading and bracket indexing. Go does not.
Every bracket/operator idiom has a named-method equivalent:

```python
df[df["age"] > 30]
```

```go
df.Where(pd.Col("age").Gt(30))
```

Errors are returned, not raised: most operations return `(result, error)`.

## Names claimed by NumPy vs the expression API

`pd.Abs`, `pd.Sqrt`, `pd.Exp`, `pd.Log` operate on `*NDArray` (NumPy
root-function parity). Inside expressions use the `*Expr` suffix:
`pd.AbsExpr(pd.Col("x"))`. Likewise `pd.Where` builds an expression;
`pd.WhereArray`/`pd.WhereScalar` are the np.where forms.

## Missing values

- `nil`, `math.NaN()`, `pd.NA()` and `pd.NaT()` are all missing; the
  empty string is not (matches pandas in-memory semantics).
- Comparisons against NA are false — including `Ne`. pandas returns True
  for `NaN != x`; go-pandas treats every NA comparison uniformly as false.
- There is one missing representation per Series (a mask), not a
  float-NaN-vs-NA distinction. `s.FillNA` therefore never distinguishes
  NaN from NA.

## DTypes

- The dtype system is an enum, not extension objects. `datetime64[ns]`,
  `datetime64[us]`, ... all parse to one `datetime64` dtype backed by
  `time.Time` (nanosecond precision, no fixed unit).
- Every Series is nullable via its mask; pandas' "Int64" vs "int64"
  distinction collapses to `int64` + mask.
- v0.2 NDArrays store float64 physically. Typed constructors and
  `Astype` record the logical dtype and normalize values, but large
  int64 values above 2^53 lose precision.
- Map-based DataFrame constructors order columns alphabetically unless
  `pd.WithColumnOrder` is given (Go maps are unordered; pandas preserves
  dict insertion order).

## Indexing and slicing

- Positional slicing (`pd.Slice`, iloc) is Go-style `[start, stop)`.
- Label slicing (`pd.LabelSlice`, `Loc().RowsBetween`) is **inclusive**
  on both ends, matching pandas `.loc["a":"z"]`.
- `iloc` with negative steps is not implemented (returns
  `ErrNotImplemented`).
- Unknown labels return `ErrInvalidIndex`; out-of-range positions return
  `ErrIndexOutOfBounds`.

## Merge

- With `LeftOn`/`RightOn`, pandas keeps both key columns; go-pandas keeps
  only the left key column (the values are equal on matches).
- Outer merges append left-only rows in left order and right-only rows
  after them; pandas sorts outer join keys. For sorted inputs the results
  coincide (the goldens verify this).

## GroupBy / aggregation naming

`Agg`/`AggList` name output columns `column_agg` (`salary_mean`). The
column order is: group keys, then aggregations sorted by source column
name (pandas follows keyword order in named aggregation).

## Rounding

`Round` uses banker's rounding (half to even) on both Series and
NDArrays, matching np.round / pandas — not Go's `math.Round`.

## Arithmetic alignment

Series arithmetic aligns **by position**, not by index labels. Use
`s.Reindex` to align explicitly first. (pandas aligns on labels and
produces NaN for non-overlapping labels.)

## Statistics defaults

- Series/DataFrame `Std`/`Var`: ddof=1 (pandas default).
- NDArray `Std`/`Var`: ddof=0 (NumPy default); `StdDDof`/`VarDDof` for
  explicit control.
- `Corr`/`Cov` use pairwise-complete observations, ddof=1.

## Timezones

`time.Time` values keep whatever location they carry; there is no
tz-aware dtype, `tz_localize` or `tz_convert`.

## Categorical

`category` parses as a dtype name but has no dedicated storage or
accessor yet (v0.3).

## Random

`pd.Rand`/`pd.Randn` match NumPy distributions and shapes, not values —
the underlying generators differ. Golden tests check properties only.

## Series results that return Series

`Series.ValueCounts` and `Series.Describe` return a `*Series` (pandas
`value_counts` does too; `describe` returns a Series for Series input).
`Series.ResetIndex` returns a `*Series` without inserting a label column;
`DataFrame.ResetIndex` does insert one, like pandas.

## Resampler / Stack / Unstack

`DataFrame.Resample`, `Stack` and `Unstack` return `ErrNotImplemented`.
