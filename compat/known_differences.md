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
- Every Series is nullable via its mask. Missing integers therefore
  behave like pandas' **nullable** `Int64` dtype, not the classic
  float64 coercion: `pd.Series([1, None, 3])` is float64 in pandas but
  an int column + mask here (golden-tested against `dtype="Int64"`).
- **v0.3 storage is typed**: NDArray backs onto `[]bool`, `[]int`,
  `[]int64`, `[]float32`, `[]float64` or `[]string`; Series/DataFrame
  columns back onto typed columns for bool/int/int64/float32/float64/
  string/time data. Object-backed `[]any` storage remains only for mixed
  or unsupported values (`s.IsObjectBacked()` / `StorageDType()` tell
  you which). Complex numbers remain object-backed. Categorical data has
  typed storage since v0.7 through int32 codes plus a shared category
  list (see the Categorical section below).
- `NDArray.Astype` and `Series.Astype` convert real storage. Float to
  integer truncates toward zero; string sources parse (string→int goes
  through float parsing, so "2.5" truncates rather than erroring like
  pandas); bool targets store `v != 0`.
- Arithmetic promotes dtypes NumPy-style (int+int→int, int+float→float64,
  float32+float32→float32, any int+float32→float64, bool arithmetic→int,
  int/int→float64 true division). `Pow` on integers computes in floating
  point and truncates — negative integer exponents differ from NumPy
  (which raises).
- `Arange` always returns Float64 (NumPy returns int64 for integer
  arguments). `Zeros`/`Ones`/`Full`/`Linspace`/random are Float64, like
  NumPy defaults.
- `a.Data()` returns values converted to `[]float64` (aliasing the
  backing only for contiguous Float64 arrays) and returns nil for string
  arrays — use `Values()`, `ValueAt` or `RawData()`. Numeric ufunc
  methods (`Sqrt`, `Exp`, ... — the error-free NumPy-shaped API) panic
  with an ErrTypeMismatch message on string arrays.
- `Series.Set` is type-checked since v0.3: storing an incompatible value
  into a typed column returns ErrTypeMismatch (the old boxed storage
  accepted anything). `FillNA` with an incompatible fill value rebuilds
  the series as object-backed instead.
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

## MultiIndex (v0.8)

Real levels + codes storage with pandas-parity sorted levels and NA
components as code -1. Documented differences and limits:

- **GroupBy defaults to as_index=false** (keys stay columns); pandas
  defaults to `as_index=True`. Opt in with `AsIndex(true)` /
  `pd.GroupAsIndex(true)`.
- **Partial selection** is prefix-only: `Loc().TuplePrefix(...)` covers
  `df.loc[(a, slice(None))]` and scans the code arrays (v0.8); general
  label-range slicing (`MultiIndex.Slice`) returns ErrNotImplemented.
- **NA tuple components**: nil components in `Loc().Tuple` match NA
  index components (pandas NaN labels are largely unmatchable);
  `Loc().Tuple` errors on unknown tuples like `Rows` does on labels.
- **Concat** stacks MultiIndexes with the same level count (names from
  the first frame, levels re-factorized); mixed index shapes fall back
  to a boxed index of tuples.
- **Merge/join**: join BY index aligns MultiIndexes through boxed tuple
  keys (works, no typed fast path); merge ON index levels is not
  supported — use key columns.
- **Levels are not compacted after Take/filtering**: codes may
  reference a level subset (pandas keeps unused levels too unless
  remove_unused_levels is called).
- **Mixed-family level labels** keep first-appearance order instead of
  pandas' object-dtype sort; single-family levels sort like pandas.
- `Series` support for MultiIndex is display/Take-level only.

## Gather semantics (v0.4.1)

`DataFrame.Take/Slice` and `Series.Take/Slice` return **copies**, never
views (unlike NDArray slicing, which is documented as views). Filtering
a default RangeIndex produces an `Int64Index` (or a `RangeIndex` when
the selected labels keep a constant step) — labels compare and print
exactly as before. Take with negative positions (outer-join fills)
falls back to a boxed index with missing labels.

## Merge

- **NA merge keys never match** — not even each other. pandas pairs NaN
  join keys together (`NaN == NaN` in merge); go-pandas treats a masked
  key as unknown, so NA-key rows only appear as left_only/right_only in
  left/right/outer joins. Unit tests lock this behavior.
- With `LeftOn`/`RightOn`, pandas keeps both key columns; go-pandas keeps
  only the left key column (the values are equal on matches).
- Outer merges append left-only rows in left order and right-only rows
  after them; pandas sorts outer join keys. For sorted inputs the results
  coincide (the goldens verify this).
- Numeric key widths match across frames (int 1 == 1.0); time keys
  compare by Go time.Time equality (wall clock + location). Duplicate
  keys expand deterministically: probe order, then build-side row order.
- Join BY a MultiIndex works since v0.8 through boxed tuple-key
  alignment (no typed fast path); merge ON MultiIndex levels is not
  supported — use key columns.
- Since v0.6 the engine is typed (docs/merge_engine.md); object-backed
  keys keep the historical `%v` matching.

## Concat (v0.6.1 typed engine)

- Column order: first frame's columns, then new columns in encounter
  order; `join="inner"` keeps the intersection.
- Compatible numeric columns promote (int+float64→float64, ...); a
  string+numeric or time+string column falls back to object storage —
  only that column. pandas coerces some of these differently (e.g. to
  object with original values, which matches, but pandas never has a
  "typed vs object" storage distinction to preserve).
- axis=1 requires equal row counts and aligns positionally — no label
  alignment (pandas aligns on the index). Duplicate names get _1/_2
  suffixes (pandas keeps duplicates).
- Preserved (non-ignored) indexes concatenate typed since v0.6.1:
  integer label families produce an Int64Index (previously labels were
  stringified into a StringIndex).

## GroupBy / aggregation naming

`Agg`/`AggList` name output columns `column_agg` (`salary_mean`). The
column order is: group keys, then aggregations sorted by source column
name (pandas follows keyword order in named aggregation).

Since v0.5 grouping runs on the typed engine (docs/groupby_engine.md):
numeric key widths group together (1 == 1.0), object-backed keys keep
the historical `%v` grouping, and with sorting enabled the NA-key group
(dropna=false) sorts **last**, matching pandas — before v0.5 it kept its
first-seen position. `std`/`var` remain ddof=1; `median` returns
float64; bool value columns aggregate as 0/1 numerics.

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

## Categorical (v0.7)

`pd.Category` has real typed storage (int32 codes + shared category
list) with pandas semantics: sorted default categories, strict explicit
categories, rank-based ordered comparisons, observed groupby, label-only
writers. Documented differences:

- **Concat with differing categories** stays categorical with the union
  of the category lists (first-seen order), like
  `pd.api.types.union_categoricals`; pandas' plain `concat` downgrades
  to object. Identical ordered lists stay ordered; otherwise the result
  is unordered.
- **GroupBy emits observed groups only** (pandas `observed=True`);
  unused categories do not appear as empty groups.
- **`Series.Gt/Ge/Lt/Le` on unordered categoricals** return all-false
  (there is no error channel on those methods); the `Cat()` accessor
  comparisons and the expression engine (`Where`/`Query`) surface
  `ErrInvalidOperation` instead, matching pandas' TypeError.
- Categories are never inferred automatically — conversion requires
  `Astype(pd.Category)`, a categorical constructor, or the
  `pd.WithCategorical` CSV option.
- **Implicit categories require one label family** (numeric, string,
  bool or time.Time) so the sorted default order is total; mixed
  families return `ErrTypeMismatch` (v0.7.1). pandas would build an
  object-dtype category list with Python's cross-type ordering quirks.
  Explicit categories (`pd.WithCategories`) accept mixed hashable
  labels because the order is user-provided.

## Random

`pd.Rand`/`pd.Randn` match NumPy distributions and shapes, not values —
the underlying generators differ. Golden tests check properties only.

## Columnar expression engine (v0.4)

`Where`/`AssignExpr`/`Query` execute over typed column buffers when
possible and fall back to per-row evaluation otherwise — results are
identical by construction (equivalence-tested). One nuance: predicate
masks are three-valued internally (Kleene `And`/`Or`/`Not` over NA), but
filters drop NA rows and predicate *assignment* stores NA as `false`,
so the observable behavior matches both the row evaluator and pandas'
classic bool arrays. `pd.DebugPlan(df, expr)` reports the chosen path.

## Series results that return Series

`Series.ValueCounts` and `Series.Describe` return a `*Series` (pandas
`value_counts` does too; `describe` returns a Series for Series input).
`Series.ResetIndex` returns a `*Series` without inserting a label column;
`DataFrame.ResetIndex` does insert one, like pandas.

## Resampler / Stack / Unstack

`DataFrame.Resample`, `Stack` and `Unstack` return `ErrNotImplemented`.
